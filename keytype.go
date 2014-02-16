package statedb

import (
	"errors"
	"fmt"
	"reflect"
)

// The key can be of type string or int64
type KeyType struct {
	K Key    // Key of the object
	T string // Type of the object
	// mut      *State // link to the dynamic part
	// Mutable  bool // does the state contain mutable data
	verified bool // for NewKeyType objects, this is false. It is only true if the objet was returned by the insert or after an update.
	// db       *StateDB
}

func CustomKeyType(key *Key, typeStr string) (*KeyType, error) {

	if typeStr == "" {
		return nil, errors.New("KeyType: typeStr = ''")
	}

	if key == nil || !key.IsValid() {
		return nil, fmt.Errorf("KeyType: %#v is not a valid key", key)
	}

	return &KeyType{
		K: *key,
		T: typeStr}, nil
}

func NewKeyType(key *Key, i interface{}) (*KeyType, error) {

	if i == nil {
		return nil, errors.New("Cannot generate KeyType from <nil>")
	}

	if key == nil || !key.IsValid() {
		return nil, fmt.Errorf("KeyType: key %#v is not a valid key", key)
	}

	typeStr := reflectType(Indirect(reflect.ValueOf(i)))

	return &KeyType{
		K: *key,
		T: typeStr}, nil
}

func ReflectKeyType(i interface{}) (*KeyType, error) {

	if i == nil {
		return nil, errors.New("Cannot generate KeyType from <nil>")
	}

	val := Indirect(reflect.ValueOf(i))

	return reflectKeyType(val)
}

func reflectKeyType(val reflect.Value) (*KeyType, error) {
	key, err := reflectKey(val)
	if err != nil {
		return nil, err
	}

	typeStr := reflectType(val)
	if typeStr == "" {
		return nil, fmt.Errorf("No accessible keytype for %v", val)
	}

	return &KeyType{
		K: *key,
		T: typeStr}, nil
}

func (k *KeyType) IsValid() bool {
	if !k.K.IsValid() || k.TypeID() == "" {
		return false
	}
	return true
}

// Generate a integer key
func NewIntKeyType(intID int64, typeStr string) (*KeyType, error) {

	if typeStr == "" {
		return nil, fmt.Errorf("Key: cannot contain an empty type")
	}

	key, err := NewIntKey(intID)
	if err != nil {
		return nil, err
	}

	return &KeyType{
		K: *key,
		T: typeStr}, nil
}

// Generate a string key
func NewStringKeyType(stringID string, typeStr string) (*KeyType, error) {

	if typeStr == "" {
		return nil, fmt.Errorf("Key: cannot contain an empty type")
	}

	key, err := NewStringKey(stringID)
	if err != nil {
		return nil, err
	}

	return &KeyType{
		K: *key,
		T: typeStr}, nil
}

// Return the value that is used as the key-value
func (k *KeyType) String() string {
	return fmt.Sprintf("<Type:'%s' Key:'%s'>", k.TypeID(), k.K.String())
}

// return the string part of the key
func (k *KeyType) StringID() string {
	return k.K.StringID
}

// return the integer part of the key
func (k *KeyType) IntID() int64 {
	return k.K.IntID
}

func (k *KeyType) TypeID() string {
	return k.T
}
