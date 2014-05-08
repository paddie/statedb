package statedb

import (
	"errors"
	"github.com/paddie/statedb/monitor"
	"time"
)

type Exponential struct {
	Mean   float64
	Lambda float64
	Bid    float64
	Stat   *Stat
	Price  float64
}

func NewExponential() *Exponential {
	return &Exponential{}
}

func (e *Exponential) Name() string {
	return "ExponentialDistribution"
}

func (e *Exponential) take(t, t_p float64) float64 {

	return 0.0, errors.New("Something")
}

func (e *Exponential) skip(t, t_p float64) float64 {
	return 0.0, errors.New("Something")
}

func (e *Exponential) t(w, t_p float64) float64 {
	return 0.0
}

func (e *Exponential) Train(trace *monitor.Trace, bid float64) error {
	e.Mean, _, _, _, _ = trace.AliveStats(bid)
	e.Lambda = 1.0 / e.Mean
	e.Price = trace.Latest.SpotPrice
	return nil
}

func (e *Exponential) Checkpoint(w, t_p time.Duration) (bool, error) {

	// to get stats a checkpoint is required
	// - stats are only sent when a checkpoint is completed
	if e.stat == nil {
		return true, nil
	}

	take, err := e.take(w, t_p)
	if err != nil {
		return false, err
	}

	skip, err := e.skip(w, t_p)
	if err != nil {
		return false, err
	}

	return take < skip, nil
}

func (e *Exponential) StatUpdate(s *Stat) error {
	e.Stat = s
	return nil
}

func (e *Exponential) PriceUpdate(p float64, _ time.Time) error {
	e.Price = p

	return nil
}
