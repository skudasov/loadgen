package loadgen

import (
	graphite "github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"math/rand"
	"net"
	"sync"
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
