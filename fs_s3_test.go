package statedb

import (
	"fmt"
	"launchpad.net/goamz/aws"
	// "launchpad.net/goamz/s3"
	"testing"
)

func TestFS_S3_Put_Get(t *testing.T) {

	auth, err := aws.EnvAuth()
	if err != nil {
		t.Fatal(err)
	}

	fs, err := NewFS_S3(auth, aws.EUWest, "test", "statedbs3")
	if err != nil {
		t.Fatal(err)
	}

	if err = fs.Init(); err != nil {
		fmt.Println(err.Error())
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

	if _, err = fs.Get("put.tst"); err == nil {
		t.Fatal("key should not exist!")
	}
}
