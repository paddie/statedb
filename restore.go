package statedb

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	// "io"
	// "io/ioutil"
	"log"
	"os"
	"reflect"
	// "strconv"
)

type entry struct {
	imm *ImmState
	mut *MutState
	kt  *KeyType
}

type Iterator struct {
	i       int
	entries []*entry
	db      *StateDB
}

func (it *Iterator) Next(imm, mut interface{}) (*KeyType, bool) {

	if it == nil {
		return nil, false
	}

	if it.i >= len(it.entries) {
		it.entries = nil
		return nil, false
	}
	entry := it.entries[it.i]

	// immutable
	buff := bytes.NewBuffer(entry.imm.Val)
	dec := gob.NewDecoder(buff)

	if err := dec.Decode(imm); err != nil {
		fmt.Println(err)
		return nil, false
	}
	it.i++

	if mut == nil {
		return entry.kt, true
	}

	if entry.mut == nil {
		panic("there should be something here!")
		return entry.kt, true
	}

	mutv := reflect.ValueOf(mut)
	if err := validateMutable(mutv); err != nil {
		return nil, false
	}

	// update the v with the new pointer value
	entry.mut.v = mutv

	// mutable
	buff = bytes.NewBuffer(entry.mut.Val)
	dec = gob.NewDecoder(buff)

	if err := dec.Decode(mut); err != nil {
		fmt.Println(err)
		return nil, false
	}

	return entry.kt, true
}

func Decode(val []byte, i interface{}) error {
	buff := bytes.NewBuffer(val)
	dec := gob.NewDecoder(buff)

	if err := dec.Decode(i); err != nil {
		return err
	}
	return nil
}

func (db *StateDB) RestoreSingle(imm, mut interface{}) error {

	t_str := ReflectType(imm)

	states, ok := db.immutable[t_str]
	if !ok {
		return fmt.Errorf("StateDB.Restore: No object of type '%s' found", t_str)
	}

	if len(states) != 1 {
		return errors.New("There are more than one item of this type, use RestoreIter to restore them all")
	}

	for _, s := range states {
		kt := &s.KT

		if err := Decode(s.Val, imm); err != nil {
			return err
		}
		ms := db.mutable.lookup(kt)
		if ms != nil {
			mutv := reflect.ValueOf(mut)
			if err := validateMutable(mutv); err != nil {
				return err
			}
			ms.v = mutv
			Decode(ms.Val, mut)
		}
		break
	}

	return nil
}

func (db *StateDB) RestoreIter(typeID string) (*Iterator, error) {

	if db == nil {
		return nil, errors.New("StateDB: database has not been initialized. Call NewStateDB(...)")
	}

	states, ok := db.immutable[typeID]
	if !ok {
		return nil, fmt.Errorf("StateDB.RestoreIter: TypeID '%s' does not exist", typeID)
	}
	entries := []*entry{}
	for _, state := range states {
		kt := &state.KT
		mp := &entry{
			imm: state,
			kt:  kt,
		}
		mp.mut = db.mutable.lookup(kt)

		entries = append(entries, mp)
	}

	return &Iterator{entries: entries}, nil
}

func (ctx *Context) RestoreStateDB(db *StateDB) error {

	if db == nil {
		return errors.New("StateDB has not been initialized yet")
	}

	if !IsValidCheckpoint(ctx.CheckpointDir()) {
		return errors.New("StateDB.Restore: invalid checkpoint directory: " + ctx.CheckpointDir())
	}

	imm, err := ctx.restoreImmutable()
	if err != nil {
		log.Println("StateDB.Restore: Failed to restore immutable from " + ctx.CheckpointDir())
		return err
	}

	// restore mutable part of the checkpoint
	mut, mcnt, dcnt_max, err := ctx.loadMutable()
	if err != nil {
		log.Println("StateDB.Restore: Failed to restore mutable from " + ctx.CheckpointDir())
		return err
	}

	ctx.mcnt = mcnt
	ctx.dcnt = dcnt_max

	db.immutable = imm
	db.mutable = mut

	// there are no delta checkpoints to log
	if dcnt_max == 0 {
		return nil
	}

	// restore and replay the delta commits
	deltas, err := ctx.loadDeltas(dcnt_max)
	if err != nil {
		log.Println("StateDB.Restore: " + err.Error())
		return nil
	}

	if err := db.replayDeltas(deltas); err != nil {
		return err
	}
	db.restored = true
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

func (ctx *Context) loadMutable() (MutKeyTypeMap, int, int, error) {

	mutables := listMutableCheckpoints(ctx.CheckpointDir())

	// ctx.mcnt = probeMCNT(ctx.CheckpointDir())
	// // if no mcnt is detected, no mutable checkpoint was committed.
	if len(mutables) == 0 {
		return nil, 0, 0, nil
	}
	path := mutables[len(mutables)-1]

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}

	mut := &mutableID{}
	enc := gob.NewDecoder(file)
	if err = enc.Decode(mut); err != nil {
		return nil, 0, 0, err
	}

	return mut.Mutable, mut.MCNT, mut.DCNT, nil
}

func (ctx *Context) loadDeltas(max_d int) ([]DeltaTypeMap, error) {

	d_paths := listDeltaCheckpoints(ctx.CheckpointDir())
	if len(d_paths) == 0 {
		return nil, nil
	}

	var ds []DeltaTypeMap
	dcnt := 0
	for _, path := range d_paths {
		d := new(deltaID)
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		dec := gob.NewDecoder(file)
		if err = dec.Decode(d); err != nil {
			return nil, err
		}
		ds = append(ds, d.Delta)
		dcnt = d.DCNT
	}

	if dcnt != max_d {
		return nil, fmt.Errorf("delta.DCNT %d does not match mutable.DCNT %d", dcnt, max_d)
	}

	return ds, nil
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
				if st_op.Action == REMOVE {
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
