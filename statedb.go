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

type CPTTYPE int

type OPERATION int

const (
	ZEROCPT CPTTYPE = iota
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

// If StateDB was restored, ensure that:
// 1) All previous mutable data has been restored
// 2) All mutable entries that have not been restored are deleted
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

func NewStateDB(fs Persistence, sched Schedular, monitor Monitor, bid float64, path string) (*StateDB, bool, error) {

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

	mnx := NewModelNexus(sched.Preemptive())
	go educate(sched, monitor, mnx, bid)

	go stateLoop(db, mnx, cnx, path)
	return db, db.restored, nil
}

func (db *StateDB) Types() []string {

	var ts []string
	for t, _ := range db.immutable {
		ts = append(ts, t)
	}
	return ts
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
