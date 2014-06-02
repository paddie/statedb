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

// func (iter *Iter) All(result interface{}) error {
// 	resultv := reflect.ValueOf(result)
// 	if resultv.Kind() != reflect.Ptr || resultv.Elem().Kind() != reflect.Slice {
// 		panic("result argument must be a slice address")
// 	}
// 	slicev := resultv.Elem()
// 	slicev = slicev.Slice(0, slicev.Cap())
// 	elemt := slicev.Type().Elem()
// 	i := 0
// 	for {
// 		if slicev.Len() == i {
// 			elemp := reflect.New(elemt)
// 			if !iter.Next(elemp.Interface()) {
// 				break
// 			}
// 			slicev = reflect.Append(slicev, elemp.Elem())
// 			slicev = slicev.Slice(0, slicev.Cap())
// 		} else {
// 			if !iter.Next(slicev.Index(i).Addr().Interface()) {
// 				break
// 			}
// 		}
// 		i++
// 	}
// 	resultv.Elem().Set(slicev.Slice(0, i))
// 	return iter.Close()
// }

func (it *Iterator) Next(imm interface{}) (*KeyType, bool) {

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

	m, ok := imm.(Mutable)
	if ok {
		mut := m.Mutable()
		if mut == nil {
			return entry.kt, true
		}

		if entry.mut == nil {
			panic("there should be something here!")
			return entry.kt, true
		}

		mutv := reflect.ValueOf(mut)
		if err := validateMutableEntry(mutv); err != nil {
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

func (db *StateDB) RestoreSingle(imm interface{}) error {

	typ := ReflectTypeM(imm)

	states, ok := db.immutable[typ]
	if !ok {
		return fmt.Errorf("StateDB.Restore: No object of type '%s' found", typ)
	}

	if len(states) == 0 {
		return errors.New("RestoreSigne: No item of type " + typ)
	}

	for _, s := range states {
		kt := &s.KT

		if err := Decode(s.Val, imm); err != nil {
			return err
		}

		m, ok := imm.(Mutable)
		if !ok {
			return nil
		}

		ms := db.mutable.lookup(kt)
		if ms.Val == nil {
			return nil
		}
		if ms != nil {
			mut := m.Mutable()
			mutv := reflect.ValueOf(mut)
			if err := validateMutableEntry(mutv); err != nil {
				return err
			}
			ms.v = mutv

			Decode(ms.Val, mut)
		}
		// break after one restore..
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
