package statedb

import (
	"fmt"
	"os"
	// "reflect"
	"testing"
)

type Main struct {
	ID  int
	Tmp int
}

type w_mut struct {
	I int
	K string
}

type Weird struct {
	ID string
	S  int
	m  w_mut
}

var main []*Main

var weird []*Weird

func init() {

	main = []*Main{
		{ID: 1, Tmp: 1},
		{ID: 2, Tmp: 2},
		{ID: 3, Tmp: 3},
		{ID: 4, Tmp: 4},
		{ID: 5, Tmp: 5},
		{ID: 6, Tmp: 6},
	}

	weird = []*Weird{
		{ID: "1", S: 1},
		{ID: "2", S: 2},
		{ID: "3", S: 3},
		{ID: "4", S: 4},
		{ID: "5", S: 5},
	}
}

func CleanUp(path string) error {
	return os.RemoveAll(path)
}

func RestoreCheckpoint(path string, t *testing.T) {

	fmt.Println("restoring from " + path)

	db, err := NewStateDB("", path, "")
	if err != nil {
		t.Error(err)
	}

	var ms []Main
	if it, err := db.RestoreImmIter(ReflectType(Main{})); err == nil {
		for {
			main := new(Main)
			_, ok := it.Next(main)
			if !ok {
				break
			}
			ms = append(ms, *main)
		}
	} else {
		t.Error(err)
	}
	fmt.Printf("restored: %#v\n", ms)

	var ws []Weird
	if it, err := db.RestoreImmIter(ReflectType(Weird{})); err == nil {
		for {
			weird := new(Weird)
			kt, ok := it.Next(weird)
			if !ok {
				break
			}
			if err := db.RestoreMutable(kt, &weird.m); err != nil {
				t.Error(err)
			}

			ws = append(ws, *weird)
		}
	} else {
		t.Error(err)
	}
	fmt.Printf("restored: %#v\n", ws)
}

func WriteFullAndDelta(path string, t *testing.T) {
	db, err := NewStateDB("", path, "")
	if err != nil {
		t.Error(err)
	}
	for _, m := range main {

		kt, err := db.Insert(*m)
		if err != nil {
			t.Error(err)
		}

		if m.ID == 2 {
			if err := db.FullCheckpoint(); err != nil {
				t.Error(err)
			}
		} else if m.ID == 4 || m.ID == 6 {
			if err := db.DeltaCheckpoint(); err != nil {
				t.Error(err)
			}
		} else if m.ID == 5 {
			if err := db.Remove(kt); err != nil {
				t.Error(err)
			}
		}
	}
	for _, m := range weird {
		kt, err := db.Insert(m)
		if err != nil {
			t.Error(err)
		}

		if err := db.RegisterMutable(kt, &m.m); err != nil {
			t.Error(err)
		}

		m.m.I = 5
		m.m.K = kt.StringID()

		if m.S == 3 {
			if err := db.FullCheckpoint(); err != nil {
				t.Error(err)
			}
		} else if m.S == 4 {
			if err := db.Remove(kt); err != nil {
				t.Error(err)
			}
		} else if m.S == 5 {
			str := ReflectType(m)
			kt, _ := NewStringKeyType("2", str)
			db.Remove(kt)

			if err := db.DeltaCheckpoint(); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestCheckpoint(t *testing.T) {
	path := "checkpoint_test"
	defer CleanUp(path)

	WriteFullAndDelta(path, t)

	RestoreCheckpoint(path, t)
}
