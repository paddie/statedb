package avgbuffer

// The AVGFloat64 is a circular buffer that will never grow
// bigger then the provided size.
//
// Every time Upsert(val) is called on the buffer, we do one of two things depending on the buffer being full or not:
// 1. Full: subtract the oldest value in the buffer from the sum,
//    overwrite the oldest value, update the sum and average
// 2. Not full: Insert the value, update the sum and the average
import (
	"fmt"
)

type AVGFloat64 struct {
	buff []float64
	i    int
	sum  float64
	avg  float64
}

func NewAVGFloat64(size int) *AVGFloat64 {
	return &AVGFloat64{
		buff: make([]float64, 0, size),
	}
}

func (b *AVGFloat64) String() string {
	return fmt.Sprintf("avg=%.2f buff=%v (i=%d, size=%d)", b.avg, b.buff, b.i, len(b.buff))
}

func (b *AVGFloat64) Upsert(val float64) float64 {

	if len(b.buff) < cap(b.buff) {
		// append until we reach 10
		b.buff = append(b.buff, val)
	} else {
		// subtract oldest value
		b.sum -= b.buff[b.i]
		// replace with new value
		b.buff[b.i] = val
	}
	// update sum
	b.sum += val
	// update average
	b.avg = b.sum / float64(len(b.buff))

	// update pointer
	// - use capasity for pointer update
	if b.i == cap(b.buff)-1 {
		b.i = 0
	} else {
		b.i += 1
	}

	return b.avg
}

func (b *AVGFloat64) Get(i int) (float64, error) {
	if i < 0 || i > len(b.buff)-1 {
		return -1, fmt.Errorf("buff: [0;%d] invalid i=%d", len(b.buff)-1, i)
	}

	return b.buff[i], nil
}

func (b *AVGFloat64) MostRecent() float64 {
	if len(b.buff) == 0 {
		return -1
	}

	if b.i == 0 {
		return b.buff[len(b.buff)-1]
	}

	return b.buff[b.i-1]
}

func (b *AVGFloat64) AVG() float64 {
	return b.avg
}

func (b *AVGFloat64) Cap() int {
	return cap(b.buff)
}

func (b *AVGFloat64) Size() int {
	return len(b.buff)
}
