package loadgen

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/insolar/x-crypto/ecdsa"
	"github.com/spf13/viper"
)

const (
	ReportFileTmpl       = "%s-%d.json"
	ParallelMode         = "parallel"
	SequenceMode         = "sequence"
	SequenceValidateMode = "sequence_validate"
)

var (
	handleReportRe = regexp.MustCompile(`-(\d+).json`)
	sigs           = make(chan os.Signal, 1)
)

// LoadManager manages data and finish criteria
type LoadManager struct {
	RootMemberPrivateKey *ecdsa.PrivateKey
	RootMemberPublicKey  *ecdsa.PublicKey
	// SuiteConfig holds data common for all groups
	SuiteConfig *SuiteConfig
	// GeneratorConfig holds generator data
	GeneratorConfig *DefaultGeneratorConfig
	// Steps runner objects that fires .Do()
	Steps []RunStep
	// AttackerConfigs attacker configs
	AttackerConfigs map[string]RunnerConfig
	// Reports run reports for every handle
	Reports map[string]*RunReport
	// CsvStore stores data for all attackers
	CsvMu    *sync.Mutex
	CsvStore map[string]*CSVData
	// all handles csv logs
	CSVLogMu      *sync.Mutex
	CSVLog        *csv.Writer
	RPSScalingLog *csv.Writer
	ReportDir     string
	// When degradation threshold is reached for any handle, see default Config
	Degradation bool
	// When there are Errors in any handle
	Failed bool
	// When max rps validation failed
	ValidationFailed bool
}

type RunStep struct {
	Name          string
	ExecutionMode string
	Runners       []*Runner
}

// NewLoadManager create example_loadtest manager with data files
func NewLoadManager(suiteCfg *SuiteConfig, genCfg *DefaultGeneratorConfig) *LoadManager {
	var err error
	csvLog := csv.NewWriter(createFileOrAppend("result.csv"))
	scalingLog := csv.NewWriter(createFileOrAppend("scaling.csv"))

	lm := &LoadManager{
		SuiteConfig:     suiteCfg,
		GeneratorConfig: genCfg,
		CsvMu:           &sync.Mutex{},
		CSVLogMu:        &sync.Mutex{},
		CSVLog:          csvLog,
		RPSScalingLog:   scalingLog,
		Steps:           make([]RunStep, 0),
		Reports:         make(map[string]*RunReport),
		CsvStore:        make(map[string]*CSVData),
		Degradation:     false,
	}
	if lm.ReportDir, err = filepath.Abs(filepath.Join("example_loadtest", "reports")); err != nil {
		log.Fatal(err)
	}
	return lm
}

func (m *LoadManager) SetupHandleStore(handle RunnerConfig) {
	csvReadName := handle.ReadFromCsvName
	recycleData := handle.RecycleData
	if csvReadName != "" {
		log.Infof("creating read file: %s", csvReadName)
		f, err := os.Open(csvReadName)
		if err != nil {
			log.Fatalf("no csv read file found: %s", csvReadName)
		}
		m.CsvMu.Lock()
		defer m.CsvMu.Unlock()
		m.CsvStore[csvReadName] = NewCSVData(f, recycleData)
	}
	csvWriteName := handle.WriteToCsvName
	if csvWriteName != "" {
		log.Infof("creating write file: %s", csvWriteName)
		csvFile := CreateOrReplaceFile(csvWriteName)
		m.CsvMu.Lock()
		defer m.CsvMu.Unlock()
		m.CsvStore[csvWriteName] = NewCSVData(csvFile, false)
	}
}

func (m *LoadManager) HandleShutdownSignal() {
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Info("exit signal received, exiting")
		if m.SuiteConfig.GoroutinesDump {
			buf := make([]byte, 1<<20)
			stacklen := runtime.Stack(buf, true)
			log.Infof("=== received SIGTERM ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
		}
		m.Shutdown()
		os.Exit(1)
	}()
}

func (m *LoadManager) Shutdown() {
	for _, s := range m.Steps {
		for _, r := range s.Runners {
			r.Shutdown()
		}
	}
	m.CSVLog.Flush()
	m.RPSScalingLog.Flush()
	for _, s := range m.CsvStore {
		s.Flush()
		s.f.Close()
	}
}

// RunSuite starts suite and wait for all generator to shutdown
func (m *LoadManager) RunSuite() {
	m.HandleShutdownSignal()

	t := timeNow()
	startTime := epochNowMillis(t)
	hrStartTime := timeHumanReadable(t)

	for _, step := range m.Steps {
		log.Infof("running step: %s, execution mode: %s", step.Name, step.ExecutionMode)
		switch step.ExecutionMode {
		case ParallelMode:
			var wg sync.WaitGroup
			wg.Add(len(step.Runners))

			for _, r := range step.Runners {
				r.SetupHandleStore(m)
				go r.Run(&wg, m)
			}
			wg.Wait()
		case SequenceMode:
			for _, r := range step.Runners {
				r.SetupHandleStore(m)
				r.Run(nil, m)
			}
		case SequenceValidateMode:
			for _, r := range step.Runners {
				r.SetupHandleStore(m)
				r.Run(nil, m)
				r.SetValidationParams()
				r.Run(nil, m)
			}
		default:
			log.Fatal("please set execution_mode, parallel, sequence or sequence_validate")
		}
	}
	if m.GeneratorConfig.Grafana.URL != "" {
		t = timeNow()
		finishTime := epochNowMillis(t)
		hrFinishTime := timeHumanReadable(t)

		TimerangeUrl(startTime, finishTime)
		HumanReadableTestInterval(hrStartTime, hrFinishTime)
	}
	m.Shutdown()
}

func (m *LoadManager) CsvForHandle(name string) *CSVData {
	m.CsvMu.Lock()
	defer m.CsvMu.Unlock()
	s, ok := m.CsvStore[name]
	if !ok {
		log.Fatalf("no csv storage file found for: %s", name)
	}
	return s
}

// StoreHandleReports stores report for every handle in suite
func (m *LoadManager) StoreHandleReports() {
	ts := time.Now().Unix()
	for handleName, r := range m.Reports {
		b, err := json.MarshalIndent(r, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		repPath := filepath.Join(m.ReportDir, fmt.Sprintf(ReportFileTmpl, handleName, ts))
		log.Infof("writing report for handle [%s] in %s", handleName, repPath)
		f, err := os.Open(repPath)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := f.Write(b); err != nil {
			log.Fatal(err)
		}
		if !m.Degradation {
			m.WriteLastSuccess(handleName, ts)
		}
	}
}

// WriteLastSuccess writes ts of last successful run for handle
func (m *LoadManager) WriteLastSuccess(handleName string, ts int64) {
	lastSuccessFile := filepath.Join(m.ReportDir, handleName+"_last")
	createFileIfNotExists(lastSuccessFile)
	err := ioutil.WriteFile(lastSuccessFile, []byte(strconv.Itoa(int(ts))), 0777)
	if err != nil {
		log.Fatal(err)
	}
}

// CheckErrors checkStopIf Errors logic
func (m *LoadManager) CheckErrors() {
	for handleName, currentReport := range m.Reports {
		if len(currentReport.Metrics[handleName].Errors) > 0 {
			m.Failed = true
		}
	}
}

// CheckDegradation checks handle performance degradation to last successful run stored in *handle_name*_last file
func (m *LoadManager) CheckDegradation() {
	handleThreshold := viper.GetFloat64("checks.handle_threshold_percent")
	for handleName, currentReport := range m.Reports {
		lastReport, err := m.LastSuccessReportForHandle(handleName)
		if os.IsNotExist(err) {
			log.Infof("nothing to compare for %s handle, no reports in %s", handleName, m.ReportDir)
			continue
		}
		if _, ok := lastReport.Metrics[handleName]; !ok {
			log.Fatalf("no last report for handle %s found in last report", handleName)
		}
		if _, ok := currentReport.Metrics[handleName]; !ok {
			log.Fatalf("no last report for handle %s found in current report", handleName)
		}
		currentMean := currentReport.Metrics[handleName].Latencies.P50 / time.Millisecond
		lastMean := lastReport.Metrics[handleName].Latencies.P50 / time.Millisecond
		log.Infof("[ %s ] current: %dms, last: %dms\n", handleName, currentMean, lastMean)
		log.Infof("ratio: %f\n", float64(currentMean)/float64(lastMean))
		if float64(currentMean)/float64(lastMean) >= handleThreshold {
			log.Infof("p50 degradation of %s handle: %d > %d", handleName, currentMean, lastMean)
			m.Degradation = true
			continue
		}
	}
}

// LastSuccessReportForHandle gets last successful report for a handle
func (m *LoadManager) LastSuccessReportForHandle(handleName string) (*RunReport, error) {
	f, err := os.Open(filepath.Join(m.ReportDir, handleName+"_last"))
	defer f.Close()
	if err != nil {
		return nil, err
	}
	lastTs, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	lf, err := os.Open(filepath.Join(m.ReportDir, fmt.Sprintf("%s-%s.json", handleName, string(lastTs))))
	defer lf.Close()
	if err != nil {
		log.Fatal(err)
	}
	data, err := ioutil.ReadAll(lf)
	if err != nil {
		log.Fatal(err)
	}
	var runReport RunReport
	if err := json.Unmarshal(data, &runReport); err != nil {
		log.Fatal(err)
	}
	return &runReport, nil
}

// createFileIfNotExists creates file if not exists, used to not override csv data
func createFileIfNotExists(fname string) *os.File {
	var file *os.File
	fpath, _ := filepath.Abs(fname)
	_, err := os.Stat(fpath)
	if err != nil {
		file, err = os.Create(fname)
	} else {
		log.Fatalf("file %s already exists, please rename write_csv or read_csv file name in Config", fname)
	}
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func createFileOrAppend(fname string) *os.File {
	var file *os.File
	fpath, _ := filepath.Abs(fname)
	_, err := os.Stat(fpath)
	if err != nil {
		file, err = os.Create(fname)
	} else {
		file, err = os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func CreateOrReplaceFile(fname string) *os.File {
	fpath, _ := filepath.Abs(fname)
	_ = os.Remove(fpath)
	file, err := os.Create(fpath)
	if err != nil {
		log.Fatal(err)
	}
	return file
}

// createDirIfNotExists create dir if not exists recursively
func createDirIfNotExists(dirPath string) {
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		log.Fatal(err)
	}
}
