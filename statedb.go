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
	// "github.com/paddie/statedb/monitor"
	"sync"
)

const (
	ZEROCPT = iota
	DELTACPT
	NONDETERMCPT
	STATS
	REMOVE
	INSERT
	RESTORE
)

var (
	ActiveCommitError     = errors.New("An active commit has not returned")
	NotRestoredError      = errors.New("Database has not been fully restored")
	UnknownOperation      = errors.New("Unknown Operation")
	timeline              *TimeLine
	DeltaBeforeZeroError  = errors.New("Delta Checkpoint requested before an initial Zero checkpoint.")
	InvalidCheckpointType = errors.New("Invalid CheckpointType")
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
	op_chan chan *stateOperation // handles insert and remove operations
	// comReqChan   chan *CommitReq
	// comRespChan  chan *CommitResp
	quit         chan chan error // shutdown signals
	sync_chan    chan *msg       // consistent state signals are sent on this channel
	init_chan    chan chan error
	sync.RWMutex // for synchronizing things that don't need the channels..
	// tl           *TimeLine
}

// func (db *StateDB) Restored() bool {
// 	return db.restored
// }

func (db *StateDB) readyCheckpoint() bool {

	if !db.restored {
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
	// db.ready = true
	return true
}

func NewStateDB(fs Persistence, model Model, monitor Monitor, bid float64, trace bool) (*StateDB, bool, error) {

	// Initialize the directories
	db, err := restore(fs)
	if db == nil || err != nil {
		db = &StateDB{
			immutable: make(ImmKeyTypeMap),
			mutable:   make(MutKeyTypeMap),
			delta:     make(DeltaTypeMap),
			ctx:       NewContext(),
		}
	}

	db.sync_chan = make(chan *msg)
	db.op_chan = make(chan *stateOperation)
	db.quit = make(chan chan error)
	db.init_chan = make(chan chan error)

	timeline = NewTimeLine()

	cnx := NewCommitNexus()
	go commitLoop(fs, cnx)

	mnx := NewModelNexus()
	go educate(model, monitor, mnx, bid)

	go stateLoop(db, mnx, cnx, trace)
	return db, db.restored, nil
}

func (db *StateDB) Types() []string {

	var ts []string
	for t, _ := range db.immutable {
		ts = append(ts, t)
	}
	return ts
}

type stateOperation struct {
	kt     *KeyType
	imm    []byte
	mut    *MutState
	action int
	err    chan error
}

// Sync is a call for consistency; if the monitor has signalled a checkpoint
// a checkpoint will be committed. During this time, we cannot allow any processes to
// write or delete objects in the database.
func (db *StateDB) Sync() error {
	// response channel
	err := make(chan error)
	c := timeline.Tick()
	c.SyncStart()
	db.sync_chan <- &msg{
		time:    time.Now(),
		err:     err,
		t:       c,
		cptType: NONDETERMCPT,
	}
	e := <-err
	c.SyncEnd()
	if e == ActiveCommitError {
		return nil
	}
	return e
}

func (db *StateDB) forceCheckpoint() error {
	err := make(chan error)
	c := timeline.Tick()
	c.SyncStart()
	db.sync_chan <- &msg{
		time:     time.Now(),
		err:      err,
		forceCPT: true,
		cptType:  NONDETERMCPT,
		t:        c,
	}
	e := <-err
	c.SyncEnd()
	if e == ActiveCommitError {
		return nil
	}
	return e
}

func (db *StateDB) forceDeltaCPT() error {
	err := make(chan error)
	c := timeline.Tick()
	c.SyncStart()
	db.sync_chan <- &msg{
		time:     time.Now(),
		err:      err,
		forceCPT: true,
		cptType:  DELTACPT,
		t:        c,
	}
	c.SyncEnd()
	return <-err
}

func (db *StateDB) forceZeroCPT() error {
	err := make(chan error)
	c := timeline.Tick()
	c.SyncStart()
	db.sync_chan <- &msg{
		time:     time.Now(),
		err:      err,
		forceCPT: true,
		cptType:  ZEROCPT,
		t:        c,
	}
	c.SyncEnd()
	return <-err
}

func (db *StateDB) forceZeroCPTBlock() error {

	errChan := make(chan error)
	commitBlock := make(chan error)

	c := timeline.Tick()
	c.SyncStart()
	m := &msg{
		time:     time.Now(),
		err:      errChan,
		forceCPT: true,
		cptType:  ZEROCPT,
		t:        c,
		waitChan: commitBlock,
	}
	db.sync_chan <- m

	if err := <-errChan; err != nil {
		if err == ActiveCommitError {
			// if we sent a final commit while another commit
			// was being checkpointed, the commitBlock
			// will be signalled once *any* commit returns
			<-commitBlock
			db.sync_chan <- m
			err = <-errChan
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	c.SyncEnd()
	// block until the final checkpoint is completed
	return <-commitBlock
}

// When called, all checkpointing is shut down,
// and a final, full checkpoint is written to disk.
func (db *StateDB) Commit() error {

	// force a zero checkpoint and wait
	// till the checkpoint has been fully committed
	err := db.forceZeroCPTBlock()
	if err != nil {
		return err
	}
	fmt.Println("sending quit signal..")
	// send an error chan on which to respond in the
	// case of shut down issues
	errChan := make(chan error)
	db.quit <- errChan

	return <-errChan
}

// called from stateLoop to guarantee there is no race condition
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
