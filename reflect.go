package statedb

import (
	"errors"
	"fmt"
	"reflect"
)

type Mutable interface {
	Mutable() interface{}
}

type KeyStr interface {
	Key() string
}

type KeyInt interface {
	Key() int
}

type Typer interface {
	Type() string
}

func ReflectKeyTypeM(v interface{}) (*KeyType, error) {

	if v == nil {
		return nil, errors.New("Keytype: <nil> does not have keytype")
	}

	typ := ReflectTypeM(v)
	key, err := ReflectKeyM(v)
	if err != nil {
		return nil, err
	}

	return &KeyType{*key, typ}, nil
}

// If no Type method is defined, use reflect to determine type
func ReflectTypeM(v interface{}) string {
	// call Type if object supports it
	if m, ok := v.(Typer); ok {
		return m.Type()

	}
	// or reflect the type instead
	return ReflectType(v)
}

func ReflectKeyM(v interface{}) (*Key, error) {
	if m, ok := v.(KeyInt); ok {
		return NewIntKey(m.Key())
	}

	if m, ok := v.(KeyStr); ok {
		return NewStringKey(m.Key())
	}

	k, err := ReflectKey(v)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// Return the unique type if for any object
func ReflectType(i interface{}) string {
	if i == nil {
		return ""
	}
	return reflectType(reflect.ValueOf(i))
}

// Return the unique type if for any object
func reflectType(val reflect.Value) string {
	return Indirect(val).Type().String()
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
