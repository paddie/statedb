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
	"os/exec"
	"path/filepath"
	"strings"
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

var (
	path       string
	price      bool
	commit     bool
	syncs      bool
	model      bool
	monitor    bool
	checkpoint bool
)

func init() {
	flag.StringVar(&path, "p", "", "path to json dump of a *Trace object")
	flag.BoolVar(&price, "price", false, "plot price changes")
	flag.BoolVar(&commit, "commit", false, "plot cpts times and durations")
	flag.BoolVar(&syncs, "sync", false, "plot sync times and durations")
	flag.BoolVar(&model, "model", false, "plot model times and durations")
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

	// if !syncs && !commit && !price {
	// 	fmt.Println("No data flags were set!")
	// 	return
	// }

	// if price {
	for i, v := range tl.PriceChanges {
		tl.PriceChanges[i] = v * 2000
	}

	pc, err := NewPriceTraces(tl.PriceTimes, tl.PriceChanges)
	if err != nil {
		panic(err)
	}
	// fmt.Println("Adding Price Traces...")
	// err = plotutil.AddLinePoints(p,
	// "price changes", pc)
	// if err != nil {
	// 	panic(err)
	// }
	// }

	// if syncs {
	fmt.Println("Adding Sync events")
	sp, err := NewXYer(tl.SyncStarts, tl.SyncDurations)
	if err != nil {
		panic(err)
	}
	// err = plotutil.AddLinePoints(p, "sync", sp)
	// if err != nil {
	// 	panic(err)
	// }
	// }

	// if commit {
	fmt.Println("Adding Commit events")

	cp, err := NewXYer(tl.CommitStarts, tl.CommitDurations)
	if err != nil {
		panic(err)
	}

	// err = plotutil.AddLinePoints(p, "commits", cp)
	// if err != nil {
	// 	panic(err)
	// }
	// }

	err = plotutil.AddLinePoints(p,
		"sync", sp,
		"commits", cp,
		"price changes", pc)
	if err != nil {
		panic(err)
	}

	path = strings.Replace(path, " ", "", -1)

	if err := p.Save(8, 4, path+".pdf"); err != nil {
		panic(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	file := filepath.Join(wd, path+".pdf")
	fmt.Println(file)
	err = exec.Command("/usr/bin/open", file).Run()
	if err != nil {
		fmt.Println(err.Error())
	}
}
