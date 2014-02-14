package statedb

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
)

type entry struct {
	imm []byte
	kt  *KeyType
}

type Iterator struct {
	i       int
	entries []*entry
}

func (it *Iterator) Next(imm interface{}) (*KeyType, bool) {

	if it == nil {
		return nil, false
	}

	if it.i == len(it.entries) {
		it.entries = nil
		return nil, false
	}
	entry := it.entries[it.i]

	// immutable
	buff := bytes.NewBuffer(entry.imm)
	dec := gob.NewDecoder(buff)

	if err := dec.Decode(imm); err != nil {
		fmt.Println(err)
		return nil, false
	}
	it.i++

	return entry.kt, true
}

func (db *StateDB) RestoreImmIter(typeID string) (*Iterator, error) {

	if db == nil {
		return nil, errors.New("StateDB: database has not been initialized. Call NewStateDB(...)")
	}

	states, ok := db.immutable[typeID]
	if !ok {
		return nil, fmt.Errorf("StateDB.RestoreImmIter: TypeID '%s' does not exist", typeID)
	}
	entries := []*entry{}
	for _, state := range states {
		mp := &entry{
			imm: state.Val,
			kt:  &state.KT,
		}
		entries = append(entries, mp)
	}

	return &Iterator{entries: entries}, nil
}

func (db *StateDB) RestoreMutable(kt *KeyType, mut interface{}) error {

	mv_ptr := reflect.ValueOf(mut)

	if mv_ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("StateDB.RestoreMutable: %s is not a pointer to a state", mv_ptr.String())
	}

	mv := mv_ptr.Elem()

	if !mv.CanSet() {
		return fmt.Errorf("StateDB.RestoreMutable: %s is not settable", mv.String())
	}

	s := db.mutable.lookup(kt)
	if s == nil {
		return fmt.Errorf("StateDB.RestoreMutable: No mutable state exists for keytype = %s", kt.String())
	}
	if s.Val == nil {
		return fmt.Errorf("StateDB.RestoreMutable: There is nothing to restore for keytype %s", kt.String())
	}
	s.v = mv_ptr

	buff := bytes.NewBuffer(s.Val)
	dec := gob.NewDecoder(buff)

	return dec.DecodeValue(mv)
}

func (ctx *Context) Restore(db *StateDB) error {

	if db == nil {
		return errors.New("StateDB has not been initialized yet")
	}

	if !IsValidCheckpoint(ctx.cpt_dir) {
		return errors.New("StateDB.Restore: invalid checkpoint directory: " + ctx.cpt_dir)
	}

	imm, err := ctx.restoreImmutable()
	if err != nil {
		log.Println("StateDB.Restore: Failed to restore immutable from " + ctx.cpt_dir)
		return err
	}

	// restore mutable part of the checkpoint
	mut, mut_id, err := ctx.restoreMutable()
	if err != nil {
		log.Println("StateDB.Restore: Failed to restore mutable from " + ctx.cpt_dir)
		return err
	}

	db.immutable = imm
	db.mutable = mut

	// restore and replay the delta commits
	deltas, delta_id, err := ctx.restoreDelta()
	if err != nil {
		return nil
	}

	// if the delta id is different from the mutable id, something has gone wrong.
	if mut_id != delta_id {
		return fmt.Errorf("StateDB.Restore: mutable_id '%d' != '%d' delta_id. A delta commit was incomplete, or the delta.cpt file is corrupt.\n", mut_id, delta_id)
	}

	// fmt.Println("Restored immutable and mutable from ", ctx_dir, ". Proceeding with delta..")

	if err := db.replayDeltas(deltas); err != nil {
		return err
	}

	return nil
}

func (ctx *Context) restoreImmutable() (ImmKeyTypeMap, error) {

	path := ctx.ImmutablePath()

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var immutable ImmKeyTypeMap
	enc := gob.NewDecoder(file)
	if err = enc.Decode(&immutable); err != nil {
		return nil, err
	}

	return immutable, nil
}

func (ctx *Context) restoreMutable() (MutKeyTypeMap, int, error) {

	path := ctx.MutablePath()

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}

	mut := &mutableID{}
	enc := gob.NewDecoder(file)
	if err = enc.Decode(mut); err != nil {
		return nil, 0, err
	}

	return mut.Mutable, mut.DeltaDiff, nil
}

func (ctx *Context) restoreDelta() ([]DeltaTypeMap, int, error) {

	path := ctx.DeltaPath()

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	dec := gob.NewDecoder(file)

	// initial values
	var deltas []DeltaTypeMap
	id := 0

	// the object to decode into
	mut := new(deltaID)

	// keep reading until it fails
	for {
		mut = new(deltaID)
		if err = dec.Decode(mut); err != nil {
			if err != io.EOF {
				continue
			}
			break
		}
		id = mut.DeltaDiff
		deltas = append(deltas, mut.Delta)
	}

	log.Printf("Read %d incremental checkpoints\n", len(deltas))

	return deltas, id, nil
}

func (db *StateDB) replayDeltas(deltas []DeltaTypeMap) error {

	// there is nothing to replay
	if len(deltas) == 0 {
		return nil
	}

	// for every type of state in the delta
	for _, delta := range deltas {
		for _, m := range delta {
			// for every StateOp in Delta
			for _, st_op := range m {
				if st_op.Action == DELETE {
					// remove immutable and mutable and register in delta
					if !db.immutable.contains(&st_op.KT) {
						return errors.New("StateDB.Replay: Trying to replay DELETE of non-existing KeyType:" + st_op.KT.String())
					}
					db.immutable.remove(&st_op.KT)
					// db.mutable.remove(kt)
				} else {
					if db.immutable.contains(&st_op.KT) {
						return errors.New("StateDB.Replay: Trying to replay CREATE of already existing KeyType:" + st_op.KT.String())
					}
					db.insertImmutable(&st_op.KT, st_op.Val)
				}
			}
		}
	}

	// validate
	immSize := 0
	for _, t := range db.immutable {
		immSize += len(t)
	}
	mutSize := 0
	for _, t := range db.mutable {
		mutSize += len(t)
	}

	if mutSize > immSize {
		panic("StateDB.replayDelta: The number of mutable states > immutable ones.")
	}

	db.delta = nil

	return nil
}

// The argument should be a current cpt_dir:
//  <bucket>/<dir>/<suffix>/<cpt_id>
// - if the immutable checkpoint is missing, it is not a valid checkpoint
// - if a delta checkpoint exists, there
func IsValidCheckpoint(path string) bool {
	fmt.Println("checking for valid checkpoint in " + path)

	if !IsDir(path) {
		return false
	}
	if !IsFile(path + "/immutable.cpt") {
		return false
	}

	return true
}

// Get the ID of the most recent full commit in the ctx.full folder.
// Returns 0 if no valid full commit exists in the context.
func (ctx *Context) probeCptId() int {
	files, err := ioutil.ReadDir(ctx.full)
	if err != nil {
		return 0
	}

	if len(files) == 0 {
		return 0
	}

	max := -1
	folders := 0
	for _, f := range files {
		if f.IsDir() {
			folders++
			if id, err := strconv.Atoi(f.Name()); err == nil {
				if id > max {
					max = id
				}
			}
		}
	}
	// if none of the files in ctx.dir were cpt folders
	if folders == 0 || max <= 0 {
		return 0
	}

	// set current cpt id in context
	return max
}
