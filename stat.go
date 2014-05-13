package statedb

import (
	"github.com/paddie/statedb/avgbuffer"
	// m "github.com/paddie/statedb/monitor"
	"time"
)

type Stat struct {
	// cpt bool
	// read/write stats
	t_i, t_m       float64 // time to cpt imm and mut
	t_d            float64 // time to cpt delta
	t_r            float64 // time to restore from previous cpt
	i, m, d_i, d_m int     // size of each database
	phi_i, phi_m   float64 // avg. cpt time pr. database
	// wall-clock datetime for events
	lastSync       time.Time
	lastCheckpoint time.Time
	alive          time.Time
	syncBuffer     *avgbuffer.AVGBuffer
	cpt_type       int
}

func NewStat(buffSize int) *Stat {
	if buffSize < 1 {
		buffSize = 1
	}
	return &Stat{
		syncBuffer:     avgbuffer.NewAVGBuffer(buffSize),
		lastSync:       time.Now(),
		lastCheckpoint: time.Now(),
	}
}

func (s *Stat) Delta() int {
	return s.d_i
}

func (s *Stat) AliveMinutes() float64 {
	return time.Now().Sub(s.alive).Minutes()
}

func (s *Stat) SinceCptMinutes() float64 {
	return time.Now().Sub(s.lastCheckpoint).Minutes()
}

func (s *Stat) AvgSyncMinutes() float64 {
	return s.syncBuffer.AVG()
}

func (s *Stat) markSyncPoint() {

	now := time.Now()

	if s.lastSync.IsZero() {
		s.lastSync = now
		return
	}

	dur := now.Sub(s.lastSync)
	s.syncBuffer.Upsert(dur.Minutes())
	s.lastSync = now
}

// i,m \in {0,1}
func (s *Stat) insert(i, m int) {
	s.d_i += i
	s.d_m += m
}

// i,m \in {0,1}
func (s *Stat) remove(i, m int) {
	s.d_i -= i
	s.d_m -= m
}

// Update stats with information after a zero-checkpoint
func (s *Stat) zeroCPT(i_dur, m_dur time.Duration) {

	T_i := i_dur.Minutes()
	T_m := m_dur.Minutes()
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
	s.phi_i = s.t_i / float64(s.i)
	s.phi_m = s.t_m / float64(s.m)
}

// Update stats with information after a delta-checkpoint
func (s *Stat) deltaCPT(m_dur, d_dur time.Duration) {

	T_m := m_dur.Minutes()
	T_d := d_dur.Minutes()
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
	s.phi_m = s.t_m / float64(s.m)
}

// Time to restore from the precious checkpoint
func (s *Stat) RestoreTime() float64 {
	return s.t_r
}

// Expected write-time of a checkpoint
func (s *Stat) ExpWriteCheckpoint() float64 {
	if s.d_i > 0 {
		return s.expWriteZero()
	}
	return s.expWriteDelta()
}

// Expected writing time of a delta-checkpoint
// using the average write time
// pr. entry in each database
// Φ(W_∆) = T_M +∆_I ·φ_I +∆_M ·φ_M
func (s *Stat) expWriteDelta() float64 {

	if s.t_i == 0.0 || s.t_m == 0.0 {
		return -1.0
	}

	return s.t_m + float64(s.d_i)*s.phi_i + float64(s.d_m)*s.phi_m
}

// Expected writing time of a zero-checkpoint
// using the average write time pr. entry in each database
// Φ(W_Z) = T_I +T_M +∆_I ·φ_I +∆_M ·φ_M
func (s *Stat) expWriteZero() float64 {
	if s.t_i == 0.0 || s.t_m == 0.0 {
		return -1.0
	}

	return s.t_i + s.t_m + float64(s.d_i)*s.phi_i + float64(s.d_m)*s.phi_m
}

func (s *Stat) ExpReadCheckpoint() float64 {
	if s.d_i > 0 {
		return s.expReadZero()
	}
	return s.expReadDelta()
}

// Expected restore-time of a zero-checkpoint
// using the average write\approxread time
// pr. entry in each database
// Φ(R_Z) = TI + TM + ∆I * φI + ∆M * φM
func (s *Stat) expReadZero() float64 {

	if s.t_i == 0.0 || s.t_m == 0.0 {
		return -1.0
	}

	return s.t_i + s.t_m + float64(s.d_i)*s.phi_i + float64(s.d_m)*s.phi_m
}

// Expected restore-time of a delta-checkpoint
// using the average write\approxread time
// pr. entry in each database
// Φ(R_∆) = TI +TM +T∆ +∆I ·φI +∆M ·φM
func (s *Stat) expReadDelta() float64 {

	if s.t_i == 0.0 || s.t_m == 0.0 {
		return -1.0
	}

	return s.t_i + s.t_m + s.t_d + float64(s.d_i)*s.phi_i + float64(s.d_m)*s.phi_m
}
