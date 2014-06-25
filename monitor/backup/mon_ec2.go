package statedb

import (
	"fmt"
	"github.com/paddie/goamz/ec2"
	"time"
)

// type PricePoint struct {
// 	spotPrice float64
// 	timeStamp time.Time
// 	key       string
// }

// func (p *PricePoint) Price() float64 {
// 	return p.spotPrice
// }

// func (p *PricePoint) Time() time.Time {
// 	return p.timeStamp
// }

type EC2Monitor struct {
	request  *ec2.SpotPriceRequest
	filter   *ec2.Filter
	ec2      *ec2.EC2
	quit     chan bool
	active   bool
	from, to time.Time
}

func NewEC2Monitor(s *ec2.EC2, instanceType, productDescription, availabilityZone string, filter *ec2.Filter) (*EC2Monitor, error) {

	if filter == nil && (instanceType == "" ||
		productDescription == "" ||
		availabilityZone == "") {
		return nil, fmt.Errorf(`Empty description parameters:
    InstanceType:       '%s'
    ProductDescription: '%s'
    AvailabilityZone:   '%s'`, instanceType, productDescription, availabilityZone)
	}

	request := &ec2.SpotPriceRequest{
		InstanceType:       instanceType,
		ProductDescription: productDescription,
		AvailabilityZone:   availabilityZone,
	}

	desc := &EC2Monitor{
		request: request,
		ec2:     s,
		filter:  filter,
		quit:    make(chan bool),
	}

	return desc, nil
}

func (s *EC2Monitor) Start(interval time.Duration) (chan *PricePoint, chan error) {

	// start any previous monitor
	if s.active {
		s.Stop()
	}

	c := make(chan *PricePoint)
	errChan := make(chan error)

	go monitor(s, interval, c, errChan)

	s.active = true

	return c, errChan
}

func (s *EC2Monitor) Stop() {
	s.quit <- true
	_ = <-s.quit
	s.active = false
}

func (s *EC2Monitor) Active() bool {
	return s.active
}

// Only sends prices when they
func monitor(s *EC2Monitor, interval time.Duration, c chan *PricePoint, errChan chan error) {

	from := time.Now()

	tick := time.NewTicker(interval)

	var latest *PricePoint

	for {
		select {
		case to := <-tick.C:
			// use first tick to initialize the

			// copy the request object
			r := *s.request
			r.StartTime = from
			r.EndTime = to

			// retrieve information
			items, err := s.ec2.SpotPriceHistory(&r, s.filter)
			if err != nil {
				tick.Stop()
				close(c)
				errChan <- err
				return
			}

			// only send item if it is newer AND different price
			// - spot prices arrive from new -> old
			//   so we reverse-iterate through them
			for i := len(items) - 1; i >= 0; i-- {
				item := items[i]

				pp := &PricePoint{
					spotPrice: item.SpotPrice,
					timeStamp: item.Timestamp,
					key:       item.Key(),
				}
				if latest == nil || (pp.spotPrice != latest.spotPrice &&
					latest.timeStamp.Before(pp.timeStamp)) {

					latest = pp
					c <- pp
				}
			}
		case <-s.quit:
			// stop ticker
			tick.Stop()
			// close trace channel
			close(c)
			// send signal that cleanup is complete
			s.quit <- true
			return
		}
	}
}

func (s *EC2Monitor) Key() string {
	return fmt.Sprintf("%s.%s.%s", s.request.AvailabilityZone, s.request.InstanceType, s.request.ProductDescription)
}

func (s *EC2Monitor) Trace(from, to time.Time) ([]PricePoint, error) {
	// to := time.Now()

	// date must be non-zero
	if from.IsZero() {
		return nil, fmt.Errorf("from-date '%v' is zero", from, to)
	}
	// from-date must be before to-date
	if !to.After(from) {
		return nil, fmt.Errorf("From-date '%v' is before to-date '%v'", from, to)
	}
	// if the difference is more than one month
	now := time.Now()
	if from.Before(now.AddDate(0, -4, 0)) {
		return nil, fmt.Errorf("from-date exceeds the 4 month limit")
	}

	r := *s.request

	r.StartTime = from
	r.EndTime = to

	items, err := s.ec2.SpotPriceHistory(&r, s.filter)
	if err != nil {
		return nil, err
	}

	// reverse the order from:
	// new >> old
	// old >> new
	key := s.Key()
	pp := make([]PricePoint, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		pp[i] = PricePoint{
			spotPrice: item.SpotPrice,
			timeStamp: item.Timestamp,
			key:       key,
		}
	}
	return pp, nil
}
