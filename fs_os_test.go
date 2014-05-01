package statedb

import (
	"fmt"
	"testing"
)

func TestList(t *testing.T) {

	dir := "fs_test"

	fs, err := NewFS_OS(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.Init()
	if err != nil {
		t.Fatal(err)
	}

	files, err := fs.List("mut_*")
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Fatal("Not all files were listed")
	}

	for _, l := range files {
		fmt.Println(l)
	}
}

func TestPut(t *testing.T) {

	dir := "fs_os_test"

	fs, err := NewFS_OS(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.Init()
	if err != nil {
		t.Fatal(err)
	}

	str := "immaculate test!"

	fmt.Println("Writing: ", str)
	err = fs.Put("put.tst", []byte(str))
	if err != nil {
		t.Fatal(err)
	}

	data, err := fs.Get("put.tst")
	if err != nil {
		t.Fatal(err)
	}

	if str != string(data) {
		t.Fatalf("PUT != GET data: '%s'", string(data))
	}

	err = fs.Delete("put.tst")
	if err != nil {
		t.Fatal(err)
	}
}
