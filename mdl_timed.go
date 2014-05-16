package statedb

import (
	// "errors"
	// "github.com/paddie/statedb"
	"time"
)

type TimedModel struct {
	interval time.Duration
	lastCpt  time.Time
}

func NewTimedModel(interval time.Duration) *TimedModel {
	return &TimedModel{
		interval: interval,
	}
}

func (r *TimedModel) Name() string {
	return "TimedModel"
}

func (r *TimedModel) Train(_ []PricePoint, _ float64) error {
	r.lastCpt = time.Now()
	return nil
}

func (r *TimedModel) StatUpdate(_, _ float64) error {
	return nil
}

func (r *TimedModel) PriceUpdate(_ float64, _ time.Time) error {
	return nil
}

func (r *TimedModel) Checkpoint(_, _, _ float64) (bool, error) {

	now := time.Now()

	if now.Sub(r.lastCpt) >= r.interval {
		r.lastCpt = now

		return true, nil
	}

	return false, nil
}

func (r *TimedModel) Quit() error {
	return nil
}
