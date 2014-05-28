package statedbtests

import (
	"fmt"
	"github.com/paddie/statedb"
	"github.com/paddie/statedb/fs"
	"github.com/paddie/statedb/monitor"
	"github.com/paddie/statedb/schedular"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

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
		{ID: "6", S: 6},
		{ID: "7", S: 7},
	}
}

func CleanUp(path string) error {
	return os.RemoveAll(path)
}

func RestoreCheckpoint(t *testing.T) {
	mdl := schedular.NewAlways()

	// s :=  monitor.NewEC2Instance(s, instanceType, productDescription, availabilityZone, filter)
	mon := monitor.NewTestMonitor(time.Second * 5)

	f, _ := fs.NewFS_OS("cpt_test")

	err := f.Init()
	if err != nil {
		t.Error(err)
	}

	db, restored, err := statedb.NewStateDB(f, mdl, mon, 1.0, "")
	if err != nil {
		t.Fatal(err)
		return
	}

	if restored == false {
		t.Fatal("StateDB: did not signal a restore")
	}

	var ws []Weird
	typ := statedb.ReflectTypeM(&Weird{})
	if it, err := db.RestoreIter(typ); err == nil {
		for {
			weird := new(Weird)
			_, ok := it.Next(weird)
			if !ok {
				// break
			}

			// fmt.Println(weird)
			ws = append(ws, *weird)
		}
	} else {
		t.Fatal(err)
	}

	if len(ws) != len(main)-1 {
		t.Fatalf("Length of restored %d != %d length of committed", len(ws), len(main))
	}

	fmt.Printf("restored: %#v\n", ws)
}

// // func RestorePartialState(path string, t *testing.T) {

// // 	fmt.Println("restoring from " + path)

// // 	db, err := NewStateDB("", path, "")
// // 	if err != nil {
// // 		t.Fatal(err)
// // 	}

// // 	var ws []Wurd
// // 	typ := ReflectTypeM(Wurd{})
// // 	if it, err := db.RestoreIter(typ); err == nil {
// // 		for {
// // 			weird := new(Wurd)
// // 			_, ok := it.Next(weird)
// // 			if !ok {
// // 				break
// // 			}

// // 			fmt.Println(weird)
// // 			ws = append(ws, *weird)
// // 		}
// // 	} else {
// // 		t.Fatal(err)
// // 	}

// // 	if err = db.Checkpoint(); err != nil {
// // 		t.Fatal(err)
// // 	}

// // 	// if

// // 	fmt.Printf("restored: %#v\n", ws)
// // }

func WriteFullAndDelta(t *testing.T) {

	mdl := schedular.NewRisingEdge()

	// s :=  monitor.NewEC2Instance(s, instanceType, productDescription, availabilityZone, filter)
	mon := monitor.NewTestMonitor(time.Second * 5)

	f, _ := fs.NewFS_OS("cpt_test")

	err := f.Init()
	if err != nil {
		t.Error(err)
	}

	db, restored, err := statedb.NewStateDB(f, mdl, mon, 1.0, "")
	if err != nil {
		t.Fatal(err)
		return
	}

	if restored == true {
		t.Fatal("StateDB: should not have restored at this time")
	}

	t_str := statedb.ReflectTypeM(Weird{})

	resp := make(chan *statedb.KeyType)
	n := 0
	var wg sync.WaitGroup

	for i, m := range weird {
		if i == 4 || i == 2 {
			wg.Add(1)
		}

		n++
		go func(m *Weird, wg *sync.WaitGroup, i int) {
			kt, err := db.Insert(m)
			// fmt.Println(kt)
			if err != nil {
				if i == 4 || i == 2 {
					wg.Done()
				}
				resp <- kt
				t.Fatal(err)
			}

			m.m.K = kt.StringID() + "test"
			wg.Done()
			resp <- kt
		}(m, &wg, i)
	}
	wg.Wait()
	db.Sync()
	for i, _ := range weird {
		// for i := 0; i < n; i++ {
		kt := <-resp
		if kt == nil {
			continue
		}
		if kt.StringID() == "4" || kt.StringID() == "2" {
			if err := db.Remove(kt); err != nil {
				t.Fatal(err)
			}

			if err := db.Sync(); err != nil {
				t.Fatal(err)
			}
		}

		// after 4 iterations, update the mutable bits
		if i == 3 {
			for i, w := range weird {
				w.m.I = i
			}
		}
		if i%2 == 0 {
			if err := db.Sync(); err != nil {
				t.Fatal(err)
			}
		}
	}
	kt, _ := statedb.NewStringKeyType("4", t_str)
	if db.immutable.contains(kt) {
		t.Error("kt " + kt.String() + " was not deleted.")
	}
	kt, _ = NewStringKeyType("2", t_str)
	if db.immutable.contains(kt) {
		t.Error("kt " + kt.String() + " was not deleted.")
	}

	err = db.Commit()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCheckpoint(t *testing.T) {

	t.Skip()

	path := "cpt_test"

	CleanUp(path)

	WriteFullAndDelta(t)

	// time.Sleep(time.Second * 1)
	RestoreCheckpoint(t)
}

type Main struct {
	ID  int
	Tmp int
}

type w_mut struct {
	I int
	K string
}

type Wurd struct {
	ID string
	w  w_mit
}

type w_mit struct {
	I int
}

func (w *Wurd) Mutable() interface{} {
	return &w.w
}

type Weird struct {
	ID string
	S  int
	m  w_mut
}

func (w *Weird) Type() string {
	return "WeirdMan"
}

func (w *Weird) Mutable() interface{} {
	return &w.m
}
