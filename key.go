package statedb

import (
	"errors"
	"fmt"
	"reflect"
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
		return nil, errors.New("Key: id = ''")
	}

	return &Key{0, id}, nil
}

func NewIntKey(id int) (*Key, error) {
	if id <= 0 {
		return nil, fmt.Errorf("Key: id <= 0: %d", id)
	}

	return &Key{id, ""}, nil
}

func (k *Key) IsValid() bool {
	if k.StringID == "" && k.IntID == 0 {
		return false
	}

	return true
}

// Use reflect to generate the key from the "ID" field in a struct
func ReflectKey(i interface{}) (*Key, error) {
	if i == nil {
		return nil, errors.New("Key: <nil> has no key")
	}

	val := Indirect(reflect.ValueOf(i))
	return reflectKey(val)
}

// Search for a field named 'ID' in the provided struct
// - The 'ID' field must be of type Int | String, any other type
//   results in an error
func reflectKey(val reflect.Value) (*Key, error) {

	if val.Kind() == reflect.Ptr && val.IsNil() {
		return nil, fmt.Errorf("Key: <nil> pointer of type %s", val.String())
	}

	if val.Kind() == reflect.Ptr {
		val = Indirect(val)
	}

	// TODO: for static-only states (meaning no mutability)
	//       this might be too limiting.
	if val.Kind() != reflect.Struct {
		return nil, errors.New("Key: only structs can have a Key")
	}

	id_field := val.FieldByName("ID")

	if !id_field.IsValid() {
		return nil, fmt.Errorf("%s does not have field with name = 'ID'", val.Type().String())
	}

	switch id_field.Kind() {
	case reflect.String:
		return NewStringKey(id_field.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return NewIntKey(int(id_field.Int()))
	default:
		return nil, fmt.Errorf("Field 'ID' is of invalid type: %v\n", id_field.Kind())
	}
}
