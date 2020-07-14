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
	"fmt"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/network"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/rcrowley/go-metrics"
)

// GetCPU get user + system cpu used
func (m *HostMetrics) GetCPU() int64 {
	before, err := cpu.Get()
	if err != nil {
		log.Info("[ OS Metrics ] failed to get cpu Metrics")
		return 0
	}
	time.Sleep(time.Duration(1) * time.Second)
	after, err := cpu.Get()
	if err != nil {
		log.Info("[ OS Metrics ] failed to get cpu Metrics")
		return 0
	}
	total := float64(after.Total - before.Total)
	// user + system
	return int64(100 - float64(after.Idle-before.Idle)/total*100)
}

func (m *HostMetrics) SelectNetworkInterface(stats []network.Stats) *network.Stats {
	for _, nd := range stats {
		if nd.Name == m.networkInterface {
			return &nd
		}
	}
	log.Fatalf("no interface found, interface %s doesn't exist", m.networkInterface)
	return nil
}

// GetNetwork get rx/tx for particular interface
func (m *HostMetrics) GetNetwork() (int64, int64) {
	before, err := network.Get()
	if err != nil {
		log.Info("[ OS Metrics ] failed to get network Metrics")
		return 0, 0
	}
	beforeData := m.SelectNetworkInterface(before)
	time.Sleep(time.Duration(1) * time.Second)
	after, err := network.Get()
	if err != nil {
		log.Info("[ OS Metrics ] failed to get network Metrics")
		return 0, 0
	}
	afterData := m.SelectNetworkInterface(after)

	totalRx := float64(afterData.RxBytes - beforeData.RxBytes)
	totalTx := float64(afterData.TxBytes - beforeData.TxBytes)
	return int64(totalRx), int64(totalTx)
}

// GetMem get all mem and swap used/free/total stats
func (m *HostMetrics) GetMem() *memory.Stats {
	mem, err := memory.Get()
	if err != nil {
		log.Info("[ OS Metrics ] failed to get memory Metrics")
		return nil
	}
	return mem
}

type HostMetrics struct {
	hostPrefix       string
	graphiteUrl      string
	flushDuration    time.Duration
	networkInterface string

	cpuUserSystemPercent metrics.Gauge

	memTotal     metrics.Gauge
	memFree      metrics.Gauge
	memUsed      metrics.Gauge
	memCached    metrics.Gauge
	memSwapTotal metrics.Gauge
	memSwapUsed  metrics.Gauge
	memSwapFree  metrics.Gauge

	rx metrics.Gauge
	tx metrics.Gauge
}

// NewOsMetrics
func NewHostOSMetrics(hostPrefix string, graphiteUrl string, flushDurationSec int, networkInterface string) *HostMetrics {
	return &HostMetrics{
		hostPrefix:           hostPrefix,
		graphiteUrl:          graphiteUrl,
		flushDuration:        time.Duration(flushDurationSec),
		networkInterface:     networkInterface,
		cpuUserSystemPercent: RegisterGauge("cpu_used"),
		memTotal:             RegisterGauge("mem_total"),
		memFree:              RegisterGauge("mem_free"),
		memUsed:              RegisterGauge("mem_used"),
		memCached:            RegisterGauge("mem_cached"),
		memSwapTotal:         RegisterGauge("mem_swap_total"),
		memSwapUsed:          RegisterGauge("mem_swap_used"),
		memSwapFree:          RegisterGauge("mem_swap_free"),
		rx:                   RegisterGauge(fmt.Sprintf("net_%s_rx", networkInterface)),
		tx:                   RegisterGauge(fmt.Sprintf("net_%s_tx", networkInterface)),
	}
}

// Watch updates generator host Metrics
func (m *HostMetrics) Watch(intervalSec int) {
	go func() {
		ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
		for {
			select {
			case <-ticker.C:
				cpuUserSystem := m.GetCPU()
				m.cpuUserSystemPercent.Update(cpuUserSystem)

				mem := m.GetMem()
				m.memTotal.Update(int64(mem.Total))
				m.memFree.Update(int64(mem.Free))
				m.memUsed.Update(int64(mem.Used))
				m.memCached.Update(int64(mem.Cached))
				m.memSwapTotal.Update(int64(mem.SwapTotal))
				m.memSwapUsed.Update(int64(mem.SwapUsed))
				m.memSwapFree.Update(int64(mem.SwapFree))

				rx, tx := m.GetNetwork()
				m.rx.Update(rx)
				m.tx.Update(tx)
			}
		}
	}()
}

// RegisterGauge registers gauge metric to graphite
func RegisterGauge(name string) metrics.Gauge {
	g := metrics.NewGauge()
	if err := metrics.Register(name, g); err != nil {
		log.Fatal(err)
	}
	return g
}
