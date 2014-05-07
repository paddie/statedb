package avgbuffer

// import (
// 	"fmt"
// 	"testing"
// )

// var buff *AVGBuffer

// func init() {
// 	buff = NewAVGBuffer(10)
// }

// func TestAVG(t *testing.T) {

// 	var v float64 = 100.0
// 	var exp float64 = 100.0

// 	avg := buff.Upsert(v)
// 	fmt.Println(buff)
// 	if avg != exp {
// 		t.Error("Incorrect average: expected %f != actual %f", exp, avg)
// 	}

// 	// overwrite all values in buffer with a new value,
// 	// changing the average to the new value
// 	v = 102.0
// 	for i := 0; i < 10; i++ {
// 		_ = buff.Upsert(v)
// 	}
// 	fmt.Println(buff)
// 	exp = float64(v)
// 	avg = buff.AVG()

// 	if avg != exp {
// 		t.Error("Incorrect average: expected %f != actual %f", exp, avg)
// 	}
// }

// func TestMostRecent(t *testing.T) {
// 	var dExp int64 = 103
// 	buff.Upsert(dExp)

// 	if recent := buff.MostRecent(); recent != dExp {
// 		t.Error("Incorrect MostRecent: expected %f != actual %f", dExp, recent)
// 	}
// }
