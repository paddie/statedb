package statedb

import (
	"bytes"
	"encoding/gob"
)

func decodeImmutable(data []byte) (ImmKeyTypeMap, error) {

	buff := bytes.NewBuffer(data)

	imm := new(ImmKeyTypeMap)
	enc := gob.NewDecoder(buff)
	if err := enc.Decode(imm); err != nil {
		return nil, err
	}

	return *imm, nil
}

func decodeDelta(data []byte) (DeltaTypeMap, error) {

	buff := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buff)
	d := new(deltaID)
	if err := dec.Decode(d); err != nil {
		return nil, err
	}

	return d.Delta, nil
}

func decodeMutable(data []byte) (MutKeyTypeMap, error) {

	buff := bytes.NewBuffer(data)

	mut := &mutableID{}
	enc := gob.NewDecoder(buff)
	if err := enc.Decode(mut); err != nil {
		return nil, err
	}

	return mut.Mutable, nil
}
