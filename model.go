package statedb

import (
	// "errors"
	"fmt"
	"github.com/paddie/statedb/monitor"
	"time"
)

type CheckpointQuery struct {
	t_processed time.Duration
	t_avg_sync  time.Duration
	cptChan     chan bool
}

type Model interface {
	Name() string
	// Before calling Start, the model is called with
	// a trace of the past 3 months of the specific instance
	// type
	Train(*monitor.Trace) error
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
	Start(statChan <-chan Stat, priceChan <-chan monitor.PricePoint, cptQueryChan <-chan *CheckpointQuery, quit chan bool, errChan chan<- error)
}

func Educate(m Model, s *monitor.EC2Instance, nx *ModelNexus) {

	to := time.Now()
	from := to.AddDate(0, -3, 0)

	trace, err := s.GetPriceTrace(from, to)
	if err != nil {
		nx.errChan <- err
		return
	}

	if err := m.Train(trace); err != nil {
		nx.errChan <- err
		return
	}

	nx.monitor, err = monitor.NewMonitor(s, 5*time.Minute)
	if err != nil {
		nx.errChan <- err
		return
	}

	// enter model loop
	fmt.Printf("Starting model <%s>\n", m.Name())

	m.Start(nx.statChan, nx.monitor.C, nx.cptQueryChan, nx.quitChan, nx.errChan)
}
