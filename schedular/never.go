package schedular

import (
	// "errors"
	// "fmt"
	"github.com/paddie/statedb"
	"time"
)

type Never struct{}

func NewNever() *Never {
	return &Never{}
}

func (r *Never) Name() string {
	return "Never"
}

func (r *Never) Train(_ []statedb.PricePoint, _ float64) error {
	return nil
}

func (r *Never) StatUpdate(stat statedb.Stat) error {
	return nil
}

func (r *Never) PriceUpdate(_ float64, _ time.Time) error {
	return nil
}

func (r *Never) PointOfConsistency() (bool, error) {
	return false, nil
}

func (r *Never) Quit() error {
	return nil
}
