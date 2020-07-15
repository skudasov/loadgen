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
	"testing"
	"time"
)

func TestAttack1(t *testing.T) {
	attacker := new(attackMock)
	dur := 10 * time.Millisecond
	attacker.sleep = dur
	next := make(chan bool)
	quit := make(chan bool)
	results := make(chan result)

	go attack(attacker, next, quit, results, 1*time.Second)

	next <- true
	r := <-results
	quit <- true
	if got, want := r.doResult.Error, error(nil); got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	if got, want := r.elapsed, dur; got < want {
		t.Fatalf("got %v want >= %v", got, want)
	}
}

func TestAttackTimeout(t *testing.T) {
	attacker := new(attackMock)
	dur := 2 * time.Second
	attacker.sleep = dur
	next := make(chan bool)
	quit := make(chan bool)
	results := make(chan result)

	go attack(attacker, next, quit, results, 1*time.Second)

	next <- true
	r := <-results
	quit <- true
	if got, want := r.doResult.Error, errAttackDoTimedOut; got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
