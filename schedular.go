package statedb

import (
	// "errors"
	"fmt"
	// "github.com/paddie/statedb/monitor"
	"time"
)

type Model interface {
	Name() string
	Train(trace []PricePoint, bid float64) error
	PriceUpdate(cost float64, datetime time.Time) error
	StatUpdate(stat Stat) error
	PointOfConsistency() (bool, error)
	Quit() error
}

type ModelNexus struct {
	errChan      chan error
	statChan     chan Stat
	cptQueryChan chan *CheckpointQuery
	quitChan     chan bool
}

func NewModelNexus() *ModelNexus {
	return &ModelNexus{
		statChan:     make(chan Stat, 1),
		cptQueryChan: make(chan *CheckpointQuery),
		errChan:      make(chan error),
		quitChan:     make(chan bool),
	}
}

func (nx *ModelNexus) Quit() {
	// send quit signal to model
	nx.quitChan <- true
	// shut down all sending channels..
	close(nx.statChan)
	close(nx.cptQueryChan)
	close(nx.quitChan)
}

func educate(model Model, monitor Monitor, nx *ModelNexus, bid float64) {

	trace, err := monitor.Trace()
	if err != nil {
		nx.errChan <- err
		return
	}

	if err := model.Train(trace, bid); err != nil {
		nx.errChan <- err
		return
	}

	priceChan := make(chan PricePoint)
	errChan := make(chan error)

	if err = monitor.Start(priceChan, errChan); err != nil {
		nx.errChan <- err
		return
	}

	fmt.Printf("Starting model <%s>\n", model.Name())

	for {
		select {
		case q := <-nx.cptQueryChan:
			do, err := model.PointOfConsistency()
			if err != nil {
				nx.errChan <- err
			}
			if do {
				fmt.Println("Schedular: take checkpoint!")
			} else {
				fmt.Println("Scheduar: skip checkpoint")
			}
			q.cptChan <- do
		case pp := <-priceChan:
			err := model.PriceUpdate(pp.Price(), pp.Time())
			timeline.PriceChange(pp.Price())
			if err != nil {
				nx.errChan <- err
			}
		case s := <-nx.statChan:
			err := model.StatUpdate(s)
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
		case err := <-errChan:
			fmt.Printf("Monitor panicked: <%s>", err.Error())
			nx.errChan <- err
			err = model.Quit()
			if err != nil {
				nx.errChan <- err
			}
			fmt.Println("Model has been shut down")
			return
		}
	}
}
