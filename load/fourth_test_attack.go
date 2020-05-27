package load

import (
	"context"
	"github.com/skudasov/loadgen"
	"time"
)

type FourthTestAttack struct {
	loadgen.WithRunner
}

func (a *FourthTestAttack) Setup(hc loadgen.RunnerConfig) error {
	return nil
}
func (a *FourthTestAttack) Do(ctx context.Context) loadgen.DoResult {
	time.Sleep(850 * time.Millisecond)
	return loadgen.DoResult{
		Error:        nil,
		RequestLabel: FourthTestLabel,
	}
}
func (a *FourthTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &FourthTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
