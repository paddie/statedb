package statedb

import (
	"time"
)

type PricePoint struct {
	spotPrice float64
	timeStamp time.Time
	key       string
}

func (p PricePoint) SpotPrice() float64 {
	return p.spotPrice
}

func (p PricePoint) TimeStamp() time.Time {
	return p.timeStamp
}

func (p PricePoint) Key() string {
	return p.key
}
