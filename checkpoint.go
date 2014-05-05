package statedb

import (
	// "encoding/gob"
	"errors"
	"fmt"
	// "log"
	// "os"
)

var (
	NoData = errors.New("No Data to checkpoint")
)

func (db *StateDB) checkpoint() error {
	// only returns encoding errors
	// - commit errors are reported on db.err_can
	// errChan := make(chan error)
	var err error
	if len(db.delta) == 0 {
		err = db.deltaCheckpoint()
	} else {
		err = db.zeroCheckpoint()
	}

	if err != nil {
		fmt.Println("Failed checkpointing: ", err)
		return err
	}

	return nil
}

// Encodes the two databases delta and mutable
// and passes the encoded data on to be committed
// - returns immediately after encoding, and handles any commit errors
//   in StateLoop
func (db *StateDB) deltaCheckpoint() error {
	if len(db.delta) == 0 && len(db.mutable) == 0 {
		return NoData
	}

	r := &CommitReq{
		cpt_type: DELTACPT,
		ctx:      db.ctx.newDeltaContext(),
	}

	// resp := &CommitResp{
	// 	cpt_type: DELTACPT,
	// }

	// we fire of the delta encoding in another
	// thread. Delta is most likely short.
	// errChan := make(chan error)
	var err error
	if len(db.delta) > 0 {
		// when writing a delta checkpoint
		// increase the delta id
		r.ctx.DCNT += 1
		// go func(r *CommitReq, delta DeltaTypeMap) {
		r.del, err = encodeDelta(db.delta, r.ctx.DCNT)
		if err != nil {
			return err
		}
		// r.del = data // will be <nil> if encoding failed
		// errChan <- err
		// }(r, db.delta)
	}

	// encode mutable but don't react on error before receiving
	// from errChan
	// var mut_err, del_err error
	r.mut, err = encodeMutable(db.mutable, db.ctx.MCNT)
	if err != nil {
		return err
	}
	// if encoding delta, wait for completion
	// if len(db.delta) > 0 {
	// 	resp.del_err = <-errChan
	// }

	// if encoding was successful, pass on to be
	// committed to fs
	db.comReqChan <- r
	return nil
}

// The FullCheckpoint serves as a forced checkpoint of all the known states
// - Assumes that the system is in a consistent state
// - Checkpoints the Immutable and Mutable states, and empties the delta log.
func (db *StateDB) zeroCheckpoint() error {
	// nothing to commit, but not an error
	if len(db.immutable) == 0 {
		return NoData
	}

	r := &CommitReq{
		cpt_type: ZEROCPT,
		ctx:      db.ctx.newZeroContext(),
	}

	// errChan := make(chan error)
	// go func(r *CommitReq, imm ImmKeyTypeMap) {

	// 	errChan <- err
	// }(r, db.immutable)
	// resp := &CommitResp{
	// 	cpt_type: DELTACPT,
	// }
	var err error
	r.imm, err = encodeImmutable(db.immutable)
	if err != nil {
		return err
	}
	// r.imm = data // will be <nil> if encoding failed

	// encode mutable but don't react on error before receiving
	// from errChan
	r.mut, err = encodeMutable(db.mutable, db.ctx.MCNT)
	if err != nil {
		return err
	}
	// if encoding delta, wait for completion

	// if encoding was successful, pass on to be
	// committed to fs

	db.comReqChan <- r
	return nil
}
