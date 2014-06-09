package schedular

import (
	// "errors"
	// "fmt"
	"github.com/paddie/statedb"
	"time"
)

type Always struct{}

func NewAlways() *Always {
	return &Always{}
}

func (r *Always) Name() string {
	return "Always"
}

func (r *Always) Train(_ []statedb.PricePoint, _ float64) error {
	return nil
}

func (r *Always) StatUpdate(stat statedb.Stat) error {
	return nil
}

func (r *Always) PriceUpdate(_ float64, _ time.Time) error {
	return nil
}

func (r *Always) PointOfConsistency() (bool, error) {
	return true, nil
}

func (r *Always) Quit() error {
	return nil
}
