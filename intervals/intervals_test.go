package intervals

import (
	"log"

	"github.com/stretchr/testify/assert"
	"testing"

	sp "sigmaos/sigmap"
)

func TestSimple(t *testing.T) {
	ivs := MkIntervals()
	ivs.Insert(&sp.Tinterval{1, 2})
	ivs.Insert(&sp.Tinterval{2, 3})
	ivs.Delete(&sp.Tinterval{1, 2})
	assert.Equal(t, 1, len(ivs.ivs))
}

func TestContains(t *testing.T) {
	ivs := MkIntervals()
	ivs.Insert(&sp.Tinterval{0, 10})
	ivs.Insert(&sp.Tinterval{90, 100})
	assert.True(t, ivs.Contains(0))
	assert.False(t, ivs.Contains(10))
	assert.False(t, ivs.Contains(11))
	assert.True(t, ivs.Contains(90))
	assert.False(t, ivs.Contains(100))
	assert.False(t, ivs.Contains(200))
}

func TestInsert(t *testing.T) {
	ivs := MkIntervals()
	ivs.Insert(&sp.Tinterval{0, 10})
	ivs.Insert(&sp.Tinterval{10, 20})
	assert.Equal(t, 1, len(ivs.ivs))
	ivs.Insert(&sp.Tinterval{15, 20})
	assert.Equal(t, 1, len(ivs.ivs))
	ivs.Insert(&sp.Tinterval{30, 40})
	assert.Equal(t, 2, len(ivs.ivs))
	ivs.Insert(&sp.Tinterval{20, 25})
	assert.Equal(t, 2, len(ivs.ivs))
	ivs.Insert(&sp.Tinterval{50, 60})
	assert.Equal(t, 3, len(ivs.ivs))
	ivs.Insert(&sp.Tinterval{70, 80})
	assert.Equal(t, 4, len(ivs.ivs))
	ivs.Insert(&sp.Tinterval{40, 50})
	assert.Equal(t, 3, len(ivs.ivs))
	ivs.Insert(&sp.Tinterval{25, 30})
	assert.Equal(t, 2, len(ivs.ivs))
	ivs.Insert(&sp.Tinterval{60, 70})
	assert.Equal(t, 1, len(ivs.ivs))
}

func TestDelete(t *testing.T) {
	ivs := MkIntervals()
	ivs.Insert(&sp.Tinterval{0, 100})
	log.Printf("ivs %v\n", ivs.ivs)
	ivs.Delete(&sp.Tinterval{5, 10})
	assert.Equal(t, 2, len(ivs.ivs))
	log.Printf("ivs %v\n", ivs.ivs)
	ivs.Delete(&sp.Tinterval{30, 50})
	log.Printf("ivs %v\n", ivs.ivs)
	assert.Equal(t, 3, len(ivs.ivs))
	ivs.Delete(&sp.Tinterval{50, 100})
	log.Printf("ivs %v\n", ivs.ivs)
	assert.Equal(t, 2, len(ivs.ivs))
	ivs.Delete(&sp.Tinterval{20, 30})
	assert.Equal(t, 2, len(ivs.ivs))
	log.Printf("ivs %v\n", ivs.ivs)
	ivs.Delete(&sp.Tinterval{0, 5})
	log.Printf("ivs %v\n", ivs.ivs)
	assert.Equal(t, 1, len(ivs.ivs))
	ivs.Delete(&sp.Tinterval{10, 20})
	assert.Equal(t, 0, len(ivs.ivs))

	ivs.Insert(&sp.Tinterval{0, 100})
	log.Printf("ivs %v\n", ivs.ivs)
	ivs.Delete(&sp.Tinterval{5, 10})
	assert.Equal(t, 2, len(ivs.ivs))
	log.Printf("ivs %v\n", ivs.ivs)
	ivs.Delete(&sp.Tinterval{0, 100})
	log.Printf("ivs %v\n", ivs.ivs)
	assert.Equal(t, 0, len(ivs.ivs))
}
