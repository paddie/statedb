package statedb

import (
	"errors"
	"github.com/paddie/statedb/monitor"
)

type RisingEdge struct {
	price float64
	risen bool
}

func NewRisingEdge() *RisingEdge {
	return
}

func (r *RisingEdge) Name() string {
	return "RisingEdge"
}

func (r *RisingEdge) Train(trace *monitor.Trace, _ float64) error {

	r.price = trace.Latest.SpotPrice

	return nil
}

func (r *RisingEdge) StatUpdate(_ Stat) error {
	return nil
}

func (r *RisingEdge) PriceUpdate(p float64, _ time.Time) error {

	if r.price < p {
		r.risen = true
	} else {
		r.risen = false
	}
	r.price = p

	return nil
}

func (r *RisingEdge) Checkpoint(_, _ time.Duration) (bool, error) {
	if r.risen {
		r.risen = false
		return true, nil
	}

	return false, nil
}

func (r *RisingEdge) Quit() error {
	return nil
}
