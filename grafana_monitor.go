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
	"context"
	"sync/atomic"
	"time"
)

type Monitored struct {
	Attack
}

func WithMonitor(a Attack) Monitored {
	return Monitored{a}
}

func (m Monitored) Do(ctx context.Context) DoResult {
	cfg := m.GetRunner().Config
	if cfg.DebugSleep != 0 {
		time.Sleep(time.Duration(cfg.DebugSleep) * time.Millisecond)
	}
	before := time.Now()
	result := m.Attack.Do(ctx)
	attackTime := time.Now().Sub(before)
	m.GetRunner().registerLabelTimings(result.RequestLabel).Update(attackTime)
	if result.Error != nil || result.StatusCode >= 400 {
		m.GetRunner().registerErrCount(result.RequestLabel).Inc(1)
	}
	return result
}

func (m Monitored) Setup(c RunnerConfig) error {
	if err := m.Attack.Setup(c); err != nil {
		return err
	}
	return nil
}

func (m Monitored) Clone(r *Runner) Attack {
	atomic.AddInt64(&r.goroutinesCount, 1)
	r.goroutinesCountGaugue.Update(r.goroutinesCount)
	return Monitored{m.Attack.Clone(r)}
}
