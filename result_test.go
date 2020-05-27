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
