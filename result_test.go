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
	e "errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRunReportWithError(t *testing.T) {
	f := filepath.Join(os.TempDir(), "report.json")
	r := NewErrorReport(e.New("something broke"), RunnerConfig{OutputFilename: f})
	if !r.Failed {
		t.Error("expected failed Run")
	}
	PrintReport(r)
	data, _ := ioutil.ReadFile(f)
	b := RunReport{}
	if err := json.Unmarshal(data, &b); err != nil {
		t.Log(err)
	}
	t.Log(f)
}
