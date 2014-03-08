package statedb

import (
	"errors"
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

// func reflectKeyType(val reflect.Value) (*KeyType, error) {
// 	key, err := reflectKey(Indirect(val))
// 	if err != nil {
// 		return nil, err
// 	}

// 	typeStr := reflectType(val)
// 	if typeStr == "" {
// 		return nil, fmt.Errorf("No accessible keytype for %v", val)
// 	}

// 	return &KeyType{
// 		K: *key,
// 		T: typeStr}, nil
// }

// Return the unique type if for any object
func reflectType(val reflect.Value) string {
	return Indirect(val).Type().String()
}
