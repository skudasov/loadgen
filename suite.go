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
	"flag"
	"os"
)

var log *Logger

type attackerFactory func(string) Attack

type attackerChecksFactory func(string) RuntimeCheckFunc

type BeforeSuite func(config *GeneratorConfig) error
type AfterSuite func(config *GeneratorConfig) error

// Run default run mode for suite, with degradation checks
func Run(factory attackerFactory, checksFactory attackerChecksFactory, beforeSuite BeforeSuite, afterSuite AfterSuite) {
	cfgPath := flag.String("config", "", "loadtest attack profile config filepath")
	genCfgPath := flag.String("gen_config", "generator.yaml", "generator config filepath")
	flag.Parse()
	if *cfgPath == "" {
		log.Fatal("provide path to suite config, -config example.yaml")
	}
	if *genCfgPath == "" {
		log.Fatal("provide path to generator config, -gen_config example.yaml")
	}
	genConfig := LoadDefaultGeneratorConfig(*genCfgPath)
	if genConfig.Host.CollectMetrics {
		log.Infof("starting host metrics monitor")
		osMetrics := NewHostOSMetrics(genConfig.Host.Name, genConfig.Graphite.URL, 1, genConfig.Host.NetworkIface)
		osMetrics.Watch(1)
	}
	lm := SuiteFromSteps(factory, checksFactory, *cfgPath, genConfig)
	if beforeSuite != nil {
		if err := beforeSuite(genConfig); err != nil {
			log.Fatalf("before suite func failed: %s", err)
		}
	}
	lm.RunSuite()
	if afterSuite != nil {
		if err := afterSuite(genConfig); err != nil {
			log.Fatalf("before suite func failed: %s", err)
		}
	}
	if lm.ValidationFailed {
		os.Exit(1)
	}
}

// SuiteFromSteps create runners for every step
func SuiteFromSteps(factory attackerFactory, checksFactory attackerChecksFactory, cfgPath string, genCfg *GeneratorConfig) *LoadManager {
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
