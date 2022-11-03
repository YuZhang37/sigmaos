package simbid

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	NTRIAL                   = 1
	AVG_ARRIVAL_RATE float64 = 0.1 // per tick
)

func TestRun(t *testing.T) {
	ps := make(Procs, 0)
	ps = append(ps, &Proc{4, 4, 0, 0.1, 0.5})
	l, _ := ps.run(0.1)
	assert.Equal(t, l, FTick(0.5))

	//fmt.Printf("run: %v %v\n", l, ps)

	ps = append(ps, &Proc{4, 4, 0, 0.1, 0.3})
	l, _ = ps.run(0.1)
	assert.Equal(t, l, FTick(0.8))

	//fmt.Printf("run: %v %v\n", l, ps)

	ps = append(ps, &Proc{4, 4, 0, 0.1, 0.4})
	l, _ = ps.run(0.1)
	assert.Equal(t, l, FTick(1.0))
	assert.Equal(t, FTick(3.5), ps[2].nTick)

	//fmt.Printf("run: %v %v\n", l, ps)

	l, d := ps.run(0.1)
	assert.Equal(t, l, FTick(1.0))
	assert.Equal(t, ps[1].nTick, FTick(3.0))
	assert.Equal(t, d, Tick(0))

	//fmt.Printf("run: %v %v\n", l, ps)

	ps = append(ps, &Proc{4, 4, 0, 0.1, 0.7})
	l, d = ps.run(0.1)
	assert.Equal(t, l, FTick(1.0))
	assert.Equal(t, ps[0].ci, FTick(0.4))
	assert.Equal(t, len(ps), 2)
	assert.Equal(t, d, Tick(0))

	l, d = ps.run(0.1)

	assert.Equal(t, d, Tick(0))

	l, d = ps.run(0.1)
	l, d = ps.run(0.1)
	l, d = ps.run(0.1)
	assert.Equal(t, d, Tick(1))
	//fmt.Printf("run: %v %v %v\n", l, ps, d)
}

const HIGH = 10

func mkArrival(t int) []float64 {
	ls := make([]float64, t, t)
	ls[0] = HIGH * AVG_ARRIVAL_RATE
	for i := 1; i < t; i++ {
		ls[i] = AVG_ARRIVAL_RATE
	}
	return ls
}

func TestOneTenant(t *testing.T) {
	nTenant := 1
	nTick := 100
	nNode := 50
	ls := mkArrival(nTenant)
	w := mkWorld(nNode, nTenant, 1, ls, Tick(nTick), policyFixed, 1.0)
	sim := runSim(w)
	sim.stats()
	ten := &sim.tenants[0]
	assert.True(t, sim.nproc > nTick-10 && sim.nproc < nTick+10)
	assert.True(t, ten.maxnode <= HIGH)
	assert.True(t, int(sim.proclen)/sim.nproc == (MAX_SERVICE_TIME+1)/2)
	fmt.Printf("ntick %v\n", ten.ntick)
	assert.True(t, int(ten.ntick)/ten.nproc == (MAX_SERVICE_TIME+1)/2)
	assert.True(t, int(ten.nwork)/ten.nproc == (MAX_SERVICE_TIME+1)/2)
	assert.True(t, ten.nwait == 0)
	assert.True(t, ten.ndelay == 0)
	assert.True(t, ten.nevict == 0)
	assert.True(t, ten.nmigrate == 0)
}

func TestComputeI(t *testing.T) {
	nTenant := 1
	nTick := 100
	nNode := 50
	ls := mkArrival(nTenant)
	w := mkWorld(nNode, nTenant, 1, ls, Tick(nTick), policyFixed, 0.5)
	sim := runSim(w)
	sim.stats()
	ten := &sim.tenants[0]
	assert.True(t, sim.nproc > nTick-10 && sim.nproc < nTick+10)
	assert.True(t, ten.maxnode >= HIGH/2)
	assert.True(t, int(sim.proclen)/sim.nproc == (MAX_SERVICE_TIME+1)/2)
	fmt.Printf("%d %d\n", (MAX_SERVICE_TIME+1)/2, (MAX_SERVICE_TIME+1)/4)
	fmt.Printf("%d %d\n", int(ten.ntick)/ten.nproc, int(ten.nwork)/ten.nproc)
	assert.True(t, int(ten.ntick)/ten.nproc == (MAX_SERVICE_TIME+1)/4)
	assert.True(t, int(ten.nwork)/ten.nproc == (MAX_SERVICE_TIME+1)/4)
}

func testMigration(t *testing.T) {
	// policies := []Tpolicy{policyFixed, policyLast, policyBidMore}
	policies := []Tpolicy{policyBidMore}
	//policies := []Tpolicy{policyFixed}

	nNode := 50
	nTenant := 100
	nTick := Tick(1000)
	ls := mkArrival(nTenant)
	npm := []int{1, 5, nNode}
	// npm := []int{1}

	for i := 0; i < NTRIAL; i++ {
		for _, p := range policies {
			for _, n := range npm {
				runSim(mkWorld(nNode, nTenant, n, ls, nTick, p, 0.5))
			}
		}
	}
}
