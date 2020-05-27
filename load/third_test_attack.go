package load

import (
	"context"
	"github.com/skudasov/loadgen"
	"time"
)

type ThirdTestAttack struct {
	loadgen.WithRunner
}

func (a *ThirdTestAttack) Setup(hc loadgen.RunnerConfig) error {
	return nil
}
func (a *ThirdTestAttack) Do(ctx context.Context) loadgen.DoResult {
	time.Sleep(400 * time.Millisecond)
	return loadgen.DoResult{
		Error:        nil,
		RequestLabel: ThirdTestLabel,
	}
}
func (a *ThirdTestAttack) Clone(r *loadgen.Runner) loadgen.Attack {
	return &ThirdTestAttack{WithRunner: loadgen.WithRunner{R: r}}
}
