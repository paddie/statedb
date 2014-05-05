package statedb

import (
	// "bytes"
	"fmt"
	// "reflect"
	"time"
	// "strconv"
	// "encoding/gob"
	"errors"
	// "log"
	"sync"
)

const (
	REMOVE  int = -1
	INSERT  int = 1
	RESTORE int = 0
)

var (
	ActiveCommitError = errors.New("An active commit has not returned")
	NotRestoredError  = errors.New("Database has not been fully restored")
	UnknownOperation  = errors.New("Unknown Operation")
)

type StateDB struct {
	// fs       Persistence
	restored bool     // has statedb just been restored
	ready    bool     // have all mutable objects been restored?
	ctx      *Context // cpt and restore information
	// State databases
	immutable ImmKeyTypeMap // immutable states
	delta     DeltaTypeMap  // static state delta
	mutable   MutKeyTypeMap // mutable state
	// Synchronization channels
	op_chan     chan *StateOperation // handles insert and remove operations
	comReqChan  chan *CommitReq
	comRespChan chan *CommitResp
	quit        chan chan error // shutdown signals
	// cpt_chan    chan time.Time  // checkpoint signals are sent on this channel
	// comm_resp    chan *CommitResp
	sync_chan    chan *Msg // consistent state signals are sent on this channel
	sync.RWMutex           // for synchronizing things that don't need the channels..
}

// Interface to checkpoint data to non-volatile memory
type Persistence interface {
	List(prefix string) ([]string, error) // list items in dir
	Put(name string, data []byte) error   // create/overwrite file
	Get(name string) ([]byte, error)      // get file
	Delete(path string) error             // delete file
	Init() error                          // ensure that directory/bucket exists
}

func (db *StateDB) readyCheckpoint() bool {

	if db.ready || !db.restored {
		return true
	}

	for _, vt := range db.mutable {
		for _, vs := range vt {
			if !vs.v.IsValid() {
				fmt.Printf("Not valid: %#v", *vs)
				return false
			}
		}
	}
	// every mutable object has been restored
	db.ready = true
	return true
}

func NewStateDB(fs Persistence) (*StateDB, error) {

	// Initialize the directories
	db, err := restore(fs)
	if err != nil {
		db = &StateDB{
			immutable: make(ImmKeyTypeMap),
			mutable:   make(MutKeyTypeMap),
			delta:     make(DeltaTypeMap),
			ctx:       NewContext(),
		}
	}

	db.sync_chan = make(chan *Msg)
	db.comReqChan = make(chan *CommitReq)
	db.comRespChan = make(chan *CommitResp)
	db.op_chan = make(chan *StateOperation)
	db.quit = make(chan chan error)
	// db.cpt_chan = make(chan time.Time)

	go stateLoop(db, fs)

	return db, nil
}

func stateLoop(db *StateDB, fs Persistence) {
	// true after a decoded state has been sent of to the
	// commit process
	committing := false
	// true after a signal has come in on cpt_chan
	cpt_notice := false
	cpt_chan := make(chan time.Time)

	go commitLoop(fs, db.comReqChan, db.comRespChan)

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
			if !msg.forceCPT && !cpt_notice {
				msg.err <- nil
				continue
			}
			// reset notice
			// stupid heuristic for when to checkpoint
			if err := db.checkpoint(); err != nil {
				msg.err <- err
				continue
			}
			msg.err <- nil
			// wait for the response from the commits
			r := <-db.comRespChan
			if !r.Success() {
				fmt.Println("Failed to checkpoint: ", r.Error())
				continue
			}

			db.delta = nil
			*db.ctx = *r.ctx
			fmt.Println("Successfully committed checkpoint")
			cpt_notice = false
		case <-cpt_chan:
			// set the checkpoint notice to force a checkpoint
			// in the next consistent state
			cpt_notice = true
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
		case err_chan := <-db.quit:
			fmt.Println("Committing final checkpoint..")
			err := db.zeroCheckpoint()
			close(db.comReqChan)
			err_chan <- err
			fmt.Println("Checkpoint committed. Shutting down..")
			return
		}
	}
}

func (db *StateDB) Types() []string {

	var ts []string
	for t, _ := range db.immutable {
		ts = append(ts, t)
	}
	return ts
}

type Msg struct {
	time     time.Time
	err      chan error
	forceCPT bool
}

type StateOperation struct {
	kt     *KeyType
	imm    []byte
	mut    *MutState
	action int
	err    chan error
}

// Sync is a call for consistency; if the monitor has signalled a checkpoint
// a checkpoint will be committed. During this time, we cannot allow any processes to
// write or delete objects in the database.
func (db *StateDB) sync() error {
	// response channel
	err := make(chan error)
	db.sync_chan <- &Msg{
		time: time.Now(),
		err:  err,
	}
	return <-err
}

func (db *StateDB) forceCheckpoint() error {
	err := make(chan error)
	db.sync_chan <- &Msg{
		time:     time.Now(),
		err:      err,
		forceCPT: true,
	}
	return <-err
}

// When called, all checkpointing is shut down,
// and a final, full checkpoint is written to disk.
func (db *StateDB) Commit() error {

	err := make(chan error)
	fmt.Println("Commenceing final commit and shutdown..")
	db.quit <- err
	return <-err
}

func (db *StateDB) insert(kt *KeyType, imm []byte, mut *MutState) error {

	if db.immutable.contains(kt) {
		return errors.New("KeyType " + kt.String() + " already exists")
	}

	db.insertImmutable(kt, imm)

	if mut != nil {
		db.insertMutable(kt, mut)
	}
	return nil
}

func (db *StateDB) remove(kt *KeyType) error {

	if !db.immutable.contains(kt) {
		return fmt.Errorf("StateDB.Remove: KeyType %s does not exist",
			kt.String())
	}

	db.immutable.remove(kt)

	if db.delta == nil {
		db.delta = make(DeltaTypeMap)
	}

	db.delta.remove(kt)
	db.mutable.remove(kt)

	return nil
}

func (db *StateDB) insertImmutable(kt *KeyType, val []byte) {

	if db.immutable == nil {
		db.immutable = make(ImmKeyTypeMap)
	}

	db.immutable.insert(kt, val)
	db.insertDelta(kt, val)
}

// Do we need a check for existense, or is that already made?
func (db *StateDB) insertDelta(kt *KeyType, val []byte) {

	if db.delta == nil {
		db.delta = make(DeltaTypeMap)
	}
	db.delta.insert(kt, val)
}

func (db *StateDB) insertMutable(kt *KeyType, mut *MutState) {
	if db.mutable == nil {
		db.mutable = make(MutKeyTypeMap)
	}
	db.mutable.insert(kt, mut)
}
