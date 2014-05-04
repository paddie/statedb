package statedb

import (
	"bytes"
	"encoding/gob"
	// "fmt"
	// "sync"
	"time"
)

func timedEncodeImmutable(immutable ImmKeyTypeMap) ([]byte, time.Duration, error) {

	now := time.Now()
	data, err := encodeImmutable(immutable)

	return data, now.Sub(time.Now()), err
}

func encodeImmutable(immutable ImmKeyTypeMap) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(immutable); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
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

	return encode(wrap)
}

type deltaID struct {
	DCNT int
	// RCID  int
	Delta DeltaTypeMap
}

func encodeDelta(delta DeltaTypeMap, dcnt int) ([]byte, error) {

	d_wrap := &deltaID{
		DCNT: dcnt,
	}

	if len(delta) != 0 {
		d_wrap.Delta = delta
	}

	return encode(d_wrap)
}

func encode(i interface{}) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(i); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}
