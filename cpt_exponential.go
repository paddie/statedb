package statedb

import (
	// "errors"
	"github.com/paddie/statedb/monitor"
	"time"
)

type Exponential struct {
	Mean         float64
	Lambda       float64
	Bid          float64
	Stat         Stat
	Price        float64
	T_Restore    float64 // minutes
	T_Checkpoint float64 // minutes
}

func NewExponential() *Exponential {
	return &Exponential{}
}

func (e *Exponential) Name() string {
	return "ExponentialDistribution"
}

func (e *Exponential) take(t, t_p float64) float64 {
	return 0.0
}

func (e *Exponential) skip(t, t_p float64) float64 {
	return 0.0
}

func (e *Exponential) t(w, t_p float64) float64 {
	return 0.0
}

func (e *Exponential) Train(trace *monitor.Trace, bid float64) error {
	e.Mean, _, _, _ = trace.AliveStats(bid)
	e.Lambda = 1.0 / e.Mean
	e.Price = trace.Latest.SpotPrice
	return nil
}

func (e *Exponential) Checkpoint(w_dur, t_p_dur time.Duration) (bool, error) {

	// to get stats a checkpoint is required
	// - stats are only sent when a checkpoint is completed
	if e.Stat == nil {
		return true, nil
	}

	w := w_dur.Minutes()
	t_p := t_p_dur.Minutes()

	return e.take(w, t_p) < e.skip(w, t_p), nil
}

func (e *Exponential) StatUpdate(s Stat) error {
	e.Stat = s

	e.T_Restore = s.LastCpt

	return nil
}

func (e *Exponential) PriceUpdate(p float64, _ time.Time) error {
	e.Price = p

	return nil
}
