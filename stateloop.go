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
	// t        *CheckpointTrace
}

type CheckpointQuery struct {
	s       *Stat
	cptChan chan bool
}

func stateLoop(db *StateDB, mnx *ModelNexus, cnx *CommitNexus) {
	// true after a decoded state has been sent of to the
	// commit process
	// committing := false
	// true after a signal has come in on cpt_chan
	// cpt_notice := false

	// Global error handing channel
	// - every error on this channel results in a panic
	errChan := make(chan error)

	// Launch the model education in another
	// thread to not block the main-thread while
	// Training on the price-trace data
	// go Educate(m, s, nx)

	// Launch the commit thread in a different thread
	// and

	stat := NewStat(3)

	cptQ := &CheckpointQuery{
		cptChan: make(chan bool),
		s:       stat,
	}

	for {
		select {
		case msg := <-db.sync_chan:
			// checkpoint trace item
			// t := msg.t
			// stat.markSyncPoint()
			// check if all mutable objects have been restored
			// - only needs to be checked once, but is check subsequent times
			if !db.readyCheckpoint() {
				fmt.Println(NotRestoredError.Error())
				msg.err <- NotRestoredError
				// t.Abort()
				continue
			}
			// only ask model to checkpoint if the forceCPT flag
			// is not set
			if !msg.forceCPT {
				// fmt.Println("mdlstart")
				// t.ModelStart()
				// cptQ.t_avg_sync = stat.t_avg_sync
				// cptQ.t_processed = time.Now().Sub(stat.checkpoint_time)
				mnx.cptQueryChan <- cptQ

				if cpt := <-cptQ.cptChan; cpt == false {
					msg.err <- nil
					// t.ModelEnd()
					// t.Abort()
					continue
				}
				// fmt.Println("mdlend")
				// t.ModelEnd()
			}

			// encode checkpoint
			// fmt.Println("encstart")
			// t.EncodingStart()
			req, err := db.checkpoint()
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
				// t.EncodingEnd()
				// t.Abort()
				continue
			}
			// fmt.Println("encend")
			// t.EncodingEnd()
			// successfully encoded, return control to application while finishing commit
			msg.err <- nil
			// fmt.Println("cptStart")
			// t.CheckpointStart()
			cnx.comReqChan <- req
			// wait for the response from the commits
			// - we do not accept insert/deletes during this process
			r := <-cnx.comRespChan
			// fmt.Println("cptEnd")
			// t.CheckpointEnd()
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

			// nil delta and update context
			// to reflect state of checkpoint
			db.delta = nil
			*db.ctx = *r.ctx
			fmt.Println("Successfully committed checkpoint")
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
				// fmt.Println("received insert: " + kt.String())
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
			fmt.Println("Committing final checkpoint..")
			req, err := db.zeroCheckpoint()
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
			err = db.tl.Write("stat")
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println("Done!")

			respChan <- nil
			return
		}
	}
}
