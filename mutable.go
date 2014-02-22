package statedb

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
)

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

func (m MutKeyTypeMap) count() int {

	if len(m) == 0 {
		return 0
	}

	cnt := 0
	for _, v := range m {
		cnt += len(v)
	}
	return cnt
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
	//  KT:  kt,
	//  Val: nil,
	//  v:   v,
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
