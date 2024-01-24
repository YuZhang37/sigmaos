package intervals_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"reflect"
	"testing"
	"time"

	db "sigmaos/debug"
	"sigmaos/interval"
	"sigmaos/intervals"
	"sigmaos/skipintervals"
	"sigmaos/sliceintervals"
)

func testSimple(t *testing.T, ivs interval.IIntervals) {
	ivs.Insert(interval.NewInterval(1, 2))
	ivs.Insert(interval.NewInterval(2, 3))
	ivs.Delete(interval.NewInterval(1, 2))
	assert.Equal(t, 1, ivs.Length())
}

func TestCompile(t *testing.T) {
}

func TestSimple(t *testing.T) {
	testSimple(t, sliceintervals.NewIInterval())
	testSimple(t, skipintervals.NewSkipIInterval())
}

func testContains(t *testing.T, ivs interval.IIntervals) {
	ivs.Insert(interval.NewInterval(0, 10))
	ivs.Insert(interval.NewInterval(90, 100))
	assert.True(t, ivs.Contains(0))
	assert.False(t, ivs.Contains(10))
	assert.False(t, ivs.Contains(11))
	assert.True(t, ivs.Contains(90))
	assert.False(t, ivs.Contains(100))
	assert.False(t, ivs.Contains(200))
}

func TestContains(t *testing.T) {
	testContains(t, sliceintervals.NewIInterval())
	testContains(t, skipintervals.NewSkipIInterval())
}

func testInsert(t *testing.T, ivs interval.IIntervals) {
	ivs.Insert(interval.NewInterval(0, 10))
	ivs.Insert(interval.NewInterval(10, 20))
	assert.Equal(t, 1, ivs.Length())
	ivs.Insert(interval.NewInterval(15, 20))
	assert.Equal(t, 1, ivs.Length())
	ivs.Insert(interval.NewInterval(30, 40))
	assert.Equal(t, 2, ivs.Length())
	ivs.Insert(interval.NewInterval(20, 25))
	assert.Equal(t, 2, ivs.Length())
	ivs.Insert(interval.NewInterval(50, 60))
	assert.Equal(t, 3, ivs.Length())
	ivs.Insert(interval.NewInterval(70, 80))
	assert.Equal(t, 4, ivs.Length())
	ivs.Insert(interval.NewInterval(40, 50))
	assert.Equal(t, 3, ivs.Length())
	ivs.Insert(interval.NewInterval(25, 30))
	assert.Equal(t, 2, ivs.Length())
	ivs.Insert(interval.NewInterval(60, 70))
	assert.Equal(t, 1, ivs.Length())
}

func TestInsert(t *testing.T) {
	testInsert(t, sliceintervals.NewIInterval())
	testInsert(t, skipintervals.NewSkipIInterval())
}

func testDelete(t *testing.T, ivs interval.IIntervals) {
	ivs.Insert(interval.NewInterval(0, 100))
	db.DPrintf(db.TEST, "ivs %v\n", ivs)
	ivs.Delete(interval.NewInterval(5, 10))
	assert.Equal(t, 2, ivs.Length())
	db.DPrintf(db.TEST, "ivs %v\n", ivs)
	ivs.Delete(interval.NewInterval(30, 50))
	db.DPrintf(db.TEST, "ivs %v\n", ivs)
	assert.Equal(t, 3, ivs.Length())
	ivs.Delete(interval.NewInterval(50, 100))
	db.DPrintf(db.TEST, "ivs %v\n", ivs)
	assert.Equal(t, 2, ivs.Length())
	ivs.Delete(interval.NewInterval(20, 30))
	assert.Equal(t, 2, ivs.Length())
	db.DPrintf(db.TEST, "ivs %v\n", ivs)
	ivs.Delete(interval.NewInterval(0, 5))
	db.DPrintf(db.TEST, "ivs %v\n", ivs)
	assert.Equal(t, 1, ivs.Length())
	ivs.Delete(interval.NewInterval(10, 20))
	assert.Equal(t, 0, ivs.Length())

	ivs.Insert(interval.NewInterval(0, 100))
	db.DPrintf(db.TEST, "ivs %v\n", ivs)
	ivs.Delete(interval.NewInterval(5, 10))
	assert.Equal(t, 2, ivs.Length())
	db.DPrintf(db.TEST, "ivs %v\n", ivs)
	ivs.Delete(interval.NewInterval(0, 100))
	db.DPrintf(db.TEST, "ivs %v\n", ivs)
	assert.Equal(t, 0, ivs.Length())
}

func TestDelete(t *testing.T) {
	testDelete(t, sliceintervals.NewIInterval())
	testDelete(t, skipintervals.NewSkipIInterval())
}

// No overlapping intervals
func testBasic(t *testing.T, siv interval.IIntervals) {
	ivs := []*interval.Tinterval{interval.NewInterval(2, 4), interval.NewInterval(10, 12),
		interval.NewInterval(5, 7), interval.NewInterval(0, 1), interval.NewInterval(20, 22)}
	e := siv.Find(interval.NewInterval(10, 12))
	assert.Nil(t, e)
	for _, iv := range ivs {
		siv.Insert(iv)
	}
	for _, iv := range ivs {
		assert.True(t, siv.Present(iv))
	}
	e = siv.Find(ivs[1])
	assert.NotNil(t, e)

	siv.Delete(ivs[3])
	siv.Delete(ivs[2])
	siv.Delete(ivs[1])
	siv.Delete(ivs[0])
	siv.Delete(ivs[4])
	for _, iv := range ivs {
		assert.False(t, siv.Present(iv))
	}
	assert.True(t, siv.Length() == 0)
}

func TestBasic(t *testing.T) {
	testBasic(t, sliceintervals.NewIInterval())
	testBasic(t, skipintervals.NewSkipIntervals())
}

func testInsert1(t *testing.T, siv interval.IIntervals) {
	ivs := []*interval.Tinterval{
		interval.NewInterval(0, 10),
		interval.NewInterval(10, 20),
		interval.NewInterval(15, 20),
		interval.NewInterval(30, 40),
		interval.NewInterval(20, 25),
		interval.NewInterval(50, 60),
		interval.NewInterval(70, 80),
		interval.NewInterval(40, 50),
		interval.NewInterval(25, 30),
		interval.NewInterval(60, 70),
	}
	lens := []int{1, 1, 1, 2, 2, 3, 4, 3, 2, 1}
	for i, iv := range ivs {
		siv.Insert(iv)
		assert.Equal(t, lens[i], siv.Length(), i)
		assert.True(t, siv.Present(iv))
	}
}

func TestInsert1(t *testing.T) {
	// testInsert1(t, sliceintervals.NewIInterval())
	testInsert1(t, skipintervals.NewSkipIntervals())
}

func testDelete1(t *testing.T, siv interval.IIntervals) {
	iv0 := interval.NewInterval(0, 100)
	ivs := []*interval.Tinterval{
		interval.NewInterval(5, 10),
		interval.NewInterval(30, 50),
		interval.NewInterval(50, 100),
		interval.NewInterval(20, 30),
		interval.NewInterval(0, 5),
		interval.NewInterval(10, 20),
	}
	lens := []int{2, 3, 2, 2, 1, 0}
	siv.Insert(iv0)
	for i, iv := range ivs {
		siv.Delete(iv)
		assert.Equal(t, lens[i], siv.Length(), i)
		assert.False(t, siv.Present(iv), i)
	}
	siv.Insert(iv0)
	siv.Delete(ivs[0])
	assert.True(t, siv.Present(ivs[4]))
	assert.Equal(t, 2, siv.Length())
	siv.Delete(iv0)
	assert.Equal(t, 0, siv.Length())
}

func TestDelete1(t *testing.T) {
	// testDelete1(t, sliceintervals.NewIInterval())
	testDelete1(t, skipintervals.NewSkipIntervals())
}

func testRandom(t *testing.T, siv interval.IIntervals) {
	const N = 128
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ivs := make([]*interval.Tinterval, 0)
	for i := 0; i < N; i++ {
		s := r.Int31() % N
		ivs = append(ivs, interval.NewInterval(uint64(s), uint64(s+1)))
	}
	for _, iv := range ivs {
		siv.Insert(iv)
	}
	for _, iv := range ivs {
		assert.True(t, siv.Present(iv), iv.Marshal())
	}
}

func TestRandom1(t *testing.T) {
	// testRandom(t, sliceintervals.NewIInterval())
	testRandom(t, skipintervals.NewSkipIntervals())
}

const (
	N = 1000
	I = 1000
)

func testManyInorder(t *testing.T, newiv func() interval.IIntervals) {
	tot := time.Duration(0)
	var v reflect.Type
	for t := 0; t < I; t++ {
		ivs := newiv()
		v = reflect.TypeOf(ivs)
		start := time.Now()
		for i := uint64(0); i < N; i++ {
			ivs.Insert(interval.NewInterval(i, i+1))
		}
		tot += time.Since(start)
	}
	fmt.Printf("%v: %d inserts took on avg %v\n", v, N, tot/time.Duration(I))
}

func TestManyInOrder(t *testing.T) {
	testManyInorder(t, skipintervals.NewSkipIInterval)
	testManyInorder(t, sliceintervals.NewIInterval)
}

func testManyGaps(t *testing.T, newiv func() interval.IIntervals) {
	const (
		B = 10
	)
	tot := time.Duration(0)
	var v reflect.Type
	for t := 0; t < I; t++ {
		ivs := newiv()
		v = reflect.TypeOf(ivs)
		start := time.Now()
		for i := uint64(N * B); i > 1; i -= B {
			ivs.Insert(interval.NewInterval(i-1, i))
		}
		tot += time.Since(start)
	}
	fmt.Printf("%v: %d reverse inserts took on avg %v\n", v, N, tot/time.Duration(I))
}

func TestManyGaps(t *testing.T) {
	testManyGaps(t, skipintervals.NewSkipIInterval)
	testManyGaps(t, sliceintervals.NewIInterval)
}

func bench(siv interval.IIntervals, ivs []*interval.Tinterval) {
	for i, iv := range ivs {
		siv.Insert(iv)
		d := i - len(ivs)/2
		if d >= 0 {
			siv.Delete(ivs[d])
		}
	}
}

func testManyRandom(t *testing.T, newiv func() interval.IIntervals) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	tot := time.Duration(0)
	var v reflect.Type
	for t := 0; t < I; t++ {
		siv := newiv()
		v = reflect.TypeOf(siv)
		ivs := make([]*interval.Tinterval, N)
		for i := 0; i < N; i++ {
			ivs[i] = interval.NewInterval(uint64(i), uint64(i+1))
		}
		// receive replies out of order
		for i := 0; i < N; i++ {
			j := r.Int31() % N
			t := ivs[i]
			ivs[i] = ivs[j]
			ivs[j] = t
		}
		start := time.Now()
		bench(siv, ivs)
		tot += time.Since(start)
	}
	fmt.Printf("%v: %d random ins/del took on avg %v\n", v, N, tot/time.Duration(I))
}

func TestManyRandom(t *testing.T) {
	testManyRandom(t, skipintervals.NewSkipIInterval)
	testManyRandom(t, sliceintervals.NewIInterval)
}

func processIV(t *testing.T, retrieved map[uint64]bool, iv *interval.Tinterval) {
	for i := iv.Start; i < iv.End; i++ {
		assert.False(t, retrieved[i], "Double retrieve %v", i)
		retrieved[i] = true
	}
}

// Check that the retrieved map contains all integers in the range
// [start, end], and not any integers before this range, which have not already
// been checked.
func checkRetrieved(t *testing.T, retrieved map[uint64]bool, prevend, start, end uint64) {
	if prevend > 0 {
		for i := prevend; i < start; i++ {
			assert.False(t, retrieved[i], "Contains %v", i)
		}
	}
	for i := start; i <= end; i++ {
		assert.True(t, retrieved[i], "Doesn't contain %v", i)
	}
}

func doNext(t *testing.T, ivs *intervals.Intervals, retrieved map[uint64]bool, starts, ends []uint64) {
	var iv interval.Tinterval
	assert.Equal(t, len(starts), len(ends))
	iv = ivs.Next()
	if !assert.NotNil(t, iv) {
		db.DPrintf(db.ERROR, "Error")
		assert.False(t, true, "Error")
	}
	processIV(t, retrieved, &iv)
	for i := range starts {
		start := starts[i]
		end := ends[i]
		var prevend uint64
		if i > 0 {
			prevend = ends[i-1] + 1
		}
		checkRetrieved(t, retrieved, prevend, start, end)
	}
}

// Spec we are testing for:
// * Unless ivs.ResetNext is called, the same number should never be returned
// twice from ivs.Next, assuming it was never inserted twice.
// * All intervals inserted in ivs will eventually be returned by Next.
func TestNextInsert(t *testing.T) {
	retrieved := make(map[uint64]bool)
	ivs := intervals.NewIntervals(12345)
	ivs.Insert(interval.NewInterval(0, 10))
	doNext(t, ivs, retrieved, []uint64{0}, []uint64{9})
	assert.Equal(t, 1, ivs.Length())
	ivs.Insert(interval.NewInterval(10, 20))
	doNext(t, ivs, retrieved, []uint64{0}, []uint64{19})
	assert.Equal(t, 1, ivs.Length())
	ivs.Insert(interval.NewInterval(25, 26))
	ivs.Insert(interval.NewInterval(22, 23))
	ivs.Insert(interval.NewInterval(23, 24))
	assert.Equal(t, 3, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 22}, []uint64{19, 23})
	ivs.Insert(interval.NewInterval(21, 22))
	assert.Equal(t, 3, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 21}, []uint64{19, 23})
	ivs.Insert(interval.NewInterval(25, 101))
	assert.Equal(t, 3, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 21, 25}, []uint64{19, 23, 100})
	ivs.Insert(interval.NewInterval(20, 21))
	assert.Equal(t, 2, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 20, 25}, []uint64{19, 23, 100})
	ivs.Insert(interval.NewInterval(24, 25))
	assert.Equal(t, 1, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0}, []uint64{100})
}

func TestNextDelete(t *testing.T) {
	retrieved := make(map[uint64]bool)
	ivs := intervals.NewIntervals(12345)
	ivs.Insert(interval.NewInterval(0, 10))
	doNext(t, ivs, retrieved, []uint64{0}, []uint64{9})
	assert.Equal(t, 1, ivs.Length())
	ivs.Insert(interval.NewInterval(10, 20))
	doNext(t, ivs, retrieved, []uint64{0}, []uint64{19})
	assert.Equal(t, 1, ivs.Length())
	ivs.Insert(interval.NewInterval(25, 26))
	ivs.Insert(interval.NewInterval(22, 23))
	ivs.Insert(interval.NewInterval(23, 24))
	assert.Equal(t, 3, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 22}, []uint64{19, 23})
	ivs.Insert(interval.NewInterval(21, 22))
	assert.Equal(t, 3, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 21}, []uint64{19, 23})
	ivs.Insert(interval.NewInterval(25, 101))
	assert.Equal(t, 3, ivs.Length())
	ivs.Delete(interval.NewInterval(50, 51))
	assert.Equal(t, 4, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 21, 25}, []uint64{19, 23, 49})
	doNext(t, ivs, retrieved, []uint64{0, 21, 25, 51}, []uint64{19, 23, 49, 100})
	ivs.Insert(interval.NewInterval(20, 21))
	doNext(t, ivs, retrieved, []uint64{0, 20, 25, 51}, []uint64{19, 23, 49, 100})
	ivs.Insert(interval.NewInterval(24, 25))
	doNext(t, ivs, retrieved, []uint64{0, 51}, []uint64{49, 100})
	ivs.Insert(interval.NewInterval(50, 51))
	doNext(t, ivs, retrieved, []uint64{0}, []uint64{100})
}

func TestNextReset(t *testing.T) {
	retrieved := make(map[uint64]bool)
	ivs := intervals.NewIntervals(12345)
	ivs.Insert(interval.NewInterval(0, 10))
	doNext(t, ivs, retrieved, []uint64{0}, []uint64{9})
	assert.Equal(t, 1, ivs.Length())
	ivs.Insert(interval.NewInterval(10, 20))
	doNext(t, ivs, retrieved, []uint64{0}, []uint64{19})
	assert.Equal(t, 1, ivs.Length())
	ivs.Insert(interval.NewInterval(25, 26))
	ivs.Insert(interval.NewInterval(22, 23))
	ivs.Insert(interval.NewInterval(23, 24))
	assert.Equal(t, 3, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 22}, []uint64{19, 23})
	ivs.Insert(interval.NewInterval(21, 22))
	assert.Equal(t, 3, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 21}, []uint64{19, 23})
	ivs.Insert(interval.NewInterval(25, 101))
	assert.Equal(t, 3, ivs.Length())
	ivs.Delete(interval.NewInterval(50, 51))
	assert.Equal(t, 4, ivs.Length())
	doNext(t, ivs, retrieved, []uint64{0, 21, 25}, []uint64{19, 23, 49})
	doNext(t, ivs, retrieved, []uint64{0, 21, 25, 51}, []uint64{19, 23, 49, 100})
	ivs.Insert(interval.NewInterval(20, 21))
	doNext(t, ivs, retrieved, []uint64{0, 20, 25, 51}, []uint64{19, 23, 49, 100})
	newRetrieved := make(map[uint64]bool)
	ivs.ResetNext()
	doNext(t, ivs, newRetrieved, []uint64{0}, []uint64{19})
	ivs.Insert(interval.NewInterval(24, 25))
	ivs.Insert(interval.NewInterval(50, 51))
	doNext(t, ivs, newRetrieved, []uint64{0}, []uint64{100})
}
