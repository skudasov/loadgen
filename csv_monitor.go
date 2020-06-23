package loadgen

import (
	"context"
	"time"
)

type CSVMonitored struct {
	Attack
}

func WithCSVMonitor(a Attack) CSVMonitored {
	return CSVMonitored{a}
}

func (m CSVMonitored) Do(ctx context.Context) DoResult {
	cfg := m.GetRunner().Config
	if cfg.DebugSleep != 0 {
		time.Sleep(time.Duration(cfg.DebugSleep) * time.Millisecond)
	}
	before := time.Now()
	result := m.Attack.Do(ctx)
	attackTime := time.Now().Sub(before)
	status := "ok"
	if result.Error != nil || result.StatusCode >= 400 {
		status = "err"
	}
	entry := []string{result.RequestLabel, before.String(), attackTime.String(), status}
	if err := m.GetManager().CSVLog.Write(entry); err != nil {
		log.Fatal(err)
	}
	return result
}

func (m CSVMonitored) Setup(c RunnerConfig) error {
	if err := m.Attack.Setup(c); err != nil {
		return err
	}
	return nil
}

func (m CSVMonitored) Clone(r *Runner) Attack {
	return CSVMonitored{m.Attack.Clone(r)}
}
