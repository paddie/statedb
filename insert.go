package statedb

import (
	"bytes"
	"encoding/gob"
	"errors"
	// "fmt"
	"reflect"
)

// Deletes an object from StateDB
// 1. Delete object from immutable TypeMap
// 2. If the object CREATE is in Delta, delete the entry (object has now never existed)
// 3. If object is not in the Delat (meaning in was created in a previous CPT), insert a REMOVE entry for that particular KeyType in the Delta
// 4. If the key is not in StateDB, return error
// func (db *StateDB) Remove(kt *KeyType) error {

// 	if !db.immutable.contains(kt) {
// 		return fmt.Errorf("StateDB.Remove: KeyType %s does not exist", kt.String())
// 	}

// 	db.Lock()
// 	defer db.Unlock()

// 	db.immutable.remove(kt)
// 	db.delta.remove(kt)
// 	db.mutable.remove(kt)

// 	return nil
// }
// Deletes an object from StateDB
// 1. Delete object from immutable TypeMap
// 2. If the object CREATE is in Delta, delete the entry (object has now never existed)
// 3. If object is not in the Delat (meaning in was created in a previous CPT), insert a REMOVE entry for that particular KeyType in the Delta
// 4. If the key is not in StateDB, return error
func (db *StateDB) Unregister(kt *KeyType) error {

	if !kt.IsValid() {
		return errors.New("StateDB.Remove: invalid keytype " + kt.String())
	}

	err_chan := make(chan error)
	db.op_chan <- &stateOperation{
		kt:     kt,
		action: REMOVE,
		err:    err_chan,
	}

	return <-err_chan
}

func (db *StateDB) Register(i interface{}) (*KeyType, error) {

	// we allow for the mutable state to be <nil>
	if i == nil {
		return nil, errors.New("StateDB: an inserted immutable state cannot be <nil>")
	}

	kt, err := ReflectKeyTypeM(i)
	if err != nil {
		return nil, err
	}

	immv := Indirect(reflect.ValueOf(i))
	imm_d, err := encodeImmutableEntry(immv)
	if err != nil {
		return nil, err
	}

	err_chan := make(chan error)

	so := &stateOperation{
		kt:     kt,
		imm:    imm_d,
		action: INSERT,
		err:    err_chan,
	}
	// if the mutable state is nil, we also validate and encode it
	m, ok := i.(Mutable)
	if ok {
		mut := m.Mutable()
		if mut != nil {
			// fmt.Println("Inserting Mutable part of " + kt.String())
			mutv := reflect.ValueOf(mut)
			if err := validateMutableEntry(mutv); err != nil {
				return nil, err
			}
			so.mut = &MutState{
				KT:  *kt,
				Val: nil,
				v:   mutv,
			}
		}
	}

	// ship the operation to be inserted
	// fmt.Println("Inserting kt: ", kt.String())
	db.op_chan <- so

	// wait for response
	return kt, <-err_chan
}

func encodeImmutableEntry(immv reflect.Value) ([]byte, error) {

	if immv.Kind() == reflect.Ptr && immv.IsNil() {
		return nil, errors.New("StateDB: Cannot encode nil pointer of type " + immv.Type().String())
	}

	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)

	if err := enc.EncodeValue(immv); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func validateMutableEntry(mutv reflect.Value) error {
	if mutv.Kind() != reflect.Ptr {
		return errors.New("StateDB.Insert: Mutable MUST be a pointer")
	}
	if mutv.Kind() == reflect.Ptr && mutv.IsNil() {
		return errors.New("StateDB.Insert: Mutable <nil> pointer")
	}
	return nil
}

// Insert a state into StateDB. Storing the object is done using encoding/gob,
// requiring all fields (that are meant to be stored) to be exported (First letter in Upper-Case).
// Additionally:
// - state must be a struct, and MUST have a field named 'ID' of type int or string
// - state must not already exist, an error is produced otherwise
// - state is encoded using encoding/gob and therefore must adhere to all the rules required by it (but gets all the benefits as well).
// func (db *StateDB) InsertImmutabe(state interface{}) (*KeyType, error) {
// func (db *StateDB) Insert(i interface{}) (*KeyType, error) {

// 	if i == nil {
// 		return nil, errors.New("StateDB: Cannot insert <nil>")
// 	}

// 	state := Indirect(reflect.ValueOf(i))

// 	// get the ID of the object and create a keytype
// 	kt, err := reflectKeyType(state)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return kt, db.InsertWithKeyType(kt, i)
// }

// By inserting directly with a KeyType valid keytype, the object can be of any type, not just a struct. This makes sense for static objects such as lists, maps etc.
// - the ID in the Key can be either string or int, but MUST be defined for the replay to work.
//   Therefore, make sure that the key is valid
// - the Type must be unique across the database, such that it does not collide with any existing type in the database. A nice design of the type-ids would be a '<packagename>.<type>' combination.
//
// To restore, simply call db.RestoreIter(kt) with the same typeid, and an iterator will be generated.
// func (db *StateDB) InsertWithKeyType(kt *KeyType, i interface{}) error {

// 	if i == nil {
// 		return errors.New("StateDB: Cannot insert <nil> state")
// 	}

// 	if kt == nil || !kt.IsValid() {
// 		return fmt.Errorf("StateDB.InsertWithKeyType: Invalid KeyType = %s", kt.String())
// 	}

// 	if db.ContainsKeyType(kt) {
// 		return errors.New("StateDB: KeyType " + kt.String() + " already exists")
// 	}

// 	return db.insertWithKeyType(kt, Indirect(reflect.ValueOf(i)))
// }

// Could be made public to allow people to cache the key instead
// of having the api use reflection to inspect it.
// func (db *StateDB) insertWithKeyType(kt *KeyType, state reflect.Value) error {

// 	if state.Kind() == reflect.Ptr && state.IsNil() {
// 		return errors.New("StateDB: Cannot encode nil pointer of type " + state.Type().String())
// 	}

// 	var buff bytes.Buffer
// 	enc := gob.NewEncoder(&buff)

// 	if err := enc.EncodeValue(state); err != nil {
// 		return err
// 	}
// 	val := buff.Bytes()

// 	// write lock for the remaining operations
// 	db.Lock()
// 	defer db.Unlock()

// 	// insert into immutable table
// 	db.insertImmutable(kt, val)
// 	// log creation in delta
// 	db.insertDelta(kt, val)

// 	return nil
// }

// func (db *StateDB) RegisterMutable(kt *KeyType, i interface{}) error {

// 	if i == nil {
// 		return errors.New("StateDB.RegisterMutable: <nil> state")
// 	}

// 	if kt == nil || !kt.IsValid() {
// 		return fmt.Errorf("StateDB.RegisterMutable: Invalid KeyType = %s", kt.String())
// 	}

// 	iv := reflect.ValueOf(i)

// 	if iv.Kind() != reflect.Ptr {
// 		return errors.New("StateDB.RegisterMutable: An update can only be performed using a pointer to the mutable state")
// 	}

// 	if iv.Kind() == reflect.Ptr && iv.IsNil() {
// 		return errors.New("StateDB.RegisterMutable: Cannot call update with <nil> pointer")
// 	}

// 	// TODO: put into safe RLock & RUnlock Contains method so we can use defer to clean it up
// 	if !db.ContainsKeyType(kt) {
// 		return errors.New("StateDB: KeyType " + kt.String() + " does not exist")
// 	}

// 	db.Lock()
// 	defer db.Unlock()

// 	return db.registerMutable(kt, iv)
// }
