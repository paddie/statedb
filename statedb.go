package statedb

import (
	"bytes"
	"fmt"
	"reflect"
	"time"
	// "strconv"
	"encoding/gob"
	"errors"
	"log"
	"sync"
)

const (
	REMOVE  int = -1
	INSERT  int = 1
	RESTORE int = 0
)

type StateDB struct {
	restored bool     // has statedb just been restored
	ready    bool     // have all mutable objects been restored?
	ctx      *Context // cpt and restore information
	// State databases
	immutable ImmKeyTypeMap // immutable states
	delta     DeltaTypeMap  // static state delta
	mutable   MutKeyTypeMap // mutable state
	// Synchronization channels
	op_chan      chan *StateOperation // handles insert and remove operations
	quit         chan chan error      // shutdown signals
	cpt_chan     chan time.Time       // checkpoint signals are sent on this channel
	cpt_notice   bool                 // true after a signal has come in on cpt_chan
	sync_chan    chan *Msg            // consistent state signals are sent on this channel
	sync.RWMutex                      // for synchronizing things that don't need the channels..
}

func (db *StateDB) readyCheckpoint() bool {

	db.Lock()
	defer db.Unlock()

	if db.ready || !db.restored {
		return true
	}

	for _, vt := range db.mutable {
		for _, vs := range vt {
			if vs.v.IsValid() {
				return false
			}
		}
	}
	// every mutable object has been restored
	db.ready = true
	return true

}

func NewStateDB(volume, dir, suffix string) (*StateDB, error) {

	ctx, err := NewContext(volume, dir, suffix)
	if err != nil {
		return nil, err
	}

	db := &StateDB{
		ctx:       ctx,
		immutable: make(ImmKeyTypeMap),
		mutable:   make(MutKeyTypeMap),
		delta:     make(DeltaTypeMap),
		cpt_chan:  make(chan time.Time),
		sync_chan: make(chan *Msg),
		op_chan:   make(chan *StateOperation),
		quit:      make(chan chan error),
	}
	go db.StateSelect()
	// No need to restore
	if !ctx.previous {
		return db, nil
	}

	// fmt.Println("StateDB: must be restored!")
	log.Println("StateDB: Previous checkpoint %s. Attempting to restore..", ctx.CheckpointDir())

	if err = db.ctx.RestoreStateDB(db); err != nil {
		return nil, err
	}
	db.restored = true

	return db, nil
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
func (db *StateDB) Sync() error {
	// response channel
	err := make(chan error)
	db.sync_chan <- &Msg{
		time: time.Now(),
		err:  err,
	}
	return <-err
}

func (db *StateDB) Checkpoint() error {
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

func (db *StateDB) StateSelect() {
	for {
		select {
		case msg := <-db.sync_chan:
			// check if all mutable objects have been restored
			if !db.readyCheckpoint() {
				msg.err <- errors.New("StateDB.Checkpoint: Some objects have not been restored")
			}
			// if the checkpoint is not forced or
			// the monitor has not given a cpt_notice
			// the sync simply returns
			if !msg.forceCPT && !db.cpt_notice {
				msg.err <- nil
				continue
			}
			// reset notice
			// stupid heuristic for when to checkpoint
			msg.err <- db.checkpoint()
			db.cpt_notice = false
		case <-db.cpt_chan:
			// set the checkpoint notice to force a checkpoint
			// in the next consistent state
			db.cpt_notice = true
		case so := <-db.op_chan:
			kt := so.kt
			if so.action == REMOVE {
				fmt.Println("received delete: " + kt.String())
				so.err <- db.remove(kt)
			} else if so.action == INSERT {
				fmt.Println("received insert: " + kt.String())
				so.err <- db.insert(kt, so.imm, so.mut)
			} else {
				so.err <- fmt.Errorf("Unknown Action: %d", so.action)
			}
		case err_chan := <-db.quit:
			fmt.Println("Committing final checkpoint..")
			err_chan <- db.fullCheckpoint()
			fmt.Println("Checkpoint committed. Shutting down..")
			break
		}
	}
}

func (db *StateDB) restoreUpdate(kt *KeyType, mut *MutState) error {

	if !db.mutable.contains(kt) {
		return errors.New("Update: Cannot update non-existant mutable " + kt.String())
	}

	db.insertMutable(kt, mut)

	return nil
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
		return fmt.Errorf("StateDB.Remove: KeyType %s does not exist", kt.String())
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

// --------------------------
// Immutable Data Structures
// --------------------------
// TypeMap is the type used for the immutable and mutable fields
// - unabstracted type: map[type string][key Key]*State
type ImmKeyTypeMap map[string]ImmStateMap

// Map from Key (string or int) to *State
type ImmStateMap map[Key]*ImmState

type ImmState struct {
	KT  KeyType // save keytype to match against in testing
	Val []byte  // serialised object
}

func (m ImmKeyTypeMap) contains(kt *KeyType) bool {
	if sm, ok := m[kt.TypeID()]; ok {
		_, ok = sm[*kt.Key()]
		return ok
	}
	return false
}

func (m ImmKeyTypeMap) lookup(kt *KeyType) *ImmState {
	if sm, ok := m[kt.TypeID()]; ok {
		return sm[*kt.Key()]
	}
	return nil
}

func (m ImmKeyTypeMap) insert(kt *KeyType, val []byte) {

	// allocate KeyMap if nil
	if _, ok := m[kt.TypeID()]; !ok {
		m[kt.TypeID()] = make(ImmStateMap)
	}

	m[kt.TypeID()][kt.K] = &ImmState{
		KT:  *kt,
		Val: val,
	}
}

func (m ImmKeyTypeMap) remove(kt *KeyType) {

	if sm, ok := m[kt.T]; ok {
		delete(sm, kt.K)

		// clen up if there are no more items of this type
		if len(m[kt.T]) == 0 {
			delete(m, kt.T)
		}
	}
}

// --------------------------
// Mutable Data Structures
// --------------------------
type MutKeyTypeMap map[string]MutStateMap

type MutStateMap map[Key]*MutState

type MutState struct {
	KT  KeyType
	Val []byte
	v   reflect.Value // pointer to the latest update
}

// When checkpointing the system, encode the non-public interface into the Val,
// followed by a normal encoding of the struct
func (m *MutState) GobEncode() ([]byte, error) {

	if !m.v.IsValid() {
		return nil, fmt.Errorf("Trying to checkpoint a mutable state with a pointer from a previous ceckpoint")
	}

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.EncodeValue(m.v.Elem()); err != nil {
		return nil, err
	}
	m.Val = b.Bytes()

	var buff bytes.Buffer

	enc = gob.NewEncoder(&buff)
	if err := enc.Encode(m.KT); err != nil {
		return nil, err
	}
	if err := enc.Encode(m.Val); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func (m *MutState) GobDecode(b []byte) error {
	buff := bytes.NewBuffer(b)
	enc := gob.NewDecoder(buff)

	if err := enc.Decode(&m.KT); err != nil {
		return err
	}

	if err := enc.Decode(&m.Val); err != nil {
		return err
	}
	return nil
}

func (m MutKeyTypeMap) contains(kt *KeyType) bool {
	if sm, ok := m[kt.TypeID()]; ok {
		_, ok = sm[*kt.Key()]
		return ok
	}
	return false
}

func (m MutKeyTypeMap) lookup(kt *KeyType) *MutState {
	if sm, ok := m[kt.TypeID()]; ok {
		return sm[*kt.Key()]
	}
	return nil
}

func (m MutKeyTypeMap) insert(kt *KeyType, mut *MutState) {

	if _, ok := m[kt.TypeID()]; !ok {
		m[kt.TypeID()] = make(MutStateMap)
	}
	m[kt.TypeID()][kt.K] = mut
	// m[kt.TypeID()][kt.K] = &MutState{
	// 	KT:  kt,
	// 	Val: nil,
	// 	v:   v,
	// }
}

func (m MutKeyTypeMap) remove(kt *KeyType) {
	if sm, ok := m[kt.T]; ok {
		delete(sm, kt.K)

		// clen up if there are no more items of this type
		if len(m[kt.T]) == 0 {
			delete(m, kt.T)
		}
	}
}

// --------------------------
// Delta Data Structures
// --------------------------
type DeltaTypeMap map[string]DeltaStateOpMap

type DeltaStateOpMap map[Key]*StateOp

type StateOp struct {
	KT     KeyType
	Action int // DELETE=-1 or CREATE=1
	Val    []byte
}

func (m DeltaTypeMap) contains(kt *KeyType) bool {
	if sm, ok := m[kt.TypeID()]; ok {
		_, ok = sm[*kt.Key()]
		return ok
	}
	return false
}

func (m DeltaTypeMap) lookup(kt *KeyType) *StateOp {
	if sm, ok := m[kt.TypeID()]; ok {
		return sm[*kt.Key()]
	}
	return nil
}

func (m DeltaTypeMap) insert(kt *KeyType, val []byte) {

	// allocate KeyMap if nil
	if _, ok := m[kt.TypeID()]; !ok {
		m[kt.TypeID()] = make(DeltaStateOpMap)
	}

	m[kt.TypeID()][kt.K] = &StateOp{
		KT:     *kt,
		Val:    val,
		Action: INSERT,
	}
}

func (m DeltaTypeMap) remove(kt *KeyType) {
	if _, ok := m[kt.TypeID()]; !ok {
		m[kt.TypeID()] = make(DeltaStateOpMap)
	}

	if _, ok := m[kt.T][kt.K]; ok {
		// delet the entry if it already exists
		delete(m[kt.T], kt.K)
		// if there are no more objects of this type;
		// delete the type entry
		// - prevents creating empty maps
		if len(m[kt.T]) == 0 {
			delete(m, kt.T)
		}
	} else {
		// insert a DELETE entry for this keytype
		m[kt.T][kt.K] = &StateOp{
			KT:     *kt,
			Action: REMOVE,
			Val:    nil,
		}
	}
}
