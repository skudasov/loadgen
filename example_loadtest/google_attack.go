package example_loadtest

import (
	"context"
	"github.com/skudasov/loadgen"
	"github.com/spf13/viper"
	"net/http"
)

type GoogleAttack struct {
	loadgen.WithRunner
	client *http.Client
}

func (a *GoogleAttack) PutData(mo interface{}) error {
	return nil
}

func (a *GoogleAttack) GetData() (interface{}, error) {
	//data := loadgen.DefaultReadCSV(a.GetManager(), a.genConfig.ReadFromCsvName)
	return nil, nil
}

func (a *GoogleAttack) Setup(hc loadgen.RunnerConfig) error {
	a.client = loadgen.NewLoggingHTTPClient(viper.GetBool("dumptransport"), viper.GetInt("http_timeout"))
	return nil
}

func (a *GoogleAttack) Do(ctx context.Context) loadgen.DoResult {
	//_, err := a.GetData()
	resp, err := a.client.Get("https://google.com")
	if err != nil {
		return loadgen.DoResult{
			RequestLabel: GoogleLabel,
			Error:        err,
		}
	}
	if resp.StatusCode >= 400 {
		return loadgen.DoResult{
			RequestLabel: GoogleLabel,
			Error:        err,
		}
	}
	return loadgen.DoResult{
		RequestLabel: GoogleLabel,
		Error:        err,
	}
}

func (a *GoogleAttack) Teardown() error { return nil }

func (a *GoogleAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &GoogleAttack{
		WithRunner: loadgen.WithRunner{r},
	}
}

func (a *GoogleAttack) StoreData() bool {
	return false
}
