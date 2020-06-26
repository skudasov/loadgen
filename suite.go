package loadgen

import (
	"flag"
	"os"
	"os/user"
	"path"

	"github.com/spf13/viper"
)

var log *Logger

type Prometheus struct {
	URL                     string `mapstructure:"url"`
	EnvLabel                string `mapstructure:"env_label"`
	Namespace               string `mapstructure:"namespace"`
	PulseLagThreshold       int    `mapstructure:"pulse_lag_threshold"`
	OpenedRequestsThreshold int    `mapstructure:"opened_requests_threshold"`
}

type DefaultGeneratorConfig struct {
	Host struct {
		Name         string `mapstructure:"name"`
		NetworkIface string `mapstructure:"network_iface"`
	} `mapstructure:"host"`
	Remotes []struct {
		Name          string `mapstructure:"name"`
		RemoteRootDir string `mapstructure:"remote_root_dir"`
		KeyPath       string `mapstructure:"key_path"`
	}
	Generator struct {
		Target             string `mapstructure:"target"`
		ResponseTimeoutSec int    `mapstructure:"responseTimeoutSec"`
		RampUpStrategy     string `mapstructure:"ramp_up_strategy"`
		Verbose            bool   `mapstructure:"verbose"`
	} `mapstructure:"generator"`
	ExecutionMode string `mapstructure:"execution_mode"`
	Grafana       struct {
		URL      string `mapstructure:"url"`
		Login    string `mapstructure:"login"`
		Password string `mapstructure:"password"`
	} `mapstructure:"grafana"`
	Graphite struct {
		URL                 string `mapstructure:"url"`
		FlushDurationSec    int    `mapstructure:"flushDurationSec"`
		LoadGeneratorPrefix string `mapstructure:"loadGeneratorPrefix"`
	} `mapstructure:"graphite"`
	Prometheus *Prometheus `mapstructure:"prometheus"`
	Checks     struct {
		HandleThresholdPercent float64 `mapstructure:"handle_threshold_percent"`
	} `mapstructure:"checks"`
	LoadScriptsDir string `mapstructure:"load_scripts_dir"`
	Timezone       string `mapstructure:"timezone"`
	Logging        struct {
		Level    string `mapstructure:"level"`
		Encoding string `mapstructure:"encoding"`
	} `mapstructure:"logging"`
}

func LoadDefaultGeneratorConfig(cfgPath string) *DefaultGeneratorConfig {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	viper.SetConfigType("yaml")
	viper.SetConfigFile(path.Join(usr.HomeDir, cfgPath))
	err = viper.MergeInConfig()
	if err != nil {
		log.Fatalf("Failed to readIn viper: %s\n", err)
	}
	var defaultGeneratorCfg *DefaultGeneratorConfig
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

func (c *DefaultGeneratorConfig) Validate() (list []string) {
	return nil
}

type attackerFactory func(string) Attack

type attackerChecksFactory func(string) RuntimeCheckFunc

// Run default run mode for suite, with degradation checks
func Run(factory attackerFactory, checksFactory attackerChecksFactory) {
	cfgPath := flag.String("config", "", "loadtest attack profile config filepath")
	genCfgPath := flag.String("gen_config", "generator.yaml", "generator config filepath")
	flag.Parse()
	genConfig := LoadDefaultGeneratorConfig(*genCfgPath)
	osMetrics := NewHostOSMetrics(genConfig.Host.Name, genConfig.Graphite.URL, 1, genConfig.Host.NetworkIface)
	osMetrics.Watch(1)
	lm := SuiteFromSteps(factory, checksFactory, *cfgPath, genConfig)
	lm.RunSuite()
	if lm.Failed {
		os.Exit(1)
	}
}

// SuiteFromSteps create runners for every step
func SuiteFromSteps(factory attackerFactory, checksFactory attackerChecksFactory, cfgPath string, genCfg *DefaultGeneratorConfig) *LoadManager {
	cfg := LoadSuiteConfig(cfgPath)
	lm := NewLoadManager(cfg, genCfg)
	for _, step := range lm.SuiteConfig.Steps {
		runners := make([]*Runner, 0)
		for _, handle := range step.Handles {
			runners = append(runners, NewRunner(
				handle.HandleName,
				lm,
				factory(handle.HandleName),
				checksFactory(handle.HandleName),
				handle),
			)
		}
		lm.Steps = append(lm.Steps, RunStep{
			Name:          step.Name,
			ExecutionMode: step.ExecutionMode,
			Runners:       runners,
		})
	}
	return lm
}
