package loadgen

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/spf13/viper"
)

const (
	fRPS            = "rps"
	fAttackTime     = "attack"
	fRampupTime     = "ramp"
	fMaxAttackers   = "max"
	fOutput         = "o"
	fVerbose        = "verbose"
	fSample         = "t"
	fRampupStrategy = "s"
	fDoTimeout      = "timeout"
)

var (
	oRPS            = flag.Int(fRPS, 1, "target number of requests per second, must be greater than zero")
	oAttackTime     = flag.Int(fAttackTime, 60, "duration of the attack in seconds")
	oRampupTime     = flag.Int(fRampupTime, 10, "ramp up time in seconds")
	oMaxAttackers   = flag.Int(fMaxAttackers, 10, "maximum concurrent attackers")
	oOutput         = flag.String(fOutput, "", "output file to write the Metrics per sample request index (use stdout if empty)")
	oVerbose        = flag.Bool(fVerbose, false, "produce more verbose logging")
	oSample         = flag.Int(fSample, 0, "test your attack implementation with a number of sample calls. Your program exits after this")
	oRampupStrategy = flag.String(fRampupStrategy, defaultRampupStrategy, "set the rampup strategy, possible values are {linear,exp2}")
	oDoTimeout      = flag.Int(fDoTimeout, 5, "timeout in seconds for each attack call")
)

var fullAttackStartedAt time.Time

// SuiteConfig suite config
type SuiteConfig struct {
	RootKeys       string `mapstructure:"rootkeys,omitempty" yaml:"rootkeys,omitempty"`
	RootRef        string `mapstructure:"rootref,omitempty" yaml:"rootref,omitempty"`
	DumpTransport  bool   `mapstructure:"dumptransport" yaml:"dumptransport"`
	GoroutinesDump bool   `mapstructure:"goroutines_dump" yaml:"goroutines_dump"`
	HttpTimeout    int    `mapstructure:"http_timeout" yaml:"http_timeout"`
	Steps          []Step `mapstructure:"steps" yaml:"steps"`
}

type Step struct {
	Name          string         `mapstructure:"name" yaml:"name"`
	ExecutionMode string         `mapstructure:"execution_mode" yaml:"execution_mode"`
	Handles       []RunnerConfig `mapstructure:"handles" yaml:"handles"`
}

type Checks struct {
	Type      string
	Query     string
	Threshold float64
	Interval  int
}

type Validation struct {
	AttackTimeSec int     `mapstructure:"attack_time_sec" yaml:"attack_time_sec"`
	Threshold     float64 `mapstructure:"threshold" yaml:"threshold"`
}

// RunnerConfig holds settings for a Runner
type RunnerConfig struct {
	WaitBeforeSec   int               `mapstructure:"wait_before_sec" yaml:"wait_before_sec"`
	HandleName      string            `mapstructure:"name" yaml:"name"`
	RPS             int               `mapstructure:"rps" yaml:"rps"`
	AttackTimeSec   int               `mapstructure:"attack_time_sec" yaml:"attack_time_sec"`
	RampUpTimeSec   int               `mapstructure:"ramp_up_sec" yaml:"ramp_up_sec"`
	RampUpStrategy  string            `mapstructure:"ramp_up_strategy" yaml:"ramp_up_strategy"`
	MaxAttackers    int               `mapstructure:"max_attackers" yaml:"max_attackers"`
	OutputFilename  string            `mapstructure:"outputFilename,omitempty" yaml:"outputFilename,omitempty"`
	Verbose         bool              `mapstructure:"verbose" yaml:"verbose"`
	Metadata        map[string]string `mapstructure:"metadata,omitempty" yaml:"metadata,omitempty"`
	DoTimeoutSec    int               `mapstructure:"do_timeout_sec" yaml:"do_timeout_sec"`
	StoreData       bool              `mapstructure:"store_data" yaml:"store_data"`
	RecycleData     bool              `mapstructure:"recycle_data" yaml:"recycle_data"`
	ReadFromCsvName string            `mapstructure:"csv_read,omitempty" yaml:"csv_read,omitempty"`
	WriteToCsvName  string            `mapstructure:"csv_write,omitempty" yaml:"csv_write,omitempty"`
	HandleParams    map[string]string `mapstructure:"handle_params,omitempty" yaml:"handle_params,omitempty"`
	IsValidationRun bool              `mapstructure:"validation_run" yaml:"validation_run"`
	StopIf          []Checks          `mapstructure:"stop_if" yaml:"stop_if"`
	Validation      Validation        `mapstructure:"validation" yaml:"validation"`

	// DebugSleep used as a crutch to not affect response time when one need to run test < 1 rps
	DebugSleep int `mapstructure:"debug_sleep"`
}

// Validate checks all settings and returns a list of strings with problems.
func (c RunnerConfig) Validate() (list []string) {
	if c.RPS <= 0 {
		list = append(list, "please set the RPS to a positive number of seconds")
	}
	if c.AttackTimeSec < 2 {
		list = append(list, "please set the attack time to a positive number of seconds > 1")
	}
	if c.RampUpTimeSec < 1 {
		list = append(list, "please set the attack time to a positive number of seconds > 0")
	}
	if c.MaxAttackers <= 0 {
		list = append(list, "please set a positive maximum number of attackers")
	}
	if c.DoTimeoutSec <= 0 {
		list = append(list, "please set the Do() timeout to a positive maximum number of seconds")
	}
	return
}

// timeout is in seconds
func (c RunnerConfig) timeout() time.Duration {
	return time.Duration(c.DoTimeoutSec) * time.Second
}

func (c RunnerConfig) rampupStrategy() string {
	if len(c.RampUpStrategy) == 0 {
		return defaultRampupStrategy
	}
	return c.RampUpStrategy
}

// ConfigFromFlags creates a RunnerConfig for use in a Runner.
func ConfigFromFlags() RunnerConfig {
	flag.Parse()
	return RunnerConfig{
		RPS:            *oRPS,
		AttackTimeSec:  *oAttackTime,
		RampUpTimeSec:  *oRampupTime,
		RampUpStrategy: *oRampupStrategy,
		Verbose:        *oVerbose,
		MaxAttackers:   *oMaxAttackers,
		OutputFilename: *oOutput,
		Metadata:       map[string]string{},
		DoTimeoutSec:   *oDoTimeout,
	}
}

// ConfigFromFile loads a RunnerConfig for use in a Runner.
func ConfigFromFile(named string) RunnerConfig {
	c := ConfigFromFlags() // always parse flags
	f, err := os.Open(named)
	defer f.Close()
	if err != nil {
		log.Fatal("unable to read configuration", err)
	}
	err = json.NewDecoder(f).Decode(&c)
	if err != nil {
		log.Fatal("unable to decode configuration", err)
	}
	applyFlagOverrides(&c)
	return c
}

// override with any flag set
func applyFlagOverrides(c *RunnerConfig) {
	flag.Visit(func(each *flag.Flag) {
		switch each.Name {
		case fRPS:
			c.RPS = *oRPS
		case fAttackTime:
			c.AttackTimeSec = *oAttackTime
		case fRampupTime:
			c.RampUpTimeSec = *oRampupTime
		case fVerbose:
			c.Verbose = *oVerbose
		case fMaxAttackers:
			c.MaxAttackers = *oMaxAttackers
		case fOutput:
			c.OutputFilename = *oOutput
		case fDoTimeout:
			c.DoTimeoutSec = *oDoTimeout
		}
	})
}

// LoadSuiteConfig loads yaml loadtest profile Config
func LoadSuiteConfig(cfgPath string) *SuiteConfig {
	log.Infof("loading suite config from: %s", cfgPath)
	viper.SetConfigType("yaml")
	viper.SetConfigFile(cfgPath)
	err := viper.MergeInConfig()
	if err != nil {
		log.Fatalf("Failed to readIn viper: %s\n", err)
	}
	var suiteCfg *SuiteConfig
	if err := viper.Unmarshal(&suiteCfg); err != nil {
		log.Fatalf("failed to unmarshal suite Config: %s\n", err)
	}
	return suiteCfg
}
