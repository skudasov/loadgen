package loadgen

import (
	"context"
	"sync/atomic"
	"time"
)

type Monitored struct {
	Attack
}

func WithMonitor(a Attack) Monitored {
	return Monitored{a}
}

func (m Monitored) Do(ctx context.Context) DoResult {
	cfg := m.GetRunner().Config
	if cfg.DebugSleep != 0 {
		time.Sleep(time.Duration(cfg.DebugSleep) * time.Millisecond)
	}
	before := time.Now()
	result := m.Attack.Do(ctx)
	attackTime := time.Now().Sub(before)
	m.GetRunner().registerLabelTimings(result.RequestLabel).Update(attackTime)
	if result.Error != nil || result.StatusCode >= 400 {
		m.GetRunner().registerErrCount(result.RequestLabel).Inc(1)
	}
	return result
}

func (m Monitored) Setup(c RunnerConfig) error {
	if err := m.Attack.Setup(c); err != nil {
		return err
	}
	return nil
}

func (m Monitored) Clone(r *Runner) Attack {
	atomic.AddInt64(&r.goroutinesCount, 1)
	r.goroutinesCountGaugue.Update(r.goroutinesCount)
	return Monitored{m.Attack.Clone(r)}
}
