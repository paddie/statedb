package statedb

import (
	// "io"
	// "buffer"
	// "io/ioutil"
	// "strconv"

	"encoding/gob"
	"errors"
	// "fmt"
	"log"
	"os"
	// "path"
)

func (db *StateDB) checkpoint() error {

	if len(db.immutable) == 0 {
		log.Println("StateDB: There is nothing to checkpoint")
		return nil
	}

	if db.ctx.rcid == 0 || db.ctx.dcnt > 5 {
		return db.fullCheckpoint()
	}

	return db.incrementalCheckpoint()
}

func (db *StateDB) incrementalCheckpoint() error {

	if len(db.delta) == 0 && len(db.mutable) == 0 {
		return nil
	}

	if err := db.ctx.commitIncrementalCpt(db); err != nil {
		return err
	}

	// reset delta after an incremental checkpoint
	db.delta = nil

	return nil
}

// The FullCheckpoint serves as a forced checkpoint of all the known states
// - Assumes that the system is in a consistent state
// - Checkpoints the Immutable and Mutable states, and empties the delta log.
func (db *StateDB) fullCheckpoint() error {

	// db.Lock()
	// defer db.Unlock()

	// if !db.consistent {
	// 	return errors.New("Context: StateDB is not in a consistent state. Use Consistent() to signal that to the database")
	// }

	// nothing to commit, but not an error
	if len(db.immutable) == 0 {
		log.Println("StateDB: There is nothing to commit")
		// return errors.New("StateDB: There is nothing to commit")
		return nil
	}

	if err := db.ctx.commitFullCheckpoint(db); err != nil {
		return err
	}

	// if the checkpoint was successfull => reset delta log
	db.delta = nil

	return nil
}

// Should only succeed if both immutable and mutable checkpoints are succesfully
// committed to disk.
func (ctx *Context) commitFullCheckpoint(db *StateDB) error {

	if ctx.restored {
		return errors.New("Context: A previous checkpoint has not been restored. Checkpoint would overwrite previous checkpoint.")
	}

	// Create a temporary context with an updated cpt_id
	// 1. create the directories associated with the full commit
	// 2. commit immutable
	// 3. commit mutabe
	// 4. if everything succeeds, replace context with the temporary one
	// 5. if not, clean up the temporary dirs we created..
	tmp := ctx.newContextWithId(ctx.rcid + 1)
	tmp.dcnt = 0
	if err := tmp.prepareDirectories(); err != nil {
		return err
	}

	if err := tmp.commitImmutable(db.immutable); err != nil {
		// 1. delete new checkpoint dir
		// 2. decrease checkpoint id
		return err
	}

	if err := tmp.commitMutable(db.mutable); err != nil {
		// 1. delete new checkpoint dir
		// 2. decrease checkpoint id
		return err
	}
	// replace original context if commit succeeded
	// - eliminates the need for playback in case of error
	*ctx = *tmp

	return nil
}

// Every time an Incremental checkpoint is generated, the delta is appended to the
// 'delta.cpt' file. The mutable table is committed in its full (but might use a swap file at some point).
// - Before an incremental checkpoint can be performed, a reference full backup
//   MUST precede it.
// - If there is no reference checkpoint, it will return an error.
func (ctx *Context) commitIncrementalCpt(db *StateDB) error {

	// ctx.Lock()
	// defer ctx.Unlock()

	if ctx.restored {
		return errors.New("Context: A previous checkpoint has not been restored. Checkpoint would overwrite previous checkpoint.")
	}

	if ctx.rcid == 0 {
		return errors.New("Context: A *full* checkpoint MUST be committed prior to any delta checkpoint")
	}

	tmp := ctx.copyContext()
	tmp.dcnt++

	if err := tmp.commitDelta(db.delta); err != nil {
		return err
	}

	if err := tmp.commitMutable(db.mutable); err != nil {
		return err
	}

	*ctx = *tmp
	// update delta diff so we can track the number
	// of delta diffs between full checkpoints

	return nil
}

func (ctx *Context) commitImmutable(immutable ImmKeyTypeMap) error {

	// Commit mutable
	i_path := ctx.ImmutablePath()

	i_file, err := os.Create(i_path)
	if err != nil {
		return err
	}
	defer i_file.Close()

	enc := gob.NewEncoder(i_file)
	if err = enc.Encode(immutable); err != nil {
		return err
	}

	return nil
}

type mutableID struct {
	DCNT    int
	RCID    int
	Mutable MutKeyTypeMap
}

// Overwrites any existing mutable checkpoint in the current
// checkpoint id directory
func (ctx *Context) commitMutable(mutable MutKeyTypeMap) error {

	// Commit mutable
	m_path := ctx.MutablePath()

	m_file, err := os.Create(m_path)
	if err != nil {
		return err
	}
	defer m_file.Close()

	// fmt.Println("commitMutable: ", mutable)

	// if there is nothing to commit, only commit the cpt id
	wrap := &mutableID{
		DCNT: ctx.dcnt,
		RCID: ctx.rcid,
	}
	if len(mutable) == 0 {
		wrap.Mutable = nil
	} else {
		wrap.Mutable = mutable
	}

	enc := gob.NewEncoder(m_file)
	if err = enc.Encode(wrap); err != nil {
		return err
	}

	return nil
}

type deltaID struct {
	DCNT  int
	RCID  int
	Delta DeltaTypeMap
}

// A delta checkpoint writes the delta and mutable to disk.
// It does not increment the ctx.cptId, and requires there
// to be an existing full commit as a reference.
// The delta and mutable checkpoints are wrapped in a checkpoint id
// to enable lock-step recovery and to make sure that the mutable checkpoint
// matches up to the delta.
// - The delta is appended to 'delta.cpt'
// - The mutable is written to 'mutable.cpt'
// - The mutable is continuously replaced in every checkpoint
//   TODO: make a swap file for the dynamic part.
func (ctx *Context) commitDelta(delta DeltaTypeMap) error {

	// fmt.Printf("Committing delta: %v (%d)\n", delta, len(delta))
	// Commit delta
	d_path := ctx.DeltaPath()

	// Create file or append to existing
	d_file, err := os.OpenFile(d_path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer d_file.Close()

	var d_wrap *deltaID
	if len(delta) == 0 {
		d_wrap = &deltaID{ctx.dcnt, ctx.rcid, nil}
	} else {
		d_wrap = &deltaID{ctx.dcnt, ctx.rcid, delta}
	}

	enc := gob.NewEncoder(d_file)
	if err = enc.Encode(d_wrap); err != nil {
		return err
	}

	log.Println("Delta checkpoint committed to: " + d_path)

	return nil
}
