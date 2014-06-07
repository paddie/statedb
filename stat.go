package statedb

import (
	"github.com/paddie/statedb/avgbuffer"
	// m "github.com/paddie/statedb/monitor"
	"time"
)

type StateDBInfo interface {
	// Time since last event
	TimeSinceInit() time.Duration
	TimeSinceLastCpt() time.Duration
	TimeSinceLastConsistentPoint() time.Duration
	// Averaged values
	AvgCheckpointInterval() time.Duration
	AvgConsistentPointInterval() time.Duration
	// Expected restore times
	ExpTimeToRestore() time.Duration
	ExpTimeToCheckpoint() time.Duration
	// Expected commit times
	TimeToCheckpointImmutable() time.Duration
	ExpTimeToCheckopintMutable() time.Duration
	ExpTimeToCheckpointDelta() time.Duration
	// Size of databases
	SizeDelta() int
	SizeImmutable() int
	SizeMutable() int
}

type stat struct {
	// cpt bool
	// read/write stats
	t_i, t_m       time.Duration // time to cpt imm and mut
	t_d            time.Duration // time to cpt delta
	t_r            time.Duration // time to restore from previous cpt
	i, m, d_i, d_m int64         // size of each database
	phi_i, phi_m   time.Duration // avg. cpt time pr. database
	// wall-clock datetime for events
	lastConsistent time.Time
	avgConsistent  *avgbuffer.AVGDuration

	lastCheckpoint time.Time
	avgCheckpoint  *avgbuffer.AVGDuration
	// cpt_type       int
	start time.Time
}

func NewStat(buffSize int) *Stat {
	if buffSize < 1 {
		buffSize = 1
	}

	now := time.Now()

	return &Stat{
		avgConsistent:  avgbuffer.NewAVGDuration(buffSize),
		lastConsistent: now,
		lastCheckpoint: now,
	}
}

func (s *Stat) Delta() int64 {
	return s.d_i
}

func (s *Stat) DurationAlive() time.Duration {
	return time.Now().Sub(s.start)
}

func (s *Stat) LastCheckpoint() time.Duration {
	return time.Now().Sub(s.lastCheckpoint)
}

func (s *Stat) AvgCheckpoint() time.Duration {
	return s.avgCheckpoint.TAVG()
}

func (s *Stat) LastConsistent() time.Duration {
	return time.Now().Sub(s.lastConsistent)
}

func (s *Stat) AvgConsistent() time.Duration {
	return s.avgConsistent.TAVG()
}

func (s *Stat) markConsistent() {
	now := time.Now()
	if s.lastConsistent.IsZero() {
		s.lastConsistent = now
		return
	}
	s.avgConsistent.Upsert(now.Sub(s.lastConsistent))
	s.lastConsistent = now
}

// i,m \in {0,1}
func (s *Stat) insert(i, m int64) {
	s.d_i += i
	s.d_m += m
}

// i,m \in {0,1}
func (s *Stat) remove(i, m int64) {
	s.d_i -= i
	s.d_m -= m
}

// Update stats with information after a zero-checkpoint
func (s *Stat) zeroCPT(i_dur, m_dur time.Duration) {

	T_i := i_dur
	T_m := m_dur

	s.lastCheckpoint = time.Now()
	// update write times
	s.t_i = T_i
	s.t_m = T_m
	s.t_d = 0

	// restore calculation
	s.t_r = T_i + T_m

	// update entry count
	s.i += s.d_i
	s.m += s.d_m
	s.d_i = 0
	s.d_m = 0

	// update approximated values
	s.phi_i = time.Duration(float64(s.t_i) / float64(s.i))
	s.phi_m = time.Duration(float64(s.t_m) / float64(s.m))
}

// Update stats with information after a delta-checkpoint
func (s *Stat) deltaCPT(m_dur, d_dur time.Duration) {

	T_m := m_dur
	T_d := d_dur
	s.lastCheckpoint = time.Now()

	// restore calculation
	s.t_r -= s.t_m     // subtract the previous mut_dur
	s.t_r += T_m + T_d // and add the new

	// update write times
	s.t_m = T_m
	s.t_d += T_d

	// update entry count
	s.m += s.d_m
	s.i += s.d_i
	s.d_i = 0
	s.d_m = 0

	// update approximated values
	s.phi_m = time.Duration(float64(s.t_m) / float64(s.m))
}

// Time to restore from the precious checkpoint
func (s *Stat) RestoreTime() time.Duration {
	return s.t_r
}

// Expected write-time of a checkpoint
func (s *Stat) ExpWriteCheckpoint() time.Duration {
	if s.d_i > 0 {
		return s.expWriteZero()
	}
	return s.expWriteDelta()
}

// Expected writing time of a delta-checkpoint
// using the average write time
// pr. entry in each database
// Φ(W_∆) = T_M +∆_I ·φ_I +∆_M ·φ_M
func (s *Stat) expWriteDelta() time.Duration {

	if s.t_i == 0 || s.t_m == 0 {
		return -1
	}

	return s.t_m + time.Duration(s.d_i)*s.phi_i + time.Duration(s.d_m)*s.phi_m
}

// Expected writing time of a zero-checkpoint
// using the average write time pr. entry in each database
// Φ(W_Z) = T_I +T_M +∆_I ·φ_I +∆_M ·φ_M
func (s *Stat) expWriteZero() time.Duration {
	if s.t_i == 0 || s.t_m == 0 {
		return -1
	}

	return s.t_i + s.t_m + time.Duration(s.d_i)*s.phi_i + time.Duration(s.d_m)*s.phi_m
}

func (s *Stat) ExpReadCheckpoint() time.Duration {
	if s.d_i > 0 {
		return s.expReadZero()
	}
	return s.expReadDelta()
}

// Expected restore-time of a zero-checkpoint
// using the average write\approxread time
// pr. entry in each database
// Φ(R_Z) = TI + TM + ∆I * φI + ∆M * φM
func (s *Stat) expReadZero() time.Duration {

	if s.t_i == 0 || s.t_m == 0 {
		return -1
	}

	return s.t_i + s.t_m + time.Duration(s.d_i)*s.phi_i + time.Duration(s.d_m)*s.phi_m
}

// Expected restore-time of a delta-checkpoint
// using the average write\approxread time
// pr. entry in each database
// Φ(R_∆) = TI +TM +T∆ +∆I ·φI +∆M ·φM
func (s *Stat) expReadDelta() time.Duration {

	if s.t_i == 0 || s.t_m == 0 {
		return -1
	}

	return s.t_i + s.t_m + s.t_d + time.Duration(s.d_i)*s.phi_i + time.Duration(s.d_m)*s.phi_m
}
