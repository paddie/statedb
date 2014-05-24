package statedb

import (
	"time"
)

const (
	MONTH = 60 * 24 * 30
	YEAR  = MONTH * 12
)

type PricePoint struct {
	spotPrice float64
	timeStamp time.Time
	key       string
}

func (p PricePoint) Price() float64 {
	return p.spotPrice
}

func (p PricePoint) Time() time.Time {
	return p.timeStamp
}

func (p PricePoint) Key() string {
	return p.key
}

type Monitor interface {
	Trace(from, to time.Time) ([]PricePoint, error)
	Start(interval time.Duration) (chan PricePoint, chan error)
	Stop()
	Active() bool
}
