package statedb

import (
	// "errors"
	"fmt"
	// "github.com/paddie/statedb/monitor"
	"time"
)

type Schedular interface {
	Name() string
	Train(trace []PricePoint, bid float64) error
	Start(chan<- bool, chan<- error) error
	PriceUpdate(cost float64, datetime time.Time) error
	StatUpdate(stat *Stat) error
	Checkpoint(stat *Stat) (bool, error)
	Quit() error
	Preemptive() bool
}

type event int

const (
	CHECKPOINTCOMMIT event = iota
	STATEREGISTER
	STATEUNREGISTER
	POINTOFCONSISTENCY
)

type SchedEvent interface {
	Event() event
}

type ModelNexus struct {
	errChan      chan error
	statChan     chan Stat
	cptQueryChan chan *CheckpointQuery
	quitChan     chan bool
	preemptChan  chan bool
	preempt      bool
}

func NewModelNexus(preempt bool) *ModelNexus {
	mnx := &ModelNexus{
		statChan:     make(chan Stat, 1),
		cptQueryChan: make(chan *CheckpointQuery),
		errChan:      make(chan error),
		quitChan:     make(chan bool),
	}

	if preempt {
		mnx.preemptChan = make(chan bool)
		mnx.preempt = true
	}

	return mnx
}

func (nx *ModelNexus) Quit() {
	// send quit signal to model
	nx.quitChan <- true
	// shut down all sending channels..
	close(nx.statChan)
	close(nx.cptQueryChan)
	close(nx.quitChan)
}

func educate(sched Schedular, monitor Monitor, nx *ModelNexus, bid float64) {

	// Collect trace from the monitor
	trace, err := monitor.Trace()
	if err != nil {
		nx.errChan <- err
		return
	}

	if err := sched.Train(trace, bid); err != nil {
		nx.errChan <- err
		return
	}

	fmt.Printf("Starting schedular: <%s>\n", sched.Name())
	schedErrChan := make(chan error)
	if err := sched.Start(nx.preemptChan, schedErrChan); err != nil {
		nx.errChan <- fmt.Errorf("Schedular: %s", err.Error())
		return
	}

	fmt.Printf("Starting monitor: <%s>\n", monitor.Name())
	// price-change channel
	C := make(chan PricePoint)
	// error channel for monitor
	monErrChan := make(chan error)

	if err = monitor.Start(C, monErrChan); err != nil {
		// shuts down the schedular etc.
		go reportError(monErrChan, err)
	}

	for {
		select {
		case q := <-nx.cptQueryChan:
			do, err := model.Checkpoint(q.s) // avg. sync time
			if err != nil {
				nx.errChan <- err
			}
			if do {
				fmt.Println("Schedular: take checkpoint!")
			} else {
				fmt.Println("Scheduar: skip checkpoint")
			}

			q.cptChan <- do
		case pp := <-C:
			err := model.PriceUpdate(pp.Price(), pp.Time())
			timeline.PriceChange(pp.Price())
			if err != nil {
				nx.errChan <- err
			}
		case s := <-nx.statChan:
			err := model.StatUpdate(&s)
			if err != nil {
				nx.errChan <- err
			}
		case _ = <-nx.quitChan:
			// shut down monitor..
			monitor.Stop()
			fmt.Println("Monitor has been shut down")
			// shut down model..
			err := model.Quit()
			if err != nil {
				nx.errChan <- err
			}
			fmt.Println("Model has been shut down")
			return
		case err := <-monErrChan:
			err = fmt.Errorf("Monitor panicked: <%s>", err.Error())

			fmt.Println(err.Error())
			// send error to stateloop
			nx.errChan <- err
			// quit schedular
			_ = sched.Quit()
			fmt.Println("Schedular has been shut down")
			return
		case err := <-schedErrChan:
			err = fmt.Errorf("Schedular panicked: <%s>", err.Error())
			fmt.Println(err.Error())
			// send error to stateloop
			nx.errChan <- err
			// quit schedular
			_ = monitor.Stop()
			fmt.Println("Monitor has been shut down")
			return
		}
	}
}

func reportError(errChn <-chan error, err error) {
	errChn <- err
	return
}
