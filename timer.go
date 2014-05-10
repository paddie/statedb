package statedb

import (
	// "errors"
	"sync"
	"time"
)

type TimeLine struct {
	Start, End   time.Time
	Events       []*CheckpointTrace
	Starts, Ends []time.Duration
	Cnt          int
	sync.Mutex
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

	c := NewCheckpointTrace(tl)
	// increase the cnt for the next tick..
	tl.Cnt++

	tl.Lock()
	tl.Events = append(tl.Events, c)
	tl.Starts = append(tl.Starts, c.Start.Sub(tl.Start))
	tl.Ends = append(tl.Ends, time.Duration(0))
	tl.Lock()
	return c
}

func (tl *TimeLine) Tock(c *CheckpointTrace) {
	tl.Lock()
	tl.Ends[c.id] = c.End.Sub(tl.Start)
	tl.End = c.End
	tl.Lock()
}

type CheckpointTrace struct {
	tl               *TimeLine
	id               int
	Aborted          bool
	Start, End       time.Time
	Total            time.Duration
	SncStart, SncEnd time.Time
	SyncDuration     time.Duration
	EncStart, EncEnd time.Time
	EncDuration      time.Duration
	CptStart, CptEnd time.Time
	CptDuration      time.Duration
	MdlStart, MdlEnd time.Time
	MdlDuration      time.Duration
	rem              int
	sync.Mutex
}

func NewCheckpointTrace(tl *TimeLine) *CheckpointTrace {

	tc := &CheckpointTrace{
		tl: tl,
		id: tl.Cnt,
	}

	return tc
}

func (t *CheckpointTrace) CheckDone(now time.Time) {
	t.Lock()
	defer t.Unlock()

	t.rem += 1
	if t.rem == 4 {
		t.End = now
		t.Total = now.Sub(t.Start)
		t.tl.Tock(t)
	}
}

func (t *CheckpointTrace) Abort() {
	now := time.Now()
	t.Aborted = true
	t.End = now
	t.Total = now.Sub(t.Start)
	t.tl.Tock(t)
}

func (t *CheckpointTrace) SyncStart() {
	now := time.Now()
	t.SncStart = now
	t.Start = now
}

func (t *CheckpointTrace) SyncEnd() {
	now := time.Now()
	defer t.CheckDone(now)
	t.SncEnd = now
	t.SyncDuration = now.Sub(t.SncStart)
}

func (t *CheckpointTrace) EncodingStart() {
	t.EncStart = time.Now()
}

func (t *CheckpointTrace) EncodingEnd() {
	now := time.Now()
	defer t.CheckDone(now)
	t.EncEnd = now
	t.EncDuration = now.Sub(t.EncStart)
}

func (t *CheckpointTrace) ModelStart() {
	t.MdlStart = time.Now()
}

func (t *CheckpointTrace) ModelEnd() {
	now := time.Now()
	defer t.CheckDone(now)
	t.MdlEnd = now
	t.MdlDuration = now.Sub(t.MdlStart)
}

func (t *CheckpointTrace) CheckpointStart() {
	t.CptStart = time.Now()
}

func (t *CheckpointTrace) CheckpointEnd() {
	now := time.Now()
	defer t.CheckDone(now)
	t.CptEnd = now
	t.CptDuration = now.Sub(t.CptStart)
}
