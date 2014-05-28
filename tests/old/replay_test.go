package statedb

// import (
// 	"bytes"
// 	"encoding/gob"
// 	"os"
// 	"reflect"
// 	"testing"
// )

// func TestReplay(t *testing.T) {

// 	dir := "replay_test"

// 	db, err := NewStateDB("", dir, "")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	type Restore struct {
// 		ID  string
// 		Val int
// 	}

// 	r := Restore{"restored", 9}

// 	kt, err := getKeyType(reflect.ValueOf(r))
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	var buff bytes.Buffer
// 	enc := gob.NewEncoder(&buff)

// 	enc.Encode(r)

// 	db.delta = nil

// 	if db.ContainsKeyType(kt) {
// 		t.Error("Invalid starting position")
// 	}

// 	db.insertDelta(kt, buff.Bytes())

// 	if db.delta[kt.T][kt.K].Action != CREATE {
// 		t.Errorf("CREATE entry for %s not in Delta", kt.String())
// 	}

// 	deltas := []DeltaTypeMap{db.delta}

// 	err = db.replayDeltas(deltas)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if db.delta != nil {
// 		t.Error("Did not nil Delta after replay")
// 	}

// 	if !db.ContainsKeyType(kt) {
// 		t.Error("Replay failed!")
// 	}

// 	res := new(Restore)
// 	err = db.DecodeFromKeyType(kt, res)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if res.Val != 9 {
// 		t.Errorf("Corrupt restore: 9 != %d", res.Val)
// 	}
// }
