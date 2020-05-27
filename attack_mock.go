package loadgen

import (
	"context"
	"time"
)

type attackMock struct {
	sleep time.Duration
}

func (m *attackMock) Setup(c RunnerConfig) error {
	return nil
}

func (m *attackMock) Do(ctx context.Context) DoResult {
	time.Sleep(m.sleep)
	return DoResult{}
}

func (m *attackMock) Teardown() error {
	return nil
}

func (m *attackMock) Clone(r *Runner) Attack {
	return m
}

func (m *attackMock) GetRunner() *Runner {
	return nil
}

func (m *attackMock) StoreData() bool {
	return false
}

func (m *attackMock) GetManager() *LoadManager {
	return nil
}

func (m *attackMock) PutData(mo interface{}) error {
	return nil
}

func (m *attackMock) GetData() (interface{}, error) {
	return nil, nil
}
