package statedb

import (
	// "errors"
	"fmt"
	// "github.com/paddie/statedb"
	"time"
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

func (r *RisingEdge) Train(trace []PricePoint, _ float64) error {

	if len(trace) > 0 {
		return nil
	}

	r.price = trace[len(trace)-1].SpotPrice()

	return nil
}

func (r *RisingEdge) StatUpdate(_, _ float64) error {
	return nil
}

func (r *RisingEdge) PriceUpdate(p float64, _ time.Time) error {
	fmt.Printf("PriceChange: %.4f --> %.4f\n", r.price, p)
	if r.price < p {
		fmt.Println("Signalling checkpoint at next sync")
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
