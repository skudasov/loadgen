package loadgen

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"

	"go.uber.org/ratelimit"
)

// BeforeRunner can be implemented by an Attacker
// and its method is called before a test or Run.
type BeforeRunner interface {
	BeforeRun(c RunnerConfig) error
}

// AfterRunner can be implemented by an Attacker
// and its method is called after a test or Run.
// The report is passed to compute the Failed field and/or store values in Output.
type AfterRunner interface {
	AfterRun(r *RunReport) error
}

type RuntimeCheckFunc func(r *Runner) bool

const (
	rampUp int = iota
	constantLoad
)

// Default runner runtime check types
const (
	prometheusCheckType = "prometheus"
	errorRatioCheckType = "error"
)

type Runner struct {
	name             string
	TestStage        int
	ReadCsvName      string
	WriteCsvName     string
	RecycleData      bool
	Manager          *LoadManager
	Config           RunnerConfig
	attackers        []Attack
	once             sync.Once
	failed           bool // if tests are failed for any reason
	running          bool
	stopped          bool // if tests are stopped by hook
	next, quit, stop chan bool
	results          chan result
	prototype        Attack
	resultsPipeline  func(r result) result

	// Checks whether to stop generator
	checkFunc RuntimeCheckFunc
	CheckData []Checks

	// Other clients for checks
	PromClient v1.API

	// Metrics
	registeredMetricsLabels []string
	RateLog                 []float64
	MaxRPS                  float64
	// RampUpMetrics store only rampup interval metrics, cleared every interval
	RampUpMetrics map[string]*Metrics
	// Metrics store full attack metrics
	Metrics               map[string]*Metrics
	timerMu               *sync.RWMutex
	timers                map[string]metrics.Timer
	errorsMu              *sync.RWMutex
	Errors                map[string]metrics.Counter
	goroutinesCountGaugue metrics.Gauge
	goroutinesCount       int64

	L *Logger
}

func NewRunner(name string, lm *LoadManager, a Attack, ch RuntimeCheckFunc, c RunnerConfig) *Runner {
	var promClient v1.API
	if lm.GeneratorConfig.Prometheus != nil {
		promC, err := api.NewClient(api.Config{
			Address: lm.GeneratorConfig.Prometheus.URL,
		})
		if err != nil {
			log.Fatalf("failed to setup prometheus client: %s", err)
		}
		promClient = v1.NewAPI(promC)
	}
	r := &Runner{
		name:      name,
		Manager:   lm,
		Config:    c,
		prototype: a,

		checkFunc: ch,
		CheckData: c.StopIf,

		PromClient: promClient,
		RateLog:    []float64{},

		once:      sync.Once{},
		next:      make(chan bool),
		quit:      make(chan bool),
		stop:      make(chan bool),
		results:   make(chan result),
		attackers: []Attack{},

		registeredMetricsLabels: make([]string, 0),
		RampUpMetrics:           make(map[string]*Metrics),
		Metrics:                 make(map[string]*Metrics),
		timerMu:                 &sync.RWMutex{},
		timers:                  make(map[string]metrics.Timer),
		errorsMu:                &sync.RWMutex{},
		Errors:                  make(map[string]metrics.Counter),
		goroutinesCount:         0,
		goroutinesCountGaugue:   metrics.NewGauge(),

		L: &Logger{log.With("runner", name)},
	}
	r.L.Infof("bootstraping generator")
	r.L.Infof("[%d] available logical CPUs", runtime.NumCPU())

	// validate the configuration
	if msg := c.Validate(); len(msg) > 0 {
		for _, each := range msg {
			fmt.Println("a configuration error was found", each)
		}
		fmt.Println()
		flag.Usage()
		os.Exit(0)
	}

	// is the attacker interested in the Run lifecycle?
	if lifecycler, ok := a.(BeforeRunner); ok {
		if err := lifecycler.BeforeRun(c); err != nil {
			log.Fatalf("BeforeRun failed: %s", err)
		}
	}

	// do a test if the flag says so
	if *oSample > 0 {
		r.test(*oSample)
		report := RunReport{}
		if lifecycler, ok := a.(AfterRunner); ok {
			if err := lifecycler.AfterRun(&report); err != nil {
				log.Fatalf("AfterRun failed: %s", err)
			}
		}
		os.Exit(0)
		// unreachable
		return r
	}
	return r
}

func (r *Runner) initPipeline() {
	r.resultsPipeline = r.addResult
}

func (r *Runner) spawnAttacker() {
	if r.Config.Verbose {
		r.L.Infof("setup and spawn new attacker [%d]", len(r.attackers)+1)
	}
	attacker := r.prototype.Clone(r)
	if err := attacker.Setup(r.Config); err != nil {
		r.L.Infof("attacker [%d] setup failed with [%v]", len(r.attackers)+1, err)
		return
	}
	r.attackers = append(r.attackers, attacker)
	go attack(attacker, r.next, r.quit, r.results, r.Config.timeout())
}

// addResult is called from a dedicated goroutine.
func (r *Runner) addResult(s result) result {
	m, ok := r.Metrics[s.doResult.RequestLabel]
	if !ok {
		m = new(Metrics)
		r.Metrics[s.doResult.RequestLabel] = m
	}
	m.add(s)
	return s
}

// test uses the Attack to perform {count} calls and report its result
// it is intended for development of an Attack implementation.
func (r *Runner) test(count int) {
	probe := r.prototype.Clone(r)
	if err := probe.Setup(r.Config); err != nil {
		log.Infof("test attack setup failed [%v]", err)
		return
	}
	defer probe.Teardown()
	for s := count; s > 0; s-- {
		now := time.Now()
		result := probe.Do(context.Background())
		log.Infof("test attack call [%s] took [%v] with status [%v] and error [%v]", result.RequestLabel, time.Now().Sub(now), result.StatusCode, result.Error)
	}
}

func (r *Runner) SetupHandleStore(m *LoadManager) {
	csvReadName := r.Config.ReadFromCsvName
	recycleData := r.Config.RecycleData
	if csvReadName != "" {
		log.Infof("creating read file: %s", csvReadName)
		f, err := os.Open(csvReadName)
		if err != nil {
			log.Fatalf("no csv read file found: %s", csvReadName)
		}
		m.CsvStore[csvReadName] = NewCSVData(f, recycleData)
	}
	csvWriteName := r.Config.WriteToCsvName
	if csvWriteName != "" {
		log.Infof("creating write file: %s", csvWriteName)
		csvFile := CreateOrReplaceFile(csvWriteName)
		m.CsvStore[csvWriteName] = NewCSVData(csvFile, false)
	}
}

// defaultCheckByData setups default prometheus or error ration check func
func (r *Runner) defaultCheckByData() {
	if r.checkFunc != nil {
		r.L.Info("custom check selected, see code in checks.go")
		return
	}
	if r.CheckData != nil {
		switch r.CheckData[0].Type {
		case prometheusCheckType:
			r.L.Infof("default prometheus check selected, query: %s", r.CheckData[0].Query)
			r.checkFunc = func(r *Runner) bool {
				return PromBooleanQuery(r)
			}
			return
		case errorRatioCheckType:
			r.L.Infof("default error check selected, threshold: %.2f perc errors", r.CheckData[0].Threshold)
			r.checkFunc = func(r *Runner) bool {
				return ErrorPercentCheck(r, r.CheckData[0].Threshold)
			}
			return
		default:
			r.L.Infof("unknown check type selected, skipping runner runtime check")
			return
		}
	}
	r.L.Info("no default check found")
}

// Run offers the complete flow of a test.
func (r *Runner) Run(wg *sync.WaitGroup, lm *LoadManager) {
	r.failed = false
	r.stopped = false
	r.resultsPipeline = r.addResult
	if wg != nil {
		defer wg.Done()
	}
	if lifecycler, ok := r.prototype.(BeforeRunner); ok {
		if err := lifecycler.BeforeRun(r.Config); err != nil {
			r.L.Infof("BeforeRun failed", err)
		}
	}
	r.collectResults()
	r.initMonitoring()

	if r.Config.WaitBeforeSec != 0 {
		r.L.Infof("awaiting runner start, sleeping for %d sec", r.Config.WaitBeforeSec)
		time.Sleep(time.Duration(r.Config.WaitBeforeSec) * time.Second)
	}
	r.defaultCheckByData()
	r.checkStopIf()
	r.running = true
	if r.rampUp() {
		r.fullAttack()
	}
	r.Shutdown()
	r.ReportMaxRPS()
	report := RunReport{}
	if lifecycler, ok := r.prototype.(AfterRunner); ok {
		if err := lifecycler.AfterRun(&report); err != nil {
			r.L.Infof("AfterRun failed", err)
		}
	}
	lm.CsvMu.Lock()
	defer lm.CsvMu.Unlock()
	rep := r.reportMetrics()
	lm.Reports[r.name] = rep
}

func (r *Runner) SetValidationParams() {
	r.RateLog = []float64{}
	r.Config.IsValidationRun = true
	r.Config.AttackTimeSec = r.Config.Validation.AttackTimeSec
	r.Config.RampUpTimeSec = 1
	r.Config.StoreData = false
	rpsWithNoErrors := int(r.Config.Validation.Threshold * r.MaxRPS)
	if rpsWithNoErrors == 0 {
		rpsWithNoErrors = 1
	}
	r.Config.RPS = rpsWithNoErrors
	r.L.Infof("running validation of max rps: %d for %d seconds", r.Config.RPS, r.Config.AttackTimeSec)
}

func (r *Runner) initMonitoring() {
	url := viper.GetString("graphite.url")
	if url == "" {
		return
	}
	flushDuration := time.Duration(viper.GetInt("graphite.flushDurationSec"))
	loadGeneratorPrefix := viper.GetString("graphite.loadGeneratorPrefix")

	StartGraphiteSender(loadGeneratorPrefix, flushDuration, url)
	r.registerMetric("goroutines-"+r.name, r.goroutinesCountGaugue)
}

func (r *Runner) registerLabelTimings(label string) metrics.Timer {
	r.timerMu.RLock()
	timer, ok := r.timers[label]
	r.timerMu.RUnlock()
	if ok {
		return timer
	}
	r.timerMu.Lock()
	defer r.timerMu.Unlock()
	timer = metrics.NewTimer()
	r.timers[label] = timer
	r.registerMetric(label+"-timer", timer)
	return timer
}

func (r *Runner) registerErrCount(label string) metrics.Counter {
	r.errorsMu.RLock()
	cnt, ok := r.Errors[label]
	r.errorsMu.RUnlock()
	if ok {
		return cnt
	}
	r.errorsMu.Lock()
	defer r.errorsMu.Unlock()
	cnt = metrics.NewCounter()
	r.Errors[label] = cnt
	r.registerMetric(label+"-err", cnt)
	return cnt
}

func (r *Runner) registerMetric(name string, metric interface{}) {
	r.registeredMetricsLabels = append(r.registeredMetricsLabels, name)
	if err := metrics.Register(name, metric); err != nil {
		log.Infof("failed to register metric: %s", err)
	}
}

func (r *Runner) fullAttack() {
	r.TestStage = constantLoad
	if r.Config.Verbose {
		r.L.Infof("begin full attack of [%d] remaining seconds", r.Config.AttackTimeSec-r.Config.RampUpTimeSec)
	}
	fullAttackStartedAt = time.Now()
	limiter := ratelimit.New(r.Config.RPS) // per second
	doneDeadline := time.Now().Add(time.Duration(r.Config.AttackTimeSec-r.Config.RampUpTimeSec) * time.Second)
	for time.Now().Before(doneDeadline) {
		limiter.Take()
		if !r.stopped {
			r.next <- true
		}
	}
	if r.Config.Verbose {
		r.L.Info("end full attack")
	}
}

func (r *Runner) rampUp() bool {
	r.TestStage = rampUp
	strategy := r.Config.rampupStrategy()
	if r.Config.Verbose {
		r.L.Infof("begin rampup of [%d] seconds to RPS [%d] within attack of [%d] seconds using strategy [%s]",
			r.Config.RampUpTimeSec,
			r.Config.RPS,
			r.Config.AttackTimeSec,
			strategy,
		)
	}
	var finished bool
	switch strategy {
	case "linear":
		finished = linearIncreasingGoroutinesAndRequestsPerSecondStrategy{}.execute(r)
	case "exp2":
		finished = spawnAsWeNeedStrategy{}.execute(r)
	}
	// restore pipeline function in case it was changed by the rampup strategy
	r.resultsPipeline = r.addResult
	if r.Config.Verbose {
		r.L.Infof("end rampup ending up with [%d] attackers", len(r.attackers))
	}
	return finished
}

func (r *Runner) quitAttackers() {
	if r.Config.Verbose {
		log.Infof("stopping attackers [%d]", len(r.attackers))
	}
	for range r.attackers {
		r.quit <- true
	}
}

func (r *Runner) tearDownAttackers() {
	if r.Config.Verbose {
		r.L.Infof("tearing down attackers [%d]", len(r.attackers))
	}
	for i, each := range r.attackers {
		if err := each.Teardown(); err != nil {
			r.L.Infof("failed to teardown attacker [%d]:%v", i, err)
		}
	}
}

func (r *Runner) unregisterMetrics() {
	for _, m := range r.registeredMetricsLabels {
		metrics.Unregister(m)
	}
}

func (r *Runner) reportMetrics() *RunReport {
	for _, each := range r.Metrics {
		each.updateLatencies()
	}
	return &RunReport{
		StartedAt:     fullAttackStartedAt,
		FinishedAt:    time.Now(),
		Configuration: r.Config,
		Metrics:       r.Metrics,
		Failed:        false, // must be overwritten by program
		Output:        map[string]interface{}{},
	}
}

func (r *Runner) collectResults() {
	go func() {
		for {
			r.resultsPipeline(<-r.results)
		}
	}()
}

func (r *Runner) ReportMaxRPS() {
	r.MaxRPS = MaxRPS(r.RateLog)
	r.L.Infof("max rps: %.2f", r.MaxRPS)
	if r.Config.IsValidationRun && !r.failed {
		entry := []string{r.name, os.Getenv("NETWORK_NODES"), fmt.Sprintf("%.2f", r.MaxRPS)}
		r.L.Infof("writing scaling info: %s", entry)
		if err := r.Manager.RPSScalingLog.Write(entry); err != nil {
			r.L.Fatal(err)
		}
	}
}

func (r *Runner) Shutdown() {
	r.once.Do(func() {
		if r.running {
			r.L.Infof("test ended, shutting down runner")
			r.stop <- true
			r.stopped = true
			r.running = false
			r.checkFunc = nil
			r.quitAttackers()
			r.tearDownAttackers()
			r.unregisterMetrics()
		}
	})
}

// checkStopIf executing check function, shutdown if it returns true
func (r *Runner) checkStopIf() {
	if r.checkFunc == nil {
		return
	}
	go func() {
		for {
			select {
			case <-r.stop:
				return
			default:
				checkTime := time.Duration(r.CheckData[0].Interval)
				time.Sleep(checkTime * time.Second)
				if r.checkFunc(r) {
					r.L.Infof("runtime check failed, exiting")
					r.failed = true
					r.Manager.Failed = true
					if r.Config.IsValidationRun {
						r.Manager.ValidationFailed = true
					}
					r.Shutdown()
					return
				}
			}
		}
	}()
}
