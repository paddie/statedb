package testmonitor

import (
	"github.com/paddie/statedb"
	// "github.com/paddie/statedb/fs"
	"time"
)

type PricePoint struct {
	spotPrice float64
	timeStamp time.Time
	key       string
}

func (p *PricePoint) Price() float64 {
	return p.spotPrice
}

func (p *PricePoint) Time() time.Time {
	return p.timeStamp
}

type TestMonitor struct {
	points   []PricePoint
	tick     *time.Ticker
	active   bool
	interval time.Duration
}

func NewTestMonitor(interval time.Duration) *TestMonitor {

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

	tm := &TestMonitor{
		interval: interval,
	}

	now := time.Now()
	for i := 0; i < len(prices); i++ {
		now = now.Add(interval)

		tm.points = append(tm.points, &PP{
			spotPrice: prices[i],
			timeStamp: now,
			key:       "test.key",
		})
	}
	return tm
}

func (t *TestMonitor) Trace(_, _ time.Time) ([]statedb.PricePoint, error) {
	return t.points, nil
}

// ignore the default interval 5 * time.Minute()
func (t *TestMonitor) Start(_ time.Duration) (chan Interface, chan error) {

	priceChan := make(chan PricePoint)

	t.tick = time.NewTicker(t.interval)
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
func pitcher(c <-chan time.Time, p []statedb.PricePoint, priceChan chan statedb.PricePoint) {
	i := 0
	for _ = range c {
		if i == len(p) {
			i = 0
		}
		priceChan <- p[i]
		i++
	}
}
