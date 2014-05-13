package statedb

import (
	// "errors"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type TimeLine struct {
	Start, End   time.Time
	events       []*CheckpointTrace
	Starts, Ends []time.Duration
	Durations    []time.Duration
	Cnt          int
	sync.Mutex
}

func (tl *TimeLine) Len() int {
	return len(tl.Starts)
}

func (tl *TimeLine) XY(i int) (x, y float64) {
	return float64(tl.Starts[i].Seconds()), float64(tl.Durations[i].Nanoseconds()) / 10e6
}

func (tl *TimeLine) Write(path string) error {
	f, err := os.Create("stat.do")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)

	if err := enc.Encode(tl); err != nil {
		return err
	}

	return nil
}

func NewTimeLine() *TimeLine {
	return &TimeLine{
		Start: time.Now(),
	}
}

// Signal that a checkpoint has been signaled
func (tl *TimeLine) Tick() *CheckpointTrace {

	if tl == nil {
		return nil
	}

	now := time.Now()
	c := NewCheckpointTrace(tl, now)
	// increase the cnt for the next tick..
	tl.Cnt++

	tl.Lock()
	defer tl.Unlock()
	tl.events = append(tl.events, c)
	tl.Starts = append(tl.Starts, now.Sub(tl.Start))
	tl.Ends = append(tl.Ends, time.Duration(0))
	tl.Durations = append(tl.Durations, time.Duration(0))
	return c
}

func (tl *TimeLine) Tock(c *CheckpointTrace) {
	tl.Lock()
	defer tl.Unlock()
	tl.End = c.End
	tl.Ends[c.id] = c.End.Sub(tl.Start)
	tl.Durations[c.id] = tl.Ends[c.id] - tl.Starts[c.id]
}

type CheckpointTrace struct {
	tl               *TimeLine
	id               int
	Aborted          bool
	Start, End       time.Time
	Total            time.Duration
	SncStart, SncEnd time.Time
	SyncDuration     time.Duration
	// sync.Mutex
}

func NewCheckpointTrace(tl *TimeLine, now time.Time) *CheckpointTrace {
	tc := &CheckpointTrace{
		tl: tl,
		id: tl.Cnt,
	}
	return tc
}

// func (t *CheckpointTrace) checkDone(now time.Time) {
// 	t.Lock()
// 	defer t.Unlock()

// 	t.rem += 1
// 	if t.rem == 4 {
// 		t.End = now
// 		t.Total = now.Sub(t.Start)
// 		t.tl.Tock(t)
// 	}
// }

// func (t *CheckpointTrace) Abort() {
// 	now := time.Now()
// 	t.Aborted = true
// 	t.End = now
// 	t.Total = now.Sub(t.Start)
// 	t.tl.Tock(t)
// }

func (t *CheckpointTrace) SyncStart() {
	now := time.Now()
	t.SncStart = now
	t.Start = now
}

func (t *CheckpointTrace) SyncEnd() {
	now := time.Now()
	// defer t.checkDone(now)
	t.SncEnd = now
	t.SyncDuration = now.Sub(t.SncStart)
	t.End = now
	t.Total = now.Sub(t.Start)
	t.tl.Tock(t)
}

// func (t *CheckpointTrace) EncodingStart() {
// 	t.EncStart = time.Now()
// }

// func (t *CheckpointTrace) EncodingEnd() {
// 	now := time.Now()
// 	defer t.checkDone(now)
// 	t.EncEnd = now
// 	t.EncDuration = now.Sub(t.EncStart)
// }

// func (t *CheckpointTrace) ModelStart() {
// 	t.MdlStart = time.Now()
// }

// func (t *CheckpointTrace) ModelEnd() {
// 	now := time.Now()
// 	defer t.checkDone(now)
// 	t.MdlEnd = now
// 	t.MdlDuration = now.Sub(t.MdlStart)
// }

// func (t *CheckpointTrace) CheckpointStart() {
// 	t.CptStart = time.Now()
// }

// func (t *CheckpointTrace) CheckpointEnd() {
// 	now := time.Now()
// 	defer t.checkDone(now)
// 	t.CptEnd = now
// 	t.CptDuration = now.Sub(t.CptStart)
// }
