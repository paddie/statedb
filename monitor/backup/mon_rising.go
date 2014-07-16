package statedb

import (
	// "testing"
	"time"
)

type RisingMonitor struct {
	start    float64
	current  float64
	tick     *time.Ticker
	active   bool
	interval time.Duration
}

func NewRisingMonitor(interval time.Duration) *RisingMonitor {

	tm := &RisingMonitor{
		interval: interval,
		start:    0.5,
		current:  0.5,
	}

	return tm
}

func (t *RisingMonitor) Trace(_, _ time.Time) ([]PricePoint, error) {

	return nil, nil
}

// ignore the default interval 5 * time.Minute()
func (t *RisingMonitor) Start(_ time.Duration) (chan PricePoint, chan error) {

	priceChan := make(chan PricePoint)
	t.tick = time.NewTicker(t.interval)
	go risingPitcher(t.tick.C, t.start, priceChan)
	t.active = true
	return priceChan, nil
}

func (t *RisingMonitor) Stop() {
	t.tick.Stop()
	// close(t.tick.C)
	t.active = false
}

func (t *RisingMonitor) Active() bool {
	return t.active
}

// will loop until we call tick.Stop() on the ticker
func risingPitcher(c <-chan time.Time, start float64, priceChan chan PricePoint) {
	inc := 0.001
	for now := range c {
		priceChan <- PricePoint{
			spotPrice: start,
			timeStamp: now,
			key:       "Rising.Key",
		}
		start += inc
	}
}
