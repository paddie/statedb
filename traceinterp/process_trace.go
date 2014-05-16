package main

import (
	// "bytes"
	"code.google.com/p/plotinum/plot"
	// "code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/plotutil"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/paddie/statedb"
	"math"
	"os"
	"time"
)

func NanToMill(nanoseconds int64) float64 {
	return math.Ceil(float64(nanoseconds) / 1000000.0)
}

type PriceTraces struct {
	Xs []time.Duration
	Ys []float64
}

func NewPriceTraces(x []time.Duration, y []float64) (*PriceTraces, error) {
	if len(x) != len(y) {
		return nil, fmt.Errorf("Uneven slice sizes: Xs[0:%d] and Ys[0:%d]", len(x), len(y))
	}
	return &PriceTraces{
		Xs: x,
		Ys: y,
	}, nil
}

func (pt *PriceTraces) XY(i int) (float64, float64) {
	return NanToMill(pt.Xs[i].Nanoseconds()), pt.Ys[i]
}

func (pt *PriceTraces) Len() int {
	return len(pt.Xs)
}

type XYer struct {
	Xs []time.Duration
	Ys []time.Duration
}

func NewXYer(x, y []time.Duration) (*XYer, error) {
	if len(x) != len(y) {
		return nil, fmt.Errorf("Uneven Xs[0:%d] and Ys[0:%d]", len(x), len(y))
	}

	return &XYer{
		Xs: x,
		Ys: y,
	}, nil
}

func (xy *XYer) Len() int {
	return len(xy.Xs)
}

func (xy *XYer) XY(i int) (float64, float64) {
	return NanToMill(xy.Xs[i].Nanoseconds()), NanToMill(xy.Ys[i].Nanoseconds())
}

var path string

// var bid float64

func init() {
	flag.StringVar(&path, "p", "", "path to json dump of a *Trace object")
	// flag.Float64Var(&bid, "b", 0.0, "define initial bid")
}

func main() {

	flag.Parse()

	if path == "" {
		fmt.Println("No path provided")
		os.Exit(1)
	}

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	tl := &statedb.TimeLine{}
	dec := json.NewDecoder(f)

	if err := dec.Decode(tl); err != nil {
		panic(err)
	}

	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = path
	p.X.Label.Text = "timeline/ms"
	p.Y.Label.Text = "event duration/ms"

	pc, err := NewPriceTraces(tl.PriceTimes, tl.PriceChanges)
	if err != nil {
		panic(err)
	}

	sp, err := NewXYer(tl.SyncStarts, tl.SyncDurations)
	if err != nil {
		panic(err)
	}

	cp, err := NewXYer(tl.CommitStarts, tl.CommitDurations)
	if err != nil {
		panic(err)
	}

	err = plotutil.AddLinePoints(p,
		"priceChanges", pc,
		"SyncPoints", sp,
		"commit", cp)
	if err != nil {
		panic(err)
	}

	// Save the plot to a PNG file.
	if err := p.Save(8, 4, path+".png"); err != nil {
		panic(err)
	}

}
