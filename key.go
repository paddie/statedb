package statedb

import (
	"errors"
	"fmt"
	// "reflect"
)

type Key struct {
	IntID    int
	StringID string
}

func (k *Key) String() string {
	if k.IntID == 0 {
		return k.StringID
	}

	return fmt.Sprintf("%d", k.IntID)
}

func NewStringKey(id string) (*Key, error) {
	if len(id) == 0 {
		return nil, errors.New("Invalid Key: ''")
	}

	return &Key{0, id}, nil
}

func NewIntKey(id int) (*Key, error) {
	if id <= 0 {
		return nil, fmt.Errorf("Invalid Key: %d <= 0", id)
	}

	return &Key{id, ""}, nil
}

func (k *Key) IsValid() bool {
	if k.StringID == "" && k.IntID == 0 {
		return false
	}

	return true
}
