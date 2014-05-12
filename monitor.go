package statedb

import (
	"time"
)

const (
	MONTH = 60 * 24 * 30
	YEAR  = MONTH * 12
)

type Monitor interface {
	Trace(from, to time.Time) ([]PricePoint, error)
	Start(interval time.Duration) (chan PricePoint, chan error)
	Stop()
	Active() bool
}

// type PricePoint interface {
// 	SpotPrice() float64
// 	TimeStamp() time.Time
// 	Key() string
// }
