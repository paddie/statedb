package statedb

import (
	// "errors"
	"fmt"
	"github.com/paddie/statedb/monitor"
	"time"
)

type Model interface {
	Name() string
	// Before calling Start, the model is called with
	// a trace of the past 3 months of the specific instance
	// type
	Train(t *monitor.Trace, bid float64) error
	PriceUpdate(price float64) error
	StatUpdate(stat Stat) error
	Checkpoint(running_t, t_p time.Duration) (bool, error)
	Quit() error
	// Init takes a channel of *Stat updates
	// and a channel of PricePoint updates.
	// - A stat is sent when:
	//     1) A Zero- or Delta checkpoint has been committed
	//     2) An object was inserted or removed
	// - A PricePoint is sent when the monitor detects a price
	// change
	// Checkpoint takes two arguments:
	// 1) time since the last checkpoint, eg. the amount of work
	//    that has to be recomputed in case of a crash
	// 2) the average time between consistent states
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

func Educate(model Model, s *monitor.EC2Instance, nx *ModelNexus) {

	to := time.Now()
	from := to.AddDate(0, -3, 0)

	trace, err := s.GetPriceTrace(from, to)
	if err != nil {
		nx.errChan <- err
		return
	}

	if err := model.Train(trace); err != nil {
		nx.errChan <- err
		return
	}

	m, err := monitor.NewMonitor(s, 5*time.Minute)
	if err != nil {
		nx.errChan <- err
		return
	}

	fmt.Printf("Starting model <%s>\n", m.Name())

	for {
		select {
		case q := <-nx.cptQueryChan:
			do, err := model.Checkpoint(
				q.t_processed,
				q.t_avg_sync)
			if err != nil {
				nx.errChan <- err
			}
			q.cptChan <- do
		case p := <-nx.priceChan:
			err := model.PriceUpdate(p)
			if err != nil {
				nx.errChan <- err
			}
		case s := <-statChan:
			err := model.StatUpdate(s)
			if err != nil {
				nx.errChan <- err
			}
		case _ = <-nx.quitChan:
			// shut down monitor..
			m.QuitBlock()
			// shut down model..
			err := model.Quit()
			if err != nil {
				nx.errChan <- err
			}
			fmt.Println("Exiting from Model")
			return
		}
	}
}
