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
	"os"
	// "time"
)

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

	p.Title.Text = "Trace"
	p.X.Label.Text = "sync/ms"
	p.Y.Label.Text = "block/ms"

	if err = plotutil.AddLinePoints(p,
		"trace", tl); err != nil {
		panic(err)
	}

	// Save the plot to a PNG file.
	if err := p.Save(8, 4, path+".png"); err != nil {
		panic(err)
	}

}
