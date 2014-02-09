package statedb

import (
	"bytes"
	// "fmt"
	"reflect"
	// "strconv"
	"encoding/gob"
	// "errors"
	"log"
	"sync"
)

const (
	DELETE int = -1
	CREATE int = 1
)

type StateDB struct {
	restored   bool     // has statedb just been restored
	consistent bool     // SignalConsistent() == true
	ctx        *Context // cpt and restore information
	// version    uint64        // Increase for every consistent state
	immutable ImmKeyTypeMap // immutable states
	delta     DeltaTypeMap  // static state delta
	mutable   MutKeyTypeMap // mutable state
	sync.RWMutex
}

func NewStateDB(bucket, dir, suffix string) (*StateDB, error) {

	ctx, restore, err := NewContext(bucket, dir, suffix)
	if err != nil {
		return nil, err
	}

	db := &StateDB{
		ctx:       ctx,
		immutable: make(ImmKeyTypeMap),
		mutable:   make(MutKeyTypeMap),
		delta:     make(DeltaTypeMap),
	}
	// No need to restore
	if !restore {
		return db, nil
	}

	// fmt.Println("StateDB: must be restored!")
	log.Println("StateDB: Previous checkpoint detected. Trying to restore..")

	if err = db.ctx.Restore(db); err != nil {
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

func (db *StateDB) IsConsistent() bool {
	db.RLock()
	defer db.RUnlock()

	return db.consistent
}

func (db *StateDB) SetConsistent() {
	db.Lock()
	defer db.Unlock()

	db.consistent = true
}

// The Version is of type uint64, meaning it will loop around
// when it overflows. This is intentional.
// func (db *StateDB) Version() int64 {
// 	return int64(db.version)
// }

func (db *StateDB) ContainsKeyType(kt *KeyType) bool {

	db.RLock()
	defer db.RUnlock()

	return db.immutable.contains(kt)

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

	s := &ImmState{
		KT:  *kt,
		Val: val,
	}

	m[kt.TypeID()][kt.K] = s
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

func (m MutKeyTypeMap) insert(kt *KeyType, v reflect.Value) error {

	if _, ok := m[kt.TypeID()]; !ok {
		m[kt.TypeID()] = make(MutStateMap)
	}

	s := &MutState{
		KT:  *kt,
		Val: nil,
		v:   v,
	}

	m[kt.TypeID()][kt.K] = s

	return nil
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

	s := &StateOp{
		KT:     *kt,
		Action: CREATE,
		Val:    val,
	}

	m[kt.TypeID()][kt.K] = s
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
			Action: DELETE,
			Val:    nil,
		}
	}
}
