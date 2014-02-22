package statedb

import (
	"fmt"
	"os"
	// "reflect"
	"runtime"
	"testing"
	// "time"
	"sync"
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

func (w *Weird) Mutable() interface{} {

	return w.m

}

var main []*Main

var weird []*Weird

func init() {

	runtime.GOMAXPROCS(4)

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
		t.Fatal(err)
	}

	var ws []Weird
	if it, err := db.RestoreIter(ReflectType(Weird{})); err == nil {
		for {
			weird := new(Weird)
			_, ok := it.Next(weird)
			if !ok {
				break
			}

			fmt.Println(weird)
			ws = append(ws, *weird)
		}
	} else {
		t.Fatal(err)
	}

	if err = db.Checkpoint(); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("restored: %#v\n", ws)
}

func WriteFullAndDelta(path string, t *testing.T) {
	db, err := NewStateDB("", path, "")
	if err != nil {
		t.Fatal(err)
	}

	if db.restored {
		return
	}

	t_str := ReflectType(Weird{})
	resp := make(chan *KeyType)
	n := 0
	var wg sync.WaitGroup

	for _, m := range weird {
		wg.Add(1)
		n++
		go func(m *Weird, wg *sync.WaitGroup) {
			kt, err := db.Insert(m)
			if err != nil {
				wg.Done()
				resp <- kt
				t.Fatal(err)
			}
			m.m.K = kt.StringID() + "test"
			wg.Done()
			resp <- kt
		}(m, &wg)
	}
	wg.Wait()
	for _, _ = range weird {
		// for i := 0; i < n; i++ {
		kt := <-resp
		if kt == nil {
			continue
		}
		if kt.StringID() == "3" {
			if err := db.Checkpoint(); err != nil {
				t.Fatal(err)
			}
			for i, w := range weird {
				w.m.I = i
			}
		} else if kt.StringID() == "4" {
			if err := db.Remove(kt); err != nil {
				t.Fatal(err)
			}
		} else if kt.StringID() == "5" {
			kt, err := NewStringKeyType("2", t_str)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Remove(kt); err != nil {
				t.Fatal(err)
			}

			if err := db.Checkpoint(); err != nil {
				t.Fatal(err)
			}
		}
	}

	kt, _ := NewStringKeyType("4", t_str)
	if db.immutable.contains(kt) {
		t.Error("kt " + kt.String() + " was not deleted.")
	}
	kt, _ = NewStringKeyType("2", t_str)
	if db.immutable.contains(kt) {
		t.Error("kt " + kt.String() + " was not deleted.")
	}
	// is not normally possible
	db.incrementalCheckpoint()
	db.incrementalCheckpoint()
	db.incrementalCheckpoint()
	db.Commit()
}

func TestCheckpoint(t *testing.T) {
	path := "checkpoint_test"
	defer CleanUp(path)

	WriteFullAndDelta(path, t)

	RestoreCheckpoint(path, t)
}
