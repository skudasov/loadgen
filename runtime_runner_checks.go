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
	"github.com/prometheus/common/model"
	"strings"
	"time"
)

// PromBooleanQuery executes prometheus boolean query
func PromBooleanQuery(r *Runner) bool {
	q := r.CheckData[0].Query
	log.Infof("executing prometheus check: query: %s", q)
	if !strings.Contains(q, "bool") {
		log.Fatalf("only bool requests is allowed with default prometheus query checkStopIf, exiting")
	}
	val, _, err := r.PromClient.Query(context.Background(), q, time.Now())
	if err != nil {
		log.Fatalf("error executing prometheus query: %s, err: %s", q, err)
		return true
	}
	log.Infof("check result: %s, val type: %s", val, val.Type())
	switch {
	case val.Type() == model.ValScalar:
		scalarVal := val.(*model.Scalar)
		if scalarVal.Value == 1 {
			return true
		}
	case val.Type() == model.ValVector:
		vectorVal := val.(model.Vector)
		if len(vectorVal) > 1 {
			log.Fatalf("ambigious default check, prometheus request must be bool and return one vector or scalar, exiting")
		}
		if vectorVal[0].Value == 1 {
			return true
		}
	}
	return false
}

func ErrorPercentCheck(r *Runner, percent float64) bool {
	if r.RampUpMetrics[r.name] != nil && r.TestStage == rampUp {
		ratio := r.RampUpMetrics[r.name].successRatio
		if ratio > percent {
			return true
		}
		return false
	}
	if r.Metrics[r.name] != nil && r.TestStage == constantLoad {
		ratio := r.RampUpMetrics[r.name].successRatio
		if ratio > percent {
			return true
		}
		return false
	}
	return false
}
