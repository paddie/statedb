package statedb

import (
	m "github.com/paddie/statedb/monitor"
	"time"
)

type Stat struct {
	t_i, t_m, t_d     float64 // actual cpt-time in minutes
	i, m, d_i, d_m    int     // entries in each database
	phi_i, phi_m      float64 // avg. checkpoint time pr. entry in imm/mut
	checkpoint_time   time.Time
	mostRecentCPTType int
	t_avg_sync        time.Duration
	latestPricePoint  *m.PricePoint // Latest PricePoint
}

// i,m \in {0,1}
func (s *Stat) Insert(i, m int) {
	s.d_i += i
	s.d_m += m
}

// i,m \in {0,1}
func (s *Stat) Remove(i, m int) {
	s.d_i -= i
	s.d_m -= m
}

// Update stats with information after a zero-checkpoint
func (s *Stat) ZeroCPT(T_i, T_m float64) {
	// update write times
	s.t_i = T_i
	s.t_m = T_m
	s.t_d = 0

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
func (s *Stat) DeltaCPT(T_m, T_d float64) {
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

// Expected writing time of a zero-checkpoint
// using the average write time pr. entry in each database
// Φ(W_Z) = T_I +T_M +∆_I ·φ_I +∆_M ·φ_M
func (s *Stat) ExpWriteZero() float64 {

	if s.t_i == 0.0 || s.t_m == 0.0 {
		return 0.0
	}

	return s.t_i + s.t_m + float64(s.d_i)*s.phi_i + float64(s.d_m)*s.phi_m
}

// Expected writing time of a delta-checkpoint
// using the average write\approx read time
// pr. entry in each database
// Φ(W_∆) = T_M +∆_I ·φ_I +∆_M ·φ_M
func (s *Stat) ExpWriteDelta() float64 {

	if s.t_i == 0.0 || s.t_m == 0.0 {
		return 0.0
	}

	return s.t_m + float64(s.d_i)*s.phi_i + float64(s.d_m)*s.phi_m
}

// Expected restore-time of a zero-checkpoint
// using the average write\approxread time
// pr. entry in each database
// Φ(R_Z) = TI + TM + ∆I * φI + ∆M * φM
func (s *Stat) ExpReadZero() float64 {

	if s.t_i == 0.0 || s.t_m == 0.0 {
		return 0.0
	}

	return s.t_i + s.t_m + float64(s.d_i)*s.phi_i + float64(s.d_m)*s.phi_m
}

// Expected restore-time of a delta-checkpoint
// using the average write\approxread time
// pr. entry in each database
// Φ(R_∆) = TI +TM +T∆ +∆I ·φI +∆M ·φM
func (s *Stat) ExpReadDelta() float64 {

	if s.t_i == 0.0 || s.t_m == 0.0 {
		return 0.0
	}

	return s.t_i + s.t_m + s.t_d + float64(s.d_i)*s.phi_i + float64(s.d_m)*s.phi_m
}
