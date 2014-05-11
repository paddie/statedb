package statedb

import (
	// "errors"
	"fmt"
	// "github.com/paddie/statedb/monitor"
	"time"
)

type Model interface {
	Name() string
	// Before calling Start, the model is called with
	// a trace of the past 3 months of the specific instance
	// type
	Train(trace []PricePoint, bid float64) error
	PriceUpdate(cost float64, datetime time.Time) error
	StatUpdate(restartTimeMinutes, checkpointTimeMinutes float64) error
	Checkpoint(minutesSinceCpt, avgSyncTimeMinutes, AliveMinutes float64) (bool, error)
	Quit() error
}

type ModelNexus struct {
	errChan      chan error
	statChan     chan *Stat
	cptQueryChan chan *CheckpointQuery
	quitChan     chan bool
}

func NewModelNexus() *ModelNexus {
	return &ModelNexus{
		statChan:     make(chan *Stat, 1),
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

	to := time.Now()
	from := to.AddDate(0, -3, 0)

	trace, err := monitor.Trace(from, to)
	if err != nil {
		nx.errChan <- err
		return
	}

	if err := model.Train(trace, bid); err != nil {
		nx.errChan <- err
		return
	}

	C, errChan := monitor.Start(5 * time.Minute)

	fmt.Printf("Starting model <%s>\n", model.Name())

	for {
		select {
		case q := <-nx.cptQueryChan:
			do, err := model.Checkpoint(
				q.s.RestoreTime(),        // time to restore
				q.s.ExpWriteCheckpoint(), // time to cpt
				q.s.AvgSyncMinutes())     // avg. sync time
			if err != nil {
				nx.errChan <- err
			}
			q.cptChan <- do
		case pp := <-C:
			err := model.PriceUpdate(pp.SpotPrice, pp.TimeStamp)
			if err != nil {
				nx.errChan <- err
			}
		case s := <-nx.statChan:
			err := model.StatUpdate(s.RestoreTime(), s.ExpWriteCheckpoint())
			if err != nil {
				nx.errChan <- err
			}
		case _ = <-nx.quitChan:
			// shut down monitor..
			monitor.Stop()
			// shut down model..
			err := model.Quit()
			if err != nil {
				nx.errChan <- err
			}
			fmt.Println("Exiting from Model")
			return
		case err := <-errChan:
			nx.errChan <- err
			err = model.Quit()
			if err != nil {
				nx.errChan <- err
			}
			fmt.Println("Exiting from Model")
			return
		}
	}
}
