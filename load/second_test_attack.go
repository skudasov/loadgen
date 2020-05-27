package load

import (
	"context"
	"github.com/skudasov/loadgen"
	"time"
)

type SecondTestAttack struct {
	loadgen.WithRunner
}

func (a *SecondTestAttack) Setup(hc loadgen.RunnerConfig) error {
	return nil
}
func (a *SecondTestAttack) Do(ctx context.Context) loadgen.DoResult {
	time.Sleep(2 * time.Second)
	return loadgen.DoResult{
		Error:        nil,
		RequestLabel: SecondTestLabel,
	}
}
func (a *SecondTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &SecondTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
