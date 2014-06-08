package statedb

import (
	"fmt"
	// "github.com/paddie/goamz/ec2"
	// "github.com/paddie/statedb/monitor"
	// "io"
	"time"
)

type msg struct {
	time     time.Time
	cptType  int
	err      chan error
	forceCPT bool
	t        *CheckpointTrace
	waitChan chan error
}

type CheckpointQuery struct {
	s       *Stat
	cptChan chan bool
}

func stateLoop(db *StateDB, mnx *ModelNexus, cnx *CommitNexus, path string) {
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
	// quit := false

	waitChans := []chan error{}

	for {
		select {
		case m := <-db.sync_chan:
			// if an active commit is running
			// ignore this sync
			stat.markConsistent()

			// if there is an ongoing commit
			// return immediately
			if active_commit {
				m.err <- ActiveCommitError
				if m.waitChan != nil {
					waitChans = append(waitChans, m.waitChan)
				}
				continue
			}

			// is only checked once, to make sure that
			// the mutable states have all been updated
			// with new pointers.
			if !ready {
				ready = db.readyCheckpoint()
				if !ready {
					m.err <- NotRestoredError
					continue
				}
			}
			// stat trace for the timeline
			t := m.t

			// forceCPT is used for testing
			// so we only query the model if
			// that particular flag is not set
			t.ModelStart()
			if !m.forceCPT {
				mnx.cptQueryChan <- cptQ
				if cpt := <-cptQ.cptChan; !cpt {
					m.err <- nil
					t.ModelEnd()
					t.Abort()
					continue
				}
			}
			t.ModelEnd()

			// TODO: possibly decide what type of checkpoint to encode

			// encode checkpoint
			t.EncodingStart()
			req, err := db.encodeCheckpoint(m.cptType, stat)
			if err != nil {
				// do not report error if there is nothing
				// to checkpoint
				if err == NoDataError {
					m.err <- nil
				} else {
					m.err <- err
					// report global error
					// - if we cannot encode, something is very wrong
					errChan <- err
				}
				t.Abort()
				continue
			}
			t.EncodingEnd()

			// state was successfully encoded, return control to application while performing commit
			active_commit = true

			if m.waitChan != nil {
				fmt.Println("received a commit checkpoint")
				waitChans = append(waitChans, m.waitChan)
			}

			cnx.comReqChan <- req
			m.err <- nil

			// forward the encoded state to be committed
			// and signal an active commit

			// update sync frequencies with model
			mnx.statChan <- *stat
			// 1) the delta has been encoded, so we reset it
			db.delta = nil
			// 2) update the context to reflect the
			//    type of checkpoint that was encoded
			*db.ctx = *req.ctx
		case r := <-cnx.comRespChan:
			// no active commits anymore
			active_commit = false
			// if the commit failed,
			// report the event on the errChan
			if !r.Success() {
				if len(waitChans) > 0 {
					for _, wc := range waitChans {
						wc <- r.Err()
					}
					// nil channel afterwards to make
					// sure that we don't mistakenly block
					// next time
					waitChans = nil
				}
				errChan <- r.Err()
				continue
			}
			// signal to any waiting process that
			// the write was completed.
			if len(waitChans) > 0 {
				for _, wc := range waitChans {
					wc <- nil
				}
				// nil channel afterwards to make
				// sure that we don't mistakenly block
				// next time
				waitChans = nil
			}

			// update the stats based on the type of checkpoint
			if r.cpt_type == DELTACPT {
				stat.deltaCPT(r.mut_dur, r.del_dur)
			} else {
				stat.zeroCPT(r.imm_dur, r.mut_dur)
			}
			// send copy of updated stat to model
			mnx.statChan <- *stat
		case so := <-db.op_chan:
			if !ready {
				so.err <- NotRestoredError
			}
			// Insert or Remove entries in the database
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
				mnx.statChan <- *stat
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
				mnx.statChan <- *stat
			} else {
				// if the action is unknown
				// it is a fatal error
				so.err <- UnknownOperation
				errChan <- UnknownOperation
			}
		case respChan := <-db.quit:
			// if an active commit is ongoing
			// TODO: set notify channel
			//       which will be checked on a completed commit
			mnx.Quit()
			cnx.Quit()
			if path != "" {
				err := timeline.Write(path)
				if err != nil {
					respChan <- err
					errChan <- err
					return
				}
			}
			respChan <- nil
			return
		}
	}
}
