package statedb

import (
	"bytes"
	"encoding/gob"
)

func (ctx *Context) commitImmutable(immutable ImmKeyTypeMap) error {
	data, err := encodeImmutable(immutable)
	if err != nil {
		return err
	}
	path := ctx.info.ImmPath()
	return ctx.fs.Put(path, data)
}

func encodeImmutable(immutable ImmKeyTypeMap) ([]byte, error) {

	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(immutable); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil

}

// Overwrites any existing mutable checkpoint in the current
// checkpoint id directory
func (ctx *Context) commitMutable(mutable MutKeyTypeMap) error {
	data, err := encodeMutable(mutable, ctx.MCNT())
	if err != nil {
		return err
	}
	path := ctx.info.MutPath()
	return ctx.fs.Put(path, data)
}

type mutableID struct {
	// DCNT    int
	// RCID    int
	MCNT    int
	Mutable MutKeyTypeMap
}

func encodeMutable(mutable MutKeyTypeMap, mcnt int) ([]byte, error) {

	wrap := &mutableID{
		MCNT: mcnt,
	}

	if len(mutable) == 0 {
		wrap.Mutable = nil
	} else {
		wrap.Mutable = mutable
	}

	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(wrap); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

type deltaID struct {
	DCNT int
	// RCID  int
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
	path := ctx.info.DelPath()
	data, err := encodeDelta(delta, ctx.DCNT())
	if err != nil {
		return err
	}

	return ctx.fs.Put(path, data)
}

func encodeDelta(delta DeltaTypeMap, dcnt int) ([]byte, error) {

	d_wrap := &deltaID{
		DCNT: dcnt,
	}

	if len(delta) != 0 {
		d_wrap.Delta = delta
	}

	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(d_wrap); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil

}
