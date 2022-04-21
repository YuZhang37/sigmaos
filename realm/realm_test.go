package realm_test

import (
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	db "ulambda/debug"
	"ulambda/fslib"
	"ulambda/linuxsched"
	"ulambda/proc"
	"ulambda/procclnt"
	"ulambda/realm"
)

const (
	SLEEP_TIME_MS = 3000
)

type Tstate struct {
	t        *testing.T
	e        *realm.TestEnv
	cfg      *realm.RealmConfig
	realmFsl *fslib.FsLib
	*fslib.FsLib
	*procclnt.ProcClnt
}

func makeTstate(t *testing.T) *Tstate {
	ts := &Tstate{}
	bin := ".."
	e := realm.MakeTestEnv(bin)
	cfg, err := e.Boot()
	if err != nil {
		t.Fatalf("Boot %v\n", err)
	}
	ts.e = e
	ts.cfg = cfg

	err = ts.e.BootMachined()
	if err != nil {
		t.Fatalf("Boot Machined 2: %v", err)
	}

	program := "realm_test"
	ts.realmFsl = fslib.MakeFsLibAddr(program, fslib.Named())
	ts.FsLib = fslib.MakeFsLibAddr(program, cfg.NamedAddrs)

	ts.ProcClnt = procclnt.MakeProcClntInit(proc.GenPid(), ts.FsLib, program, cfg.NamedAddrs)

	linuxsched.ScanTopology()

	ts.t = t

	return ts
}

func (ts *Tstate) spawnSpinner() proc.Tpid {
	pid := proc.GenPid()
	a := proc.MakeProcPid(pid, "bin/user/spinner", []string{"name/"})
	a.Ncore = proc.Tcore(1)
	err := ts.Spawn(a)
	if err != nil {
		db.DFatalf("Error Spawn in RealmBalanceBenchmark.spawnSpinner: %v", err)
	}
	go func() {
		ts.WaitExit(pid)
	}()
	return pid
}

// Check that the test realm has min <= nMachineds <= max machineds assigned to it
func (ts *Tstate) checkNMachineds(min int, max int) {
	db.DPrintf("TEST", "Checking num machineds")
	cfg := realm.GetRealmConfig(ts.realmFsl, realm.TEST_RID)
	nMachineds := len(cfg.MachinedsActive)
	db.DPrintf("TEST", "Done Checking num machineds")
	ok := assert.True(ts.t, nMachineds >= min && nMachineds <= max, "Wrong number of machineds (x=%v), expected %v <= x <= %v", nMachineds, min, max)
	if !ok {
		debug.PrintStack()
	}
}

func TestStartStop(t *testing.T) {
	ts := makeTstate(t)
	ts.checkNMachineds(1, 1)
	ts.e.Shutdown()
}

// Start enough spinning lambdas to fill two Machineds, check that the test
// realm's allocation expanded, evict the spinning lambdas, and check the
// realm's allocation shrank. Then spawn and evict a few more to make sure we
// can still spawn after shrinking.  Assumes other machines in the cluster have
// the same number of cores.
func TestRealmGrowShrink(t *testing.T) {
	ts := makeTstate(t)

	N := int(linuxsched.NCores) / 2

	db.DPrintf("TEST", "Starting %v spinning lambdas", N)
	pids := []proc.Tpid{}
	for i := 0; i < N; i++ {
		pids = append(pids, ts.spawnSpinner())
	}

	db.DPrintf("TEST", "Sleeping for a bit")
	time.Sleep(SLEEP_TIME_MS * time.Millisecond)

	db.DPrintf("TEST", "Starting %v more spinning lambdas", N)
	for i := 0; i < N; i++ {
		pids = append(pids, ts.spawnSpinner())
	}

	db.DPrintf("TEST", "Sleeping again")
	time.Sleep(SLEEP_TIME_MS * time.Millisecond)

	ts.checkNMachineds(2, 100)

	db.DPrintf("TEST", "Evicting %v spinning lambdas", N+7*N/8)
	cnt := 0
	for i := 0; i < N+7*N/8; i++ {
		err := ts.Evict(pids[0])
		assert.Nil(ts.t, err, "Evict")
		cnt += 1
		pids = pids[1:]
	}

	db.DPrintf("TEST", "Sleeping yet again")
	time.Sleep(SLEEP_TIME_MS * time.Millisecond)

	ts.checkNMachineds(1, 1)

	db.DPrintf("TEST", "Starting %v more spinning lambdas", N/2)
	for i := 0; i < int(N/2); i++ {
		pids = append(pids, ts.spawnSpinner())
	}

	db.DPrintf("TEST", "Sleeping yet again")
	time.Sleep(SLEEP_TIME_MS * time.Millisecond)

	ts.checkNMachineds(1, 100)

	db.DPrintf("TEST", "Evicting %v spinning lambdas again", N/2)
	for i := 0; i < int(N/2); i++ {
		ts.Evict(pids[0])
		pids = pids[1:]
	}

	ts.checkNMachineds(1, 1)

	ts.e.Shutdown()
}
