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
	e "errors"
	"time"
)

// Attack must be implemented by a service client.
type Attack interface {
	Runnable
	// Setup should establish the connection to the service
	// It may want to access the Config of the Runner.
	Setup(c RunnerConfig) error
	// Do performs one request and is executed in a separate goroutine.
	// The context is used to cancel the request on timeout.
	Do(ctx context.Context) DoResult
	// Teardown can be used to close the connection to the service
	Teardown() error
	// Clone should return a fresh new Attack
	// Make sure the new Attack has values for shared struct fields initialized at Setup.
	Clone(r *Runner) Attack
}

// Runnable contains default generator/suite configs and methods to access them
type Runnable interface {
	// GetManager get test manager with all required data files/readers/writers
	GetManager() *LoadManager
	// GetRunner get current runner
	GetRunner() *Runner
}

type Datable interface {
	// PutData writes object representation to handle file
	PutData(mo interface{}) error
	// GetData reads object from handle file
	GetData() (interface{}, error)
}

// WithRunner embeds Runner with all configs to be accessible for attacker
type WithRunner struct {
	R *Runner
}

func (a *WithRunner) Teardown() error { return nil }

func (a *WithRunner) GetManager() *LoadManager {
	return a.R.Manager
}

func (a *WithRunner) GetRunner() *Runner {
	return a.R
}

type WithData struct {
}

var errAttackDoTimedOut = e.New("Attack Do(ctx) timedout")

// attack calls attacker.Do upon each received next token, forever
// attack aborts the loop on a quit receive
// attack sends a result on the results channel after each call.
func attack(attacker Attack, next, quit <-chan bool, results chan<- result, timeout time.Duration) {
	for {
		select {
		case <-next:
			begin := time.Now()
			done := make(chan DoResult)
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			go func() {
				done <- attacker.Do(ctx)
			}()
			var dor DoResult
			// either get the result from the attacker or from the timeout
			select {
			case <-ctx.Done():
				dor = DoResult{Error: errAttackDoTimedOut}
			case dor = <-done:
			}
			end := time.Now()
			results <- result{
				doResult: dor,
				begin:    begin,
				end:      end,
				elapsed:  end.Sub(begin),
			}
		case <-quit:
			return
		}
	}
}
