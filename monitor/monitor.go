package monitor

import (
	"fmt"
	// "github.com/titanous/goamz/aws"
	"github.com/paddie/goamz/ec2"
	"time"
)

type Monitor struct {
	s       *ec2.EC2              // ec2 server credentials
	request *ec2.SpotPriceRequest // base request
	filter  *ec2.Filter
	// lastUpdated time.Time       // time of last update
	C        chan PricePoint // channel for pricepoints
	quitChan chan bool       // channel to signal exit
	ticker   *time.Ticker    // ticks every 'duration'
	Latest   *PricePoint     // last value that was changed the
}

func NewMonitor(s *EC2Instance, interval time.Duration) (*Monitor, error) {

	if interval < time.Minute {
		return nil, fmt.Errorf("Monitor: Update interval cannot be less than one minute %v", interval)
	}

	m := &Monitor{
		s:        s.ec2,
		filter:   s.filter,
		request:  s.request,
		quitChan: make(chan bool),
		C:        make(chan PricePoint),
		ticker:   time.NewTicker(interval),
	}

	go m.ChangeMonitor()

	return m, nil
}

// Blocking Shutdown of the Monitor
func (m *Monitor) QuitBlock() {
	if m == nil {
		return
	}
	m.quitChan <- true

	<-m.quitChan
}

// Only sends prices when they
func (m *Monitor) ChangeMonitor() {

	from := time.Now()

	for {
		select {
		case to := <-m.ticker.C:
			// use first tick to initialize the

			// copy the request object
			r := *m.request
			r.StartTime = from
			r.EndTime = to

			// retrieve interformation
			items, err := getSpotPriceHistory(m.s, &r, m.filter)
			if err != nil {
				fmt.Println("PriceMonitor: ", err.Error())
				continue
			}

			// only send item if it is newer AND different price
			// - spot prices arrive from new -> old
			//   so we reverse-iterate through them
			for i := len(items) - 1; i >= 0; i-- {
				item := items[i]

				pp := &PricePoint{
					SpotPrice: item.SpotPrice,
					TimeStamp: item.Timestamp,
					Key:       item.Key(),
				}
				if m.Latest == nil || (pp.SpotPrice != m.Latest.SpotPrice &&
					m.Latest.TimeStamp.Before(pp.TimeStamp)) {

					m.Latest = pp
					m.C <- *pp
				}
			}
		case <-m.quitChan:
			// stop ticker
			m.ticker.Stop()
			// close trace channel
			close(m.C)
			// send signal that cleanup is complete
			m.quitChan <- true
			return
		}

	}
}
