package statedb

import (
	// "errors"
	"github.com/paddie/statedb/monitor"
	// "time"
)

type RisingEdge struct {
	price float64
	risen bool
}

func NewRisingEdge() *RisingEdge {
	return &RisingEdge{}
}

func (r *RisingEdge) Name() string {
	return "RisingEdge"
}

func (r *RisingEdge) Train(trace *monitor.Trace, _ float64) error {

	r.price = trace.Latest.SpotPrice

	return nil
}

func (r *RisingEdge) StatUpdate(_, _ float64) error {
	return nil
}

func (r *RisingEdge) PriceUpdate(p float64, _ float64) error {

	if r.price < p {
		r.risen = true
	} else {
		r.risen = false
	}
	r.price = p

	return nil
}

func (r *RisingEdge) Checkpoint(_, _, _ float64) (bool, error) {
	if r.risen {
		r.risen = false
		return true, nil
	}
	return false, nil
}

func (r *RisingEdge) Quit() error {
	return nil
}
