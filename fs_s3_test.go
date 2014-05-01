package statedb

import (
	"fmt"
	"launchpad.net/goamz/aws"
	// "launchpad.net/goamz/s3"
	"testing"
)

func TestS3Put(t *testing.T) {

	auth, err := aws.EnvAuth()
	if err != nil {
		t.Fatal(err)
	}

	fs, err := NewFS_S3(auth, aws.EUWest, "test", "statedbs3")
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
