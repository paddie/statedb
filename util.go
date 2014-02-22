package statedb

import (
	"errors"
	// "fmt"
	"os"
	"reflect"
)

// Create all directories in the provided path
func CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// Dereferences a pointer until we get to the final value
func Indirect(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Ptr {
		return Indirect(value.Elem())
	}

	return value
}

// Only returns true if the path is of a file, that is not a dir, and that exists
func IsFile(path string) bool {
	stat, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		return false
	}

	if stat.IsDir() {
		return false
	}

	return true
}

// Only returns true if the path is of a directory that already exists
func IsDir(path string) bool {

	stat, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		return false
	}

	if !stat.IsDir() {
		return false
	}

	return true
}

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
	val := reflect.ValueOf(i)
	return reflectType(val)
}

// Return the unique type if for any object
func reflectType(val reflect.Value) string {
	return Indirect(val).Type().String()
}
