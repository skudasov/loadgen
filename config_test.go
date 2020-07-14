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
	"testing"
)

func TestLoadConfig(t *testing.T) {
	c := ConfigFromFile("config_test.json")
	if len(c.Metadata) != 0 {
		t.Error("expected empty metadata")
	}
	if c.RampUpTimeSec != 10 {
		t.Error("expected RampUpTimeSec 10")
	}
	if c.DoTimeoutSec != 5 {
		t.Error("expected timeout 5")
	}
}

func TestOverrideLoadedConfig(t *testing.T) {
	flag.Set("rps", "31")
	flag.Set("attack", "32")
	flag.Set("ramp", "33")
	flag.Set("max", "34")
	flag.Set("o", "here")
	flag.Set("verbose", "false")
	flag.Set("s", "?")
	flag.Set("timeout", "35")
	c := ConfigFromFile("config_test.json")
	if got, want := c.RPS, 31; got != want {
		t.Errorf("got %v want %v", got, want)
	}
	if got, want := c.AttackTimeSec, 32; got != want {
		t.Errorf("got %v want %v", got, want)
	}
	if got, want := c.RampUpTimeSec, 33; got != want {
		t.Errorf("got %v want %v", got, want)
	}
	if got, want := c.MaxAttackers, 34; got != want {
		t.Errorf("got %v want %v", got, want)
	}
	if got, want := c.OutputFilename, "here"; got != want {
		t.Errorf("got %v want %v", got, want)
	}
	if got, want := c.Verbose, false; got != want {
		t.Errorf("got %v want %v", got, want)
	}
	if got, want := c.RampUpStrategy, "?"; got != want {
		t.Errorf("got %v want %v", got, want)
	}
	if got, want := c.MaxAttackers, 34; got != want {
		t.Errorf("got %v want %v", got, want)
	}
	if got, want := c.DoTimeoutSec, 35; got != want {
		t.Errorf("got %v want %v", got, want)
	}
}
