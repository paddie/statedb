package statedb

import (
	// "errors"
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
