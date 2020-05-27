#### Load testing library & cli
Created for load tests that use (generated) clients in Go to communicate to services (in any supported language). By providing the Attack interface, any client and protocol could potentially be tested with this package.

Compared to existing HTTP load testing tools (e.g. tsenart/vegeta) that can send raw HTTP requests, this package requires the use of client code to send the requests and receive the response.

This tool is heavily based on https://github.com/emicklei/hazana, with added functionality:
- [x] multiple generators in one runtime
- [x] generate grafana dashboard for all attackers
- [x] load and store data for attackers
- [x] performance degradation checks
- [x] dump transport for debug
- [ ] automatic generation of load profile from logs

Install lib
```
go get github.com/skudasov/loadgen
```
Install cli
```
go install ~/go/pkg/mod/github.com/skudasov/loadgen\@${version}/cmd/loadcli.go
```

Bootstrap grafana + grafite
```
docker run -d -p 8181:80 -p 8125:8125/udp -p 8126:8126 --publish=2003:2003 --name kamon-grafana-dashboard kamon/grafana_graphite
```

Create default generator config in your home dir ~/generator.yaml
```yaml
host:  // host data for monitoring
  name: local_generator // used as graphite metrics prefix
  network_iface: en0
generator:
  target: https://ya.ru  // default target of attack
  responseTimeoutSec: 20
  rampUpStrategy: linear // linear | exp2
  verbose: true
execution_mode: parallel // generator execution mode, run attacker in modes parallel | sequence
grafana: // grafana configuration
  url: http://0.0.0.0:8181
  login: "admin"
  password: "admin"
graphite:
  url: 0.0.0.0:2003
  flushDurationSec: 1
  loadGeneratorPrefix: observer // prefix for graphite metrics
checks:
  handle_threshold_percent: 1.20
root_package_name: loadgen // your root package name
load_scripts_dir: load  // where all attackers and suite configs will be stored
timezone: Europe/Moscow
```

### Creating tests
Create new test
```
loadcli new first_test
```

Open load/first_test_attack.go, implement Setup and Do methods
```go
// Attack must be implemented by a service client.
type Attack interface {
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
	// StoreData should return if this scenario will save data, that gonna be needed for another scenario or verification
	StoreData() bool
}
```
If you require to put or get data for test, also implement
```go
	// PutData writes object representation to handle file
	PutData(mo interface{}) error
	// GetData reads object from handle file
	GetData() (interface{}, error)
```

Also default run config was created in run_configs/first_test.yaml
```yaml
dumptransport: true
http_timeout: 20
handles:
- name: first_test
  rps: 1
  attack_time_sec: 30
  ramp_up_sec: 1
  ramp_up_strategy: exp2
  max_attackers: 1
  verbose: true
  do_timeout_sec: 40
  store_data: false
  recycle_data: true
execution_mode: sequence
```

Now it's time to generate and upload grafana dashboard for your test
```
loadcli dashboard
```

And run test, build options are linux|darwin for now
```go
loadcli build darwin
./load_suite -config load/run_configs/first_test.yaml
```
If you have remote vm for running tests, upload it (you must have ssh keys copied to remote)
```
loadcli upload myuser@102.37.13.83:/home/myuser/loadtest
```

#### CI Run
If handle threshold percent is reached (default is 20% of p50 for any handle), or there is errors in any handle, pipeline will fail.
```yaml
checks:
    handle_threshold_percent: 1.2
```
All reports for handle is stored in reports dir

#### Debug
Bootstrap local kamon for debugging metrics, export dashboard from dir
```
docker run -d -p 8181:80 -p 8125:8125/udp -p 8126:8126 --publish=2003:2003 --name kamon-grafana-dashboard kamon/grafana_graphite
```
Turn on dumptransport in run config:
```yaml
dumptransport: true
```