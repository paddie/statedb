package schedular

import (
	// "errors"
	"fmt"
	"github.com/paddie/statedb"
	"time"
)

type PreemptRising struct {
	price   float64
	risen   bool
	preempt <-chan bool
	errchan <-chan error
}

func NewPreemptRising() *PreemptRising {
	return &PreemptRising{}
}

func (r *PreemptRising) Name() string {
	return "RisingEdge"
}

func (r *PreemptRising) Preemtive() bool {
	return true
}

func (r *PreemptRising) Start(preempt chan<- bool, errchan chan<- error) error {

	if preempt == nil {
		return fmt.Errorf("Did not receive a preempt channel")
	}

	if errchan == nil {
		return fmt.Errorf("Did not receive an error channel")
	}

	r.signal = preempt
	r.errchan = errchan

	return nil
}

func (r *PreemptRising) Train(trace []statedb.PricePoint, _ float64) error {

	if len(trace) == 0 {
		return nil
	}

	r.price = trace[len(trace)-1].Price()

	return nil
}

func (r *PreemptRising) StatUpdate(_ *statedb.Stat) error {

	return nil
}

func (r *PreemptRising) PriceUpdate(p float64, _ time.Time) error {
	if r.price < p {
		fmt.Printf("<Rising> %.4f --> %.4f: checkpoint at next sync\n", r.price, p)
		// fmt.Println("signalling checkpoint at next sync")
		r.risen = true
	} else {
		fmt.Printf("<Rising> %.4f --> %.4f: no checkpoint at sync\n", r.price, p)
		r.risen = false
	}
	r.price = p

	return nil
}

func (r *PreemptRising) Checkpoint(_ *statedb.Stat) (bool, error) {
	if r.risen {
		r.risen = false
		return true, nil
	}
	return false, nil
}

func (r *PreemptRising) Quit() error {
	return nil
}
