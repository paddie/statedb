package statedb

import (
	// "errors"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type TimeLine struct {
	Start  time.Time
	End    time.Duration
	events []*CheckpointTrace
	// Sync traces
	SyncStarts, SyncDurations []time.Duration
	EncStart, EncDurations    []time.Duration
	MdlStart, MdlDurations    []time.Duration
	SyncCnt                   int
	// Commit Traces
	CommitStarts, CommitDurations []time.Duration
	CmtCnt                        int
	// Price Traces
	PriceChanges []float64
	PriceTimes   []time.Duration
	PriceCnt     int
	// inserts
	InsertDurations  []time.Duration
	Inserts, Removes int
	RemoveDurations  []time.Duration
	sync.Mutex
}

// We use time relative to the Start
func (tl *TimeLine) time() time.Duration {
	return time.Now().Sub(tl.Start)
}

// *********************************
// PriceChanges - not synced
// *********************************
func (tl *TimeLine) PriceChange(c float64) {
	tl.PriceChanges = append(tl.PriceChanges, c)
	tl.PriceTimes = append(tl.PriceTimes, tl.time())
}

func (tl *TimeLine) Commit(start time.Time) {

	dur := time.Now().Sub(start)

	tl.CommitStarts = append(tl.CommitStarts, start.Sub(tl.Start))
	tl.CommitDurations = append(tl.CommitDurations, dur)
	tl.CmtCnt++
}

func (tl *TimeLine) Len() int {
	return len(tl.SyncStarts)
}

func (tl *TimeLine) XY(i int) (x, y float64) {
	return tl.SyncStarts[i].Seconds(), tl.SyncDurations[i].Seconds()
}

func (tl *TimeLine) Write(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

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
	c := NewCheckpointTrace(tl, tl.Start)
	// increase the cnt for the next tick..
	tl.SyncCnt++

	tl.Lock()
	defer tl.Unlock()
	tl.events = append(tl.events, c)
	tl.SyncStarts = append(tl.SyncStarts, now.Sub(tl.Start))
	// placeholders
	tl.SyncDurations = append(tl.SyncDurations, time.Duration(0))
	tl.MdlDurations = append(tl.MdlDurations, time.Duration(0))
	tl.EncDurations = append(tl.EncDurations, time.Duration(0))
	return c
}

func (tl *TimeLine) Tock(c *CheckpointTrace) {
	// tl.Lock()
	// defer tl.Unlock()
	// tl.Starts[c.id] = c.Start
	tl.SyncDurations[c.id] = c.Duration
	tl.MdlDurations[c.id] = c.MdlDuration
	tl.EncDurations[c.id] = c.EncDuration
}

type CheckpointTrace struct {
	tl                     *TimeLine
	id                     int
	Aborted                bool
	Start                  time.Time
	Duration               time.Duration
	SncStart, SyncDuration time.Duration
	CptStart, CptDuration  time.Duration
	EncStart, EncDuration  time.Duration
	MdlStart, MdlDuration  time.Duration
}

func (t *CheckpointTrace) time() time.Duration {
	return time.Now().Sub(t.Start)
}

func NewCheckpointTrace(tl *TimeLine, start time.Time) *CheckpointTrace {
	tc := &CheckpointTrace{
		tl:    tl,
		id:    tl.SyncCnt,
		Start: start,
	}
	return tc
}

// func (t *CheckpointTrace) checkDone() {
// 	t.Lock()
// 	defer t.Unlock()

// 	t.rem += 1
// 	if t.rem == 2 {
// 		t.Duration = t.time() - t.Start
// 		t.tl.Tock(t)
// 	}
// }

func (t *CheckpointTrace) Abort() {
	// now := time.Now()
	t.Aborted = true
	// t.End = now
	// t.Total = now.Sub(t.Start)
	t.tl.Tock(t)
}

func (t *CheckpointTrace) SyncStart() {
	t.SncStart = t.time()
}

func (t *CheckpointTrace) SyncEnd() {
	t.SyncDuration = t.time() - t.SncStart
	// t.checkDone()
}

func (t *CheckpointTrace) EncodingEnd() {
	t.EncDuration = t.time() - t.EncStart
}

func (t *CheckpointTrace) EncodingStart() {
	t.EncStart = t.time()
}

func (t *CheckpointTrace) ModelStart() {
	t.MdlStart = t.time()
}

func (t *CheckpointTrace) ModelEnd() {
	t.MdlDuration = t.time() - t.MdlStart
}

func (t *CheckpointTrace) CommitStart() {
	t.CptStart = t.time()
}

func (t *CheckpointTrace) CommitEnd() {
	t.CptDuration = t.time() - t.CptStart
	// defer t.checkDone()
}
