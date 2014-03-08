package statedb

import (
	"errors"
	"fmt"
)

// The key can be of type string or int64
type KeyType struct {
	K Key    // Key of the object
	T string // Type of the object
}

func (kt *KeyType) Key() *Key {
	return &kt.K
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

func NewKeytype(key *Key, typ string) (*KeyType, error) {

	if key == nil || !key.IsValid() {
		return nil, errors.New("Key: Invalid key " + key.String())
	}

	return &KeyType{
		K: *key,
		T: typ,
	}, nil
}

func (k *KeyType) IsValid() bool {
	if !k.K.IsValid() || k.TypeID() == "" {
		return false
	}
	return true
}

// Generate a integer key
func NewIntKeyType(intID int, typeStr string) (*KeyType, error) {

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
func (k *KeyType) IntID() int {
	return k.K.IntID
}

func (k *KeyType) TypeID() string {
	return k.T
}
