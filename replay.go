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
	NoCheckpointError = errors.New("No Previous Checkpoint")
)

func restore(fs Persistence) (*StateDB, error) {

	// retrieve the most recent context
	ctx, err := retrieveContext(fs)
	if err != nil {
		// No previous checkpoint was registered
		return nil, NoCheckpointError
	}

	db := &StateDB{
		ctx: ctx,
		// op_chan: make(chan *StateOperation),
		// quit:    make(chan chan error),
	}

	imm, err := retrieveImmutable(fs, ctx)
	if err != nil {
		return nil, err
	}
	db.immutable = imm

	if ctx.MCNT > 0 {
		mut, err := retrieveMutable(fs, ctx)
		if err != nil {
			db.immutable = nil
			return nil, err
		}
		db.mutable = mut
	}
	db.restored = true
	db.ctx = ctx

	if ctx.DCNT != 0 {
		deltas, err := retrieveDeltas(fs, ctx)
		if err != nil {
			return nil, err
		}

		if err := db.replayDeltas(deltas); err != nil {
			return nil, err
		}
	}

	return db, nil
}

func retrieveContext(fs Persistence) (*Context, error) {
	data0, err0 := fs.Get("cpt0.nfo")
	data1, err1 := fs.Get("cpt1.nfo")

	// if no ctx was found
	if err0 != nil && err1 != nil {
		return nil, err0
	}
	// if only one returned
	// => decode and return
	if err0 == nil && err1 != nil {
		return decodeContext(data0)
	}
	if err1 == nil && err0 != nil {
		return decodeContext(data1)
	}

	// two context were found
	// - deserialize and determine which is the most recent
	var c0, c1 *Context
	c0, err0 = decodeContext(data0)
	c1, err1 = decodeContext(data1)

	// if both failed, return error
	if err0 != nil && err1 != nil {
		return nil, err0
	}
	if err0 == nil && err1 != nil {
		return c0, nil
	}
	if err1 == nil && err0 != nil {
		return c1, nil
	}

	return MostRecent(c0, c1), nil
}

func decodeContext(data []byte) (*Context, error) {
	buff := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buff)

	ctx := new(Context)
	if err := dec.Decode(ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}

func retrieveImmutable(fs Persistence, ctx *Context) (ImmKeyTypeMap, error) {
	path := ctx.ImmPath()
	data, err := fs.Get(path)
	if err != nil {
		return nil, err
	}

	return decodeImmutable(data)
}

func retrieveMutable(fs Persistence, ctx *Context) (MutKeyTypeMap, error) {
	path := ctx.MutPath()
	data, err := fs.Get(path)
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

func retrieveDeltas(fs Persistence, ctx *Context) ([]DeltaTypeMap, error) {

	paths := ctx.DeltaPaths()
	deltas := make([]DeltaTypeMap, len(paths))

	res := make(chan *DeltaGet, 0)
	for i, path := range paths {
		go func(path string, id int) {
			dg := &DeltaGet{
				id: id,
			}
			// retrieve binary data from fs
			data, err := fs.Get(path)
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
