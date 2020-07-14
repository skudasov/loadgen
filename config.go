/*
 *    Copyright [2020] Sergey Kudasov
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

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

// Prometheus prometheus config
type Prometheus struct {
	// URL prometheus base url
	URL string `mapstructure:"url"`
	// EnvLabel prometheus environment label
	EnvLabel string `mapstructure:"env_label"`
	// Namespace prometheus namespace
	Namespace string `mapstructure:"namespace"`
}

type GeneratorConfig struct {
	// Host current vm host configuration
	Host struct {
		// Name used in grafana metrics as prefix
		Name string `mapstructure:"name"`
		// NetworkIface default network interface to collect metrics from
		NetworkIface string `mapstructure:"network_iface"`
		// CollectMetrics collect host metrics flag
		CollectMetrics bool `mapstructure:"collect_metrics"`
	} `mapstructure:"host"`
	// Remotes contains remote generator vm data
	Remotes []struct {
		// Name hostname of remote generator
		Name string `mapstructure:"name"`
		// RemoteRootDir remote root dir of a test
		RemoteRootDir string `mapstructure:"remote_root_dir"`
		// KeyPath path to ssh pub key
		KeyPath string `mapstructure:"key_path"`
	}
	// Generator generator specific config
	Generator struct {
		// Target base url to attack
		Target string `mapstructure:"target"`
		// ResponseTimeoutSec response timeout in seconds
		ResponseTimeoutSec int `mapstructure:"responseTimeoutSec"`
		// RampUpStrategy ramp up strategy: linear | exp2
		RampUpStrategy string `mapstructure:"ramp_up_strategy"`
		// Verbose allows to print debug generator logs
		Verbose bool `mapstructure:"verbose"`
	} `mapstructure:"generator"`
	// ExecutionMode step execution mode: sequence, sequence_validate, parallel
	ExecutionMode string `mapstructure:"execution_mode"`
	// Grafana related config
	Grafana struct {
		// URL base url of grafana, ex.: http://0.0.0.0:8181
		URL string `mapstructure:"url"`
		// Login login
		Login string `mapstructure:"login"`
		// Password password
		Password string `mapstructure:"password"`
	} `mapstructure:"grafana"`
	// Graphite related config
	Graphite struct {
		// URL graphite base url, ex.: 0.0.0.0:2003
		URL string `mapstructure:"url"`
		// FlushIntervalSec flush interval in seconds
		FlushIntervalSec int `mapstructure:"flushDurationSec"`
		// LoadGeneratorPrefix prefix to be used in graphite metrics
		LoadGeneratorPrefix string `mapstructure:"loadGeneratorPrefix"`
	} `mapstructure:"graphite"`
	Prometheus *Prometheus `mapstructure:"prometheus"`
	// LoadScriptsDir relative from cwd load dir path, ex.: load
	LoadScriptsDir string `mapstructure:"load_scripts_dir"`
	// Timezone timezone used for grafana url, ex.: Europe/Moscow
	Timezone string `mapstructure:"timezone"`
	// Logging logging related config
	Logging struct {
		// Level level of allowed log messages,ex.: debug | info
		Level string `mapstructure:"level"`
		// Encoding encoding of logs, ex.: console | json
		Encoding string `mapstructure:"encoding"`
	} `mapstructure:"logging"`
}

func LoadDefaultGeneratorConfig(cfgPath string) *GeneratorConfig {
	viper.SetConfigType("yaml")
	viper.SetConfigFile(cfgPath)
	err := viper.MergeInConfig()
	if err != nil {
		log.Fatalf("Failed to readIn viper: %s\n", err)
	}
	var defaultGeneratorCfg *GeneratorConfig
	if err := viper.Unmarshal(&defaultGeneratorCfg); err != nil {
		log.Fatalf("failed to unmarshal default generator config: %s\n", err)
	}
	errs := defaultGeneratorCfg.Validate()
	if len(errs) != 0 {
		log.Fatalf("Errors in default suite config validation: %s", errs)
	}
	log = NewLogger()
	return defaultGeneratorCfg
}

func (c *GeneratorConfig) Validate() (list []string) {
	return nil
}

// SuiteConfig suite config
type SuiteConfig struct {
	RootKeys string `mapstructure:"rootkeys,omitempty" yaml:"rootkeys,omitempty"`
	RootRef  string `mapstructure:"rootref,omitempty" yaml:"rootref,omitempty"`
	// DumpTransport dumps request/response in stdout
	DumpTransport bool `mapstructure:"dumptransport" yaml:"dumptransport"`
	// GoroutinesDump dump goroutines when SIGHUP
	GoroutinesDump bool `mapstructure:"goroutines_dump" yaml:"goroutines_dump"`
	// HttpTimeout default http client timeout
	HttpTimeout int `mapstructure:"http_timeout" yaml:"http_timeout"`
	// Steps load test steps
	Steps []Step `mapstructure:"steps" yaml:"steps"`
}

// Step loadtest step config
type Step struct {
	// Name loadtest step name
	Name string `mapstructure:"name" yaml:"name"`
	// ExecutionMode handles execution mode: sequence, sequence_validate, parallel
	ExecutionMode string `mapstructure:"execution_mode" yaml:"execution_mode"`
	// Handles handle configs
	Handles []RunnerConfig `mapstructure:"handles" yaml:"handles"`
}

// Checks stop criteria checks
type Checks struct {
	// Type error check mode, ex.: error | prometheus
	Type string
	// Query prometheus bool query
	Query string
	// Threshold fail threshold, from 0 to 1, float
	Threshold float64
	// Interval check interval in seconds
	Interval int
}

// Validation validation config
type Validation struct {
	// AttackTimeSec validation attack time sec
	AttackTimeSec int `mapstructure:"attack_time_sec" yaml:"attack_time_sec"`
	// Threshold percent of max rps to validate, ex.: 0.7 means 70% of max rps
	Threshold float64 `mapstructure:"threshold" yaml:"threshold"`
}

// RunnerConfig runner config
type RunnerConfig struct {
	// WaitBeforeSec debug sleep before starting runner when checking condition is impossible
	WaitBeforeSec int `mapstructure:"wait_before_sec" yaml:"wait_before_sec"`
	// HandleName name of a handle, must be the same as test label in labels.go
	HandleName string `mapstructure:"name" yaml:"name"`
	// RPS max requests per second limit, load profile depends on AttackTimeSec and RampUpTimeSec
	RPS int `mapstructure:"rps" yaml:"rps"`
	// AttackTimeSec time of the test in seconds
	AttackTimeSec int `mapstructure:"attack_time_sec" yaml:"attack_time_sec"`
	// RampUpTimeSec ramp up period in seconds, in which RPS will be increased to max of RPS parameter
	RampUpTimeSec int `mapstructure:"ramp_up_sec" yaml:"ramp_up_sec"`
	// RampUpStrategy ramp up strategy: linear | exp2
	RampUpStrategy string `mapstructure:"ramp_up_strategy" yaml:"ramp_up_strategy"`
	// MaxAttackers max amount of goroutines to attack
	MaxAttackers int `mapstructure:"max_attackers" yaml:"max_attackers"`
	// OutputFilename report filename
	OutputFilename string `mapstructure:"outputFilename,omitempty" yaml:"outputFilename,omitempty"`
	// Verbose allows to print generator debug info
	Verbose bool `mapstructure:"verbose" yaml:"verbose"`
	// Metadata load run metadata
	Metadata map[string]string `mapstructure:"metadata,omitempty" yaml:"metadata,omitempty"`
	// DoTimeoutSec attacker.Do() func timeout
	DoTimeoutSec int `mapstructure:"do_timeout_sec" yaml:"do_timeout_sec"`
	// StoreData flag to check if test must put some data in csv for later validation
	StoreData bool `mapstructure:"store_data" yaml:"store_data"`
	// RecycleData flag to allow recycling data from csv when it ends
	RecycleData bool `mapstructure:"recycle_data" yaml:"recycle_data"`
	// ReadFromCsvName path to csv file to get data for test, use DefaultReadCSV/DefaultWriteCSV to read/write data for test
	ReadFromCsvName string `mapstructure:"csv_read,omitempty" yaml:"csv_read,omitempty"`
	// WriteToCsvName path to csv file to write data from test, use DefaultReadCSV/DefaultWriteCSV to read/write data for test
	WriteToCsvName string `mapstructure:"csv_write,omitempty" yaml:"csv_write,omitempty"`
	// HandleParams handle params metadata, ex. limit=100
	HandleParams map[string]string `mapstructure:"handle_params,omitempty" yaml:"handle_params,omitempty"`
	// IsValidationRun flag to know it's test run that validates max rps
	IsValidationRun bool `mapstructure:"validation_run" yaml:"validation_run"`
	// StopIf describes stop test criteria
	StopIf []Checks `mapstructure:"stop_if" yaml:"stop_if"`
	// Validation validation config
	Validation Validation `mapstructure:"validation" yaml:"validation"`

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
