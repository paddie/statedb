package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grd/stat"
	// mon "github.com/paddie/ec2spotmonitor"
	"github.com/paddie/goamz/ec2"
	"math"
	"os"
	"time"
)

var (
	OutOfSequenceError = errors.New("OutOfSequenceError: SpotPriceItem predates the prevously added item")
)

type Trace struct {
	Key            string
	From, To       time.Time
	Max, Min       float64
	Duration       time.Duration
	latest         *PricePoint
	Prices         []float64 // Cost
	AbsPriceChange []float64 // price change
	PriceChange    []float64
	Times          []time.Time // ChangeTime
	TimeChanges    []float64   // minutes - time since last change
}

type PricePoint struct {
	SpotPrice float64
	TimeStamp time.Time
	Key       string
}

func (t *Trace) AddAll(items []*ec2.SpotPriceItem) error {

	var err error

	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		err = t.Add(item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Trace) Add(item *ec2.SpotPriceItem) error {

	if t.latest == nil {
		t.From = item.Timestamp
		t.To = item.Timestamp
		t.Duration = time.Second * 0
		t.Max = item.SpotPrice
		t.Min = item.SpotPrice

		// save as reference for later changes
		t.latest = &PricePoint{
			SpotPrice: item.SpotPrice,
			TimeStamp: item.Timestamp,
			Key:       item.Key(),
		}

		// append initial values
		t.Prices = append(t.Prices, item.SpotPrice)
		t.PriceChange = append(t.PriceChange, 0.0)
		t.AbsPriceChange = append(t.PriceChange, 0.0)
		t.Times = append(t.Times, item.Timestamp)
		t.TimeChanges = append(t.TimeChanges, (time.Second * 0).Minutes())
		return nil
	}

	if t.latest.TimeStamp.After(item.Timestamp) {
		return OutOfSequenceError
	}

	// check for spot price difference first
	// - it is faster to compare!
	if t.latest.SpotPrice == item.SpotPrice || !t.latest.TimeStamp.Before(item.Timestamp) {
		return nil
	}

	// update global variables
	t.To = item.Timestamp
	t.Duration = t.To.Sub(t.From)
	// update global min/max
	if t.Max < item.SpotPrice {
		t.Max = item.SpotPrice
	}
	if t.Min > item.SpotPrice {
		t.Min = item.SpotPrice
	}

	// insert into series
	t.Prices = append(t.Prices, item.SpotPrice)
	t.AbsPriceChange = append(t.PriceChange, math.Abs(t.latest.SpotPrice-item.SpotPrice))
	t.PriceChange = append(t.PriceChange, t.latest.SpotPrice-item.SpotPrice)
	t.Times = append(t.Times, item.Timestamp)
	t.TimeChanges = append(t.TimeChanges, item.Timestamp.Sub(t.latest.TimeStamp).Minutes())

	t.latest = &PricePoint{
		SpotPrice: item.SpotPrice,
		TimeStamp: item.Timestamp,
	}

	return nil
}

func (t *Trace) PriceStats() (mean, variance, sd, skew float64) {

	return Stats(stat.Float64Slice(t.Prices))
}

func (t *Trace) PriceChangeStats() (mean, variance, sd, skew float64) {

	return Stats(stat.Float64Slice(t.PriceChange))
}

func (t *Trace) TimeChangeStats() (mean, variance, sd, skew float64) {

	return Stats(stat.Float64Slice(t.TimeChanges))
}

func Stats(data stat.Float64Slice) (mean, variance, sd, skew float64) {
	mean = stat.Mean(data)
	variance = stat.VarianceMean(data, mean)
	sd = stat.SdMean(data, mean)
	skew = stat.SkewMeanSd(data, mean, sd)
	return
}

func LoadTrace(path string) (*Trace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	t := &Trace{}
	dec := json.NewDecoder(f)
	if err := dec.Decode(t); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Trace) AliveStats(bid float64) (mean, variance, sd, skew float64) {

	if len(t.Prices) < 2 {
		panic("Not enough values to process!")
	}
	alives := []float64{}

	// cPrice := 0.0
	cTime := time.Time{}
	for i := 0; i < len(t.Prices); {
		// search until we are alive
		// fmt.Println("Scanning for next alive..")
		for i < len(t.Prices) {
			if t.Prices[i] <= bid {
				// cPrice = t.Prices[i]
				cTime = t.Times[i]
				break
			}
			i++
		}
		i++
		// scan until we die or run out of events
		for i < len(t.Prices) {
			if t.Prices[i] > bid {
				duration := t.Times[i].Sub(cTime).Minutes()
				alives = append(alives, duration)
				break
			}
			i++
		}
		i++
	}
	mean, variance, sd, skew = Stats(stat.Float64Slice(alives))

	return
}

func CostToString(price float64) string {
	return fmt.Sprintf("%.4f", price)
}

// 0.0001 = 1.0
func CostToInt(cost float64) int {
	return int(cost * 1e4)
}

// 1 = 0.0001
func IntToFloat64(units int) float64 {
	return float64(units) / 1e4
}

// func (t *Trace) MinMaxMedianCost() (float64, float64, float64) {

//  min := 1.0
//  max := 0.0
//  sum := 0.0
//  cnt := 0

//  for _, v := range t.PricePoints {
//      if v.SpotPrice > max {
//          max = v.SpotPrice
//      } else if v.SpotPrice < min {
//          min = v.SpotPrice
//      }

//      sum += v.SpotPrice
//      cnt += 1
//  }

//  if cnt == 0 {
//      return 0.0, 0.0, 0.0
//  }

//  return min, max, sum / float64(cnt)
// }

// func (t *Trace) AvgPriceSpan() float64 {
//  // unit := time.Minute

//  if len(t.PricePoints) == 0 {
//      return 0.0
//  }

//  i := 0
//  l := t.PricePoints[i]

//  changes := []float64{}
//  cnt := 0
//  sum := 0.0
//  for i = 1; i < len(t.PricePoints); i++ {
//      pp := t.PricePoints[i]

//      if pp.SpotPrice == l.SpotPrice {
//          continue
//      }

//      duration := pp.TimeStamp.Sub(l.TimeStamp).Minutes()
//      // units := math.Ceil(float64(duration) / float64(unit))

//      sum += duration
//      cnt += 1

//      changes = append(changes, duration)

//      l = pp
//  }

//  avg := sum / float64(cnt)

//  fmt.Printf(`Changes: %d
// Sum: %f
// Avg: %f
// `, cnt, sum, avg)

//  return avg
// }
