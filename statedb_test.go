package statedb

import (
	// "bytes"
	// "encoding/gob"
	"fmt"
	"os"
	// "reflect"
	"testing"
)

// var db *StateDB

func init() {
	// ctx, err := NewContext("test_root", "main", "statedb")

}

func TestInsertAndDelete(t *testing.T) {

	t.Skip("Skipping test for now..")

	// var err error
	db, err := NewStateDB("", "statedb_test", "")
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll("statedb_test")

	type Main struct {
		ID  string
		tmp int
	}

	m := Main{"tmp", 2}

	type R struct {
		ID int
		V  int   `json:"lols" cpt:"tag1"`
		D  *Main // dynamic part of the state
	}

	m = Main{"Looove", 0}
	r1 := R{2, 1, &m}
	// r2 := R{3, 2, &m}

	kt, _ := ReflectKeyType(r1)

	fmt.Println("Inserting", kt.String())
	k1, err := db.Insert(&r1)
	if err != nil {
		fmt.Println(err)
	}

	if db.delta[k1.T][k1.K].Action != CREATE {
		t.Errorf("CREATE entry for %s not in Delta", kt.String())
	}

	fmt.Println("Deleting " + kt.String())
	db.Remove(kt)

	if db.immutable.contains(kt) {
		t.Errorf("%s is not deleted", kt.String())
	}

	_, _ = db.Insert(&r1)

	db.delta = nil

	db.Remove(kt)

	if db.delta[k1.T][k1.K].Action != DELETE {
		t.Errorf("DELETE entry for %s not in Delta", kt.String())
	}
}
