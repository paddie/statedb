package statedb

import (
	// "testing"
	"time"
)

type TestMonitor struct {
	points []PricePoint
	tick   *time.Ticker
	active bool
}

func NewTestMonitor() *TestMonitor {

	prices := []float64{
		1.2,
		1.3,
		1.0,
		1.5,
		1.1,
		0.9,
		0.8,
		1.3,
		1.6,
		0.7,
		0.6,
		0.7,
		1.9,
		2.0,
		1.5,
		1.2,
		1.3,
		1.4,
		1.1,
		1.0,
	}

	tm := &TestMonitor{}

	now := time.Now()
	for i := 0; i < len(prices); i++ {
		now = now.Add(time.Minute * 5)

		tm.points = append(tm.points, PricePoint{
			spotPrice: prices[i],
			timeStamp: now,
			key:       "test.key",
		})
	}

	return tm
}

func (t *TestMonitor) Trace(_, _ time.Time) ([]PricePoint, error) {
	return t.points, nil
}

func (t *TestMonitor) Start(interval time.Duration) (chan PricePoint, chan error) {

	priceChan := make(chan PricePoint)

	t.tick = time.NewTicker(interval)
	go pitcher(t.tick.C, t.points, priceChan)

	t.active = true

	return priceChan, nil
}

func (t *TestMonitor) Stop() {
	t.tick.Stop()
	// close(t.tick.C)
	t.active = false
}

func (t *TestMonitor) Active() bool {
	return t.active
}

// will loop until we call tick.Stop() on the ticker
func pitcher(c <-chan time.Time, p []PricePoint, priceChan chan PricePoint) {
	i := 0
	for _ = range c {
		if i == len(p) {
			i = 0
		}
		priceChan <- p[i]
		i++
	}
}
