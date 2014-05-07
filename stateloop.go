package statedb

import (
	"fmt"
	// "github.com/paddie/goamz/ec2"
	"github.com/paddie/statedb/monitor"
	"time"
)

type Msg struct {
	time     time.Time
	err      chan error
	forceCPT bool
}

type CheckpointQuery struct {
	t_processed time.Duration
	t_avg_sync  time.Duration
	cptChan     chan bool
}

// type StateNexus struct {
// 	syncChan chan
// }

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

	defer mxn.Quit()
	defer cnx.Quit()

	cptQ := &CheckpointQuery{
		cptChan: make(chan bool),
	}
	stat := &Stat{}
	for {
		select {
		case msg := <-db.sync_chan:
			// check if all mutable objects have been restored
			// - only needs to be checked once, but is check subsequent times
			if !db.readyCheckpoint() {
				fmt.Println(NotRestoredError.Error())
				msg.err <- NotRestoredError
				continue
			}
			// if the checkpoint is not forced or
			// the monitor has not given a cpt_notice
			// the sync simply returns
			if !msg.forceCPT {
				cptQ.t_avg_sync = stat.t_avg_sync
				cptQ.t_processed = time.Now().Sub(stat.checkpoint_time)
				mnx.cptQueryChan <- cptQ

				if cpt := <-cptQ.cptChan; cpt == false {
					msg.err <- nil
					continue
				}
			}
			// reset notice
			// stupid heuristic for when to checkpoint
			if err := db.checkpoint(); err != nil {
				msg.err <- err
				errChan <- err
				continue
			}
			msg.err <- nil
			// wait for the response from the commits
			r := <-cnx.comRespChan
			if !r.Success() {
				errChan <- r.Err()
				continue
			}

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
				stat.Remove(1, 1)
				mnx.statChan <- *stat

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
				stat.Insert(1, 1)
				mnx.statChan <- *stat

			} else {
				so.err <- UnknownOperation //fmt.Errorf("Unknown Action: %d", so.action)
			}
		case respChan := <-db.quit:
			fmt.Println("Committing final checkpoint..")
			err := db.zeroCheckpoint()
			if err != nil {
				errChan <- err
				respChan <- err
				return
			} else {
				r := <-cnx.comRespChan
				if !r.Success() {
					respChan <- r.Err()
				}
				return
			}
			fmt.Println("Checkpoint committed. Shutting down..")

			respChan <- nil
			return
		}
	}
}
