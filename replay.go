package statedb

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	// "io"
	// "io/ioutil"
	// "log"
	// "os"
	// "reflect"
	// "sync"
)

var (
	NoCheckpoint = errors.New("No Previous Checkpoint")
)

func (ctx *Context) Restore(db *StateDB) error {

	info, err := ctx.retrieveInfo()
	if err != nil {
		// No previous checkpoint was registered
		return err
	}

	// update context info
	ctx.info = *info
	imm, err := ctx.retrieveImmutable()
	if err != nil {
		return err
	}
	db.immutable = imm

	if ctx.MCNT() > 0 {
		mut, err := ctx.retrieveMutable()
		if err != nil {
			db.immutable = nil
			return err
		}
		db.mutable = mut
	}
	db.restored = true

	if ctx.DCNT() == 0 {
		return nil
	}

	deltas, err := ctx.retrieveDeltas()
	if err != nil {
		db.immutable = nil
		db.mutable = nil
		db.delta = nil
		return err
	}

	if err := db.replayDeltas(deltas); err != nil {
		db.immutable = nil
		db.mutable = nil
		db.delta = nil
		return err
	}

	return nil
}

func (ctx *Context) retrieveInfo() (*CptInfo, error) {
	data, err := ctx.fs.Get("cpt.nfo")
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buff)

	info := &CptInfo{}
	if err := dec.Decode(info); err != nil {
		return nil, err
	}

	return info, nil
}

func (i CptInfo) ImmPath() string {
	return fmt.Sprintf("/%d/imm.cpt", i.RCID)
}

func (i CptInfo) MutPath() string {
	return fmt.Sprintf("/%d/mut_%d.cpt", i.RCID, i.MCNT)
}

func (info CptInfo) DeltaPaths() []string {
	paths := make([]string, 0, info.DCNT)
	for i := 0; i < info.DCNT; i++ {
		paths = append(paths, fmt.Sprintf("/%d/del_%d.cpt", info.RCID, i))
	}

	return paths
}

func (info CptInfo) DelPath() string {
	return fmt.Sprintf("/%d/del_%d.cpt", info.DCNT, info.DCNT)
}

func (ctx *Context) retrieveImmutable() (ImmKeyTypeMap, error) {
	path := ctx.info.ImmPath()
	data, err := ctx.fs.Get(path)
	if err != nil {
		return nil, err
	}

	return decodeImmutable(data)
}

func (ctx *Context) retrieveMutable() (MutKeyTypeMap, error) {
	path := ctx.info.MutPath()
	data, err := ctx.fs.Get(path)
	if err != nil {
		return nil, err
	}

	return decodeMutable(data)
}

type DeltaGet struct {
	id   int
	data DeltaTypeMap
	err  error
}

func (ctx *Context) retrieveDeltas() ([]DeltaTypeMap, error) {

	paths := ctx.info.DeltaPaths()
	deltas := make([]DeltaTypeMap, len(paths))

	res := make(chan *DeltaGet, 0)
	for i, path := range paths {
		go func(path string, id int) {
			dg := &DeltaGet{
				id: id,
			}
			// retrieve binary data from fs
			data, err := ctx.fs.Get(path)
			if err != nil {
				dg.err = err
				res <- dg
				return
			}
			// decode into delta structure
			tm, err := decodeDelta(data)
			if err != nil {
				dg.err = err
				res <- dg
				return
			}

			dg.data = tm

			res <- dg
		}(path, i)
	}

	for i := 0; i < len(paths); i++ {
		dg := <-res
		if dg.err != nil {
			return nil, dg.err
		}
		deltas[dg.id] = dg.data
	}
	return deltas, nil
}

// func (ctx *Context) RestoreStateDB(db *StateDB) error {

// 	if db == nil {
// 		return errors.New("StateDB has not been initialized yet")
// 	}

// 	info, err := ctx.retrieveInfo()
// 	if err != nil {
// 		// No previous checkpoint was recovered
// 		return err
// 	}
// 	ctx.info = info

// 	// if !IsValidCheckpoint(ctx.CheckpointDir()) {
// 	// 	return errors.New("StateDB.Restore: invalid checkpoint directory: " + ctx.CheckpointDir())
// 	// }

// 	imm, err := ctx.restoreImmutable()
// 	if err != nil {
// 		log.Println("StateDB.Restore: Failed to decode immutable checkpoint from " + ctx.CheckpointDir())
// 		return err
// 	}

// 	// restore mutable part of the checkpoint
// 	mut, mcnt, dcnt_max, err := ctx.loadMutable()
// 	if err != nil {
// 		log.Println("StateDB.Restore: Failed to restore mutable from " + ctx.CheckpointDir())
// 		return err
// 	}
// 	ctx.mcnt = mcnt
// 	ctx.dcnt = dcnt_max

// 	db.immutable = imm
// 	db.mutable = mut

// 	db.restored = true

// 	// there are no delta checkpoints to log
// 	if dcnt_max == 0 {
// 		return nil
// 	}

// 	// restore and replay the delta commits
// 	deltas, err := ctx.loadDeltas(dcnt_max)
// 	if err != nil {
// 		return err
// 	}

// 	if err := db.replayDeltas(deltas); err != nil {
// 		return err
// 	}

// 	return nil
// }

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
