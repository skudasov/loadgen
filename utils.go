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
	graphite "github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var once sync.Once

func StartGraphiteSender(prefix string, flushDuration time.Duration, url string) {
	once.Do(func() {
		log.Infof("[grafana-monitoring] setup graphite client with url: %s", url)
		addr, err := net.ResolveTCPAddr("tcp", url)
		if err != nil {
			log.Fatalf("[grafana-monitoring] ResolveTCPAddr on [%s] failed error [%v] ", url, err)
		}
		go graphite.Graphite(
			metrics.DefaultRegistry,
			flushDuration*time.Second,
			prefix,
			addr,
		)
	})
}

func timeNow() time.Time {
	return time.Now()
}

func timeHumanReadable(t time.Time) string {
	location, _ := time.LoadLocation(viper.GetString("timezone"))
	return t.In(location).String()
}

func epochNowMillis(t time.Time) int64 {
	return t.UnixNano() / 1000000
}

func RandInt() int {
	r1 := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r1.Intn(999999999)
}

type AtomicBool struct {
	flag int32
}

func (b *AtomicBool) Set(value bool) {
	var i int32 = 0
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.flag), i)
}

func (b *AtomicBool) Get() bool {
	if atomic.LoadInt32(&(b.flag)) != 0 {
		return true
	}
	return false
}
