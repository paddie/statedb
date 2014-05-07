package statedb

import (
	"fmt"
	"github.com/paddie/goamz/ec2"
	"github.com/paddie/statedb/monitor"
	"time"
)

type Msg struct {
	time     time.Time
	err      chan error
	forceCPT bool
}

type ModelNexus struct {
	errChan chan error
	// modelChan    chan Model
	statChan     chan Stat
	cptQueryChan chan *CheckpointQuery
	monitor      *monitor.Monitor
	quitChan     chan bool
}

func (nx *ModelNexus) Quit() {

	// send quit signal to model
	nx.quitChan <- true

	// turn of the ec2 price monitor
	nx.monitor.QuitBlock()
	// shut down all sending channels..
	close(nx.statChan)
	close(nx.cptQueryChan)
	close(nx.quitChan)
}

func NewModelNexus(quitChan chan bool, errChan chan bool) {
	return &ModelNexus{
		statChan:     make(chan Stat),
		cptQueryChan: make(chan *CheckpointQuery),
		errChan:      errChan,
		// modelChan:    make(chan Model),
		quitChan: modelQuit,
	}
}
func stateLoop(db *StateDB, fs Persistence, m Model, s *monitor.EC2Instance) {
	// true after a decoded state has been sent of to the
	// commit process
	// committing := false
	// true after a signal has come in on cpt_chan
	// cpt_notice := false

	// Global error handing channel
	// - every error on this channel results in a panic
	errChan := make(chan error)

	// Model specific channels
	modelQuit := make(chan bool)
	statChan := make(chan Stat, 1)

	// ModelNexus holds all the channels used to
	// communicate with the model
	nx := NewModelNexus(modelQuit, errChan)

	// Launch the model education in another
	// thread to not block the main-thread while
	// Training on the price-trace data
	go Educate(m, s, nx)

	// Launch the commit thread in a different thread
	// and
	go commitLoop(fs, db.comReqChan, db.comRespChan)

	cptQ := &CheckpointQuery{
		cptChan: make(chan bool),
	}
	stat := &Stat{}
	var model Model
	for {
		select {
		case msg := <-db.sync_chan:
			if committing {
				msg.err <- ActiveCommitError
				continue
			}
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
				nx.cptQueryChan <- cptQ

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
			r := <-db.comRespChan
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
				// fmt.Println("received delete: " + kt.String())
				so.err <- db.remove(kt)
			} else if so.action == INSERT {
				// fmt.Println("received insert: " + kt.String())
				so.err <- db.insert(kt, so.imm, so.mut)
			} else {
				so.err <- UnknownOperation //fmt.Errorf("Unknown Action: %d", so.action)
			}
		case respChan := <-db.quit:
			fmt.Println("Committing final checkpoint..")
			err := db.zeroCheckpoint()
			if err != nil {
				respChan <- err
			} else {
				r := <-db.comRespChan
				if !r.Success() {
					respChan <- r.Err()
				}
			}
			nx.Quit()
			fmt.Println("Checkpoint committed. Shutting down..")
			respChan <- nil
			return
			// case m := <-nx.modelChan:
			// 	model = m
			// 	// close the channel so we dont
			// 	// waste resources listening on a channel
			// 	// that is never sent on again
			// 	close(nx.modelChan)
		}
	}
}
