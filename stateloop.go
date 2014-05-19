package statedb

import (
	"fmt"
	// "github.com/paddie/goamz/ec2"
	// "github.com/paddie/statedb/monitor"
	"time"
)

type Msg struct {
	time     time.Time
	err      chan error
	forceCPT bool
	t        *CheckpointTrace
}

type CheckpointQuery struct {
	s       *Stat
	cptChan chan bool
	// t       *CheckpointTrace
}

func stateLoop(db *StateDB, mnx *ModelNexus, cnx *CommitNexus) {
	// Global error handing channel
	// - every error on this channel results in a panic
	errChan := make(chan error)

	stat := NewStat(3)

	cptQ := &CheckpointQuery{
		cptChan: make(chan bool),
		s:       stat,
	}

	// this variable is true during a commit
	active_commit := false

	// if the database was restored, one first needs to
	// restore all the mutable entries before we can
	// start encoding the new states.
	// 1) call db.Init() to run the check to see if all states
	//    have been restored prior to a run.
	ready := !db.restored

	for {
		select {
		case err := <-db.init_chan:
			// check if all mutable objects have been restored
			// - only needs to be checked once, but is check subsequent times
			if ready {
				err <- nil
				continue
			}
			// run through the mutablr db and make sure
			// all states have been updated with new pointers
			if !db.readyCheckpoint() {
				err <- NotRestoredError
				// t.Abort()
				continue
			}
			// now ready to accept sync requests
			ready = true
			// report no error, signalling OK to go!
			err <- nil
		case msg := <-db.sync_chan:
			// if an active commit is running
			// ignore this sync
			stat.markSyncPoint()
			// if there is an ongoing commit
			// return immediately
			if active_commit {
				msg.err <- nil
				continue
			}

			// if init has not been called prior to this
			// return error
			if !ready {
				msg.err <- NotRestoredError
			}

			t := msg.t
			// only ask model to checkpoint if the forceCPT flag
			// is not set
			t.ModelStart()
			if !msg.forceCPT {
				t := msg.t
				mnx.cptQueryChan <- cptQ

				if cpt := <-cptQ.cptChan; cpt == false {
					msg.err <- nil
					t.ModelEnd()
					t.Abort()
					continue
				}
			}
			t.ModelEnd()

			// TODO: possibly decide what type of checkpoint to encode

			// encode checkpoint
			t.EncodingStart()
			req, err := db.encodeCheckpoint()
			if err != nil {
				// do not report error if there is nothing
				// to checkpoint
				if err == NoDataError {
					msg.err <- nil
				} else {
					msg.err <- err
					// report global error
					// - if we cannot encode, something is very wrong
					errChan <- err
				}
				t.Abort()
				continue
			}
			t.EncodingEnd()

			// state was successfully encoded, return control to application while performing commit
			msg.err <- nil

			// forward the encoded state to be committed
			// and signal an active commit
			active_commit = true
			cnx.comReqChan <- req
			// update sync frequencies with model
			mnx.statChan <- stat

			// 1) the delta has been encoded, so we reset it
			// 2) update the context to reflect the
			//    type of checkpoint that was encoded
			db.delta = nil
			*db.ctx = *req.ctx
			fmt.Println("Successfully committed checkpoint")

		case r := <-cnx.comRespChan:
			active_commit = false

			if !r.Success() {
				errChan <- r.Err()
				continue
			}

			if r.cpt_type == DELTACPT {
				stat.deltaCPT(r.mut_dur, r.del_dur)
			} else {
				stat.zeroCPT(r.imm_dur, r.mut_dur)
			}
			// send copy of updated stat to model
			mnx.statChan <- stat
		case so := <-db.op_chan:
			kt := so.kt
			if so.action == REMOVE {
				err := db.remove(kt)
				if err != nil {
					so.err <- err
					continue
				}
				// reply success
				so.err <- nil
				// Update and send stat
				stat.remove(1, 1)
				mnx.statChan <- stat
			} else if so.action == INSERT {
				err := db.insert(kt, so.imm, so.mut)
				if err != nil {
					so.err <- err
					continue
				}
				// reply success
				so.err <- nil

				// Update and ship stat
				stat.insert(1, 1)
				mnx.statChan <- stat
			} else {
				so.err <- UnknownOperation
				errChan <- UnknownOperation
			}
		case respChan := <-db.quit:
			// if an active commit is ongoing
			// TODO: set notify channel
			//       which will be checked on a completed commit
			if active_commit {
				respChan <- ActiveCommitError
				continue
			}

			fmt.Println("Committing final checkpoint..")
			req, err := db.encodeZeroCheckpoint()
			if err != nil {
				if err == NoDataError {
					err = nil
				}
				errChan <- err
				respChan <- err
				return
			}
			cnx.comReqChan <- req
			resp := <-cnx.comRespChan
			if !resp.Success() {
				err := resp.Err()
				respChan <- err
				errChan <- err
				return
			}

			fmt.Println("Checkpoint committed. Shutting down..")

			mnx.Quit()
			cnx.Quit()
			err = timeline.Write("stat")
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println("Done!")

			respChan <- nil
			return
		}
	}
}
