package loadgen

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"
)

type result struct {
	begin, end time.Time
	elapsed    time.Duration
	doResult   DoResult
}

// DoResult is the return value of a Do call on an Attack.
type DoResult struct {
	// Label identifying the request that was send which is only used for reporting the Metrics.
	RequestLabel string
	// The error that happened when sending the request or receiving the response.
	Error error
	// The HTTP status code.
	StatusCode int
	// Number of bytes transferred when sending the request.
	BytesIn int64
	// Number of bytes transferred when receiving the response.
	BytesOut int64
}

// RunReport is a composition of configuration, measurements and custom output from a loadtest Run.
type RunReport struct {
	StartedAt     time.Time    `json:"startedAt"`
	FinishedAt    time.Time    `json:"finishedAt"`
	Configuration RunnerConfig `json:"configuration"`
	// RunError is set when a Run could not be called or executed.
	RunError string              `json:"runError"`
	Metrics  map[string]*Metrics `json:"Metrics"`
	// Failed can be set by your loadtest test program to indicate that the results are not acceptable.
	Failed bool `json:"failed"`
	// Output is used to publish any custom output in the report.
	Output map[string]interface{} `json:"output"`
}

// NewErrorReport returns a report when a Run could not be called or executed.
func NewErrorReport(err error, config RunnerConfig) RunReport {
	return RunReport{
		StartedAt:     time.Now(),
		FinishedAt:    time.Now(),
		RunError:      err.Error(),
		Configuration: config,
		Failed:        true, // clearly the Run was not acceptable
		Output:        map[string]interface{}{},
	}
}

// PrintReport writes the JSON report to a file or stdout, depending on the configuration.
func PrintReport(r RunReport) {
	// make secrets in Metadata unreadable
	for k := range r.Configuration.Metadata {
		if strings.HasSuffix(k, "*") {
			r.Configuration.Metadata[k] = "***---***---***"
		}
	}
	var out io.Writer
	if len(r.Configuration.OutputFilename) > 0 {
		file, err := os.Create(r.Configuration.OutputFilename)
		if err != nil {
			log.Fatal("unable to create output file", err)
		}
		defer file.Close()
		out = file
	} else {
		out = os.Stdout
	}
	data, _ := json.MarshalIndent(r, "", "\t")
	out.Write(data)
	// if verbose and filename is given
	if len(r.Configuration.OutputFilename) > 0 && r.Configuration.Verbose {
		os.Stdout.Write(data)
	}
}
