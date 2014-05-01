package statedb

type Stat struct {
	T_i, T_m, T_d  float64 // actual cpt-time in minutes
	I, M, D_i, D_m int     // entries in each database
	Phi_i, Phi_m   float64 // avg. checkpoint time pr. entry in imm/mut
}

// i,m \in {0,1}
func (s *Stat) Insert(i, m int) {
	s.I += i
	s.M += m
}

// i,m \in {0,1}
func (s *Stat) Remove(i, m int) {
	s.I -= i
	s.M -= m
}

// Update stats with information after a zero-checkpoint
func (s *Stat) ZeroCPT(T_i, T_m float64) {
	// update write times
	s.T_i = T_i
	s.T_m = T_m
	s.T_d = 0

	// update entry count
	s.I += s.D_i
	s.M += s.D_m
	s.D_i = 0
	s.D_m = 0

	// update approximated values
	s.Phi_i = s.T_i / float64(s.I)
	s.Phi_m = s.T_m / float64(s.M)
}

// Update stats with information after a delta-checkpoint
func (s *Stat) DeltaCPT(T_m, T_d float64) {
	// update write times
	s.T_m = T_m
	s.T_d += T_d

	// update entry count
	s.M += s.D_m
	s.I += s.D_i
	s.D_i = 0
	s.D_m = 0

	// update approximated values
	s.Phi_m = s.T_m / float64(s.M)
}

// Expected writing time of a zero-checkpoint
// using the average write time pr. entry in each database
// Φ(W_Z) = T_I +T_M +∆_I ·φ_I +∆_M ·φ_M
func (s *Stat) ExpWriteZero() float64 {

	if s.T_i == 0.0 || s.T_m == 0.0 {
		return 0.0
	}

	return s.T_i + s.T_m + float64(s.D_i)*s.Phi_i + float64(s.D_m)*s.Phi_m
}

// Expected writing time of a delta-checkpoint
// using the average write\approx read time
// pr. entry in each database
// Φ(W_∆) = T_M +∆_I ·φ_I +∆_M ·φ_M
func (s *Stat) ExpWriteDelta() float64 {

	if s.T_i == 0.0 || s.T_m == 0.0 {
		return 0.0
	}

	return s.T_m + float64(s.D_i)*s.Phi_i + float64(s.D_m)*s.Phi_m
}

// Expected restore-time of a zero-checkpoint
// using the average write\approxread time
// pr. entry in each database
// Φ(R_Z) = TI + TM + ∆I * φI + ∆M * φM
func (s *Stat) ExpReadZero() float64 {

	if s.T_i == 0.0 || s.T_m == 0.0 {
		return 0.0
	}

	return s.T_i + s.T_m + float64(s.D_i)*s.Phi_i + float64(s.D_m)*s.Phi_m
}

// Expected restore-time of a delta-checkpoint
// using the average write\approxread time
// pr. entry in each database
// Φ(R_∆) = TI +TM +T∆ +∆I ·φI +∆M ·φM
func (s *Stat) ExpReadDelta() float64 {

	if s.T_i == 0.0 || s.T_m == 0.0 {
		return 0.0
	}

	return s.T_i + s.T_m + s.T_d + float64(s.D_i)*s.Phi_i + float64(s.D_m)*s.Phi_m
}
