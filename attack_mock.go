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
	"time"
)

type attackMock struct {
	sleep time.Duration
}

func (m *attackMock) Setup(c RunnerConfig) error {
	return nil
}

func (m *attackMock) Do(ctx context.Context) DoResult {
	time.Sleep(m.sleep)
	return DoResult{}
}

func (m *attackMock) Teardown() error {
	return nil
}

func (m *attackMock) Clone(r *Runner) Attack {
	return m
}

func (m *attackMock) GetRunner() *Runner {
	return nil
}

func (m *attackMock) StoreData() bool {
	return false
}

func (m *attackMock) GetManager() *LoadManager {
	return nil
}

func (m *attackMock) PutData(mo interface{}) error {
	return nil
}

func (m *attackMock) GetData() (interface{}, error) {
	return nil, nil
}
