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
	"fmt"
	"time"
)

type CSVMonitored struct {
	Attack
}

func WithCSVMonitor(a Attack) CSVMonitored {
	return CSVMonitored{a}
}

func (m CSVMonitored) Do(ctx context.Context) DoResult {
	cfg := m.GetRunner().Config
	if cfg.DebugSleep != 0 {
		time.Sleep(time.Duration(cfg.DebugSleep) * time.Millisecond)
	}
	before := time.Now()
	result := m.Attack.Do(ctx)
	attackTime := time.Now().Sub(before)
	status := "ok"
	if result.Error != nil || result.StatusCode >= 400 {
		m.GetRunner().L.Infof("err: %s", result.Error)
		status = "err"
	}
	beforeUnix := fmt.Sprintf("%d", before.Unix())
	entry := []string{result.RequestLabel, beforeUnix, attackTime.String(), status}
	m.GetManager().CSVLogMu.Lock()
	defer m.GetManager().CSVLogMu.Unlock()
	if err := m.GetManager().CSVLog.Write(entry); err != nil {
		log.Fatal(err)
	}
	return result
}

func (m CSVMonitored) Setup(c RunnerConfig) error {
	if err := m.Attack.Setup(c); err != nil {
		return err
	}
	return nil
}

func (m CSVMonitored) Clone(r *Runner) Attack {
	return CSVMonitored{m.Attack.Clone(r)}
}
