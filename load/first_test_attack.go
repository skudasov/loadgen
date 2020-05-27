package load

import (
	"context"
	"errors"
	"github.com/skudasov/loadgen"
	"math/rand"
	"time"
)

type FirstTestAttack struct {
	loadgen.WithRunner
}

func (a *FirstTestAttack) Setup(hc loadgen.RunnerConfig) error {
	return nil
}
func (a *FirstTestAttack) Do(ctx context.Context) loadgen.DoResult {
	var err error
	time.Sleep(1 * time.Second)
	r := rand.Intn(100)
	if r > 50 {
		err = errors.New("epic error")
	}
	return loadgen.DoResult{
		Error:        err,
		RequestLabel: FirstTestLabel,
	}
}
func (a *FirstTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &FirstTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
