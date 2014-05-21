package statedb

import (
	// "encoding/gob"
	"errors"
	"fmt"
	// "log"
	// "os"
)

var (
	NoDataError = errors.New("No Data to checkpoint")
)

func (db *StateDB) encodeCheckpoint(cptType int, stat *Stat) (*CommitReq, error) {
	// only returns encoding errors
	// - commit errors are reported on db.err_can
	// errChan := make(chan error)

	switch cptType {
	case ZEROCPT:
		return db.encodeZeroCheckpoint()
	case DELTACPT:
		// A delta checkpoint must be preceeded by
		// an initial zero checkpoint
		// - should maybe just return zero checkpoint
		if db.ctx.RCID == 0 {
			return nil, DeltaBeforeZeroError
		}
		return db.encodeDeltaCheckpoint()
	case NONDETERMCPT:
		// base case
		if db.ctx.RCID == 0 {
			return db.encodeZeroCheckpoint()
		}
		// if nothing is in the delta
		// - we obviously commit a delta checkpoint
		if len(db.delta) == 0 {
			return db.encodeDeltaCheckpoint()
		}

		// use the stat to determine which checkpoint to choose
		if stat.expReadDelta() < stat.expReadZero() {
			return db.encodeDeltaCheckpoint()
		}

		// TODO: maybe provide this with seperate heuristic
		return db.encodeZeroCheckpoint()
	}

	return nil, fmt.Errorf("InvalidCheckpointType: %d", cptType)
}

// Encodes the two databases delta and mutable
// and passes the encoded data on to be committed
// - returns immediately after encoding, and handles any commit errors
//   in StateLoop
func (db *StateDB) encodeDeltaCheckpoint() (*CommitReq, error) {
	if len(db.delta) == 0 && len(db.mutable) == 0 {
		return nil, NoDataError
	}

	r := &CommitReq{
		cpt_type: DELTACPT,
		ctx:      db.ctx.newDeltaContext(),
	}

	var err error
	if len(db.delta) > 0 {
		// when writing a delta checkpoint
		// increase the delta id
		r.ctx.DCNT += 1
		r.del, err = encodeDelta(db.delta, r.ctx.DCNT)
		if err != nil {
			return nil, err
		}
	}
	r.mut, err = encodeMutable(db.mutable, db.ctx.MCNT)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// The FullCheckpoint serves as a forced checkpoint of all the known states
// - Assumes that the system is in a consistent state
// - Checkpoints the Immutable and Mutable states, and empties the delta log.
func (db *StateDB) encodeZeroCheckpoint() (*CommitReq, error) {
	// nothing to commit, but not an error
	if len(db.immutable) == 0 {
		return nil, NoDataError
	}

	r := &CommitReq{
		cpt_type: ZEROCPT,
		ctx:      db.ctx.newZeroContext(),
	}

	var err error
	r.imm, err = encodeImmutable(db.immutable)
	if err != nil {
		return nil, err
	}

	r.mut, err = encodeMutable(db.mutable, db.ctx.MCNT)
	if err != nil {
		return nil, err
	}

	return r, nil
}
