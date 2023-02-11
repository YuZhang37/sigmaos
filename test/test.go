package test

import (
	"flag"
	"fmt"
	"testing"

	"sigmaos/bootkernelclnt"
	db "sigmaos/debug"
	"sigmaos/kernel"
	"sigmaos/port"
	"sigmaos/proc"
	"sigmaos/realmclnt"
	"sigmaos/sigmaclnt"
	sp "sigmaos/sigmap"
)

const (
	BOOT_REALM = "realm"
	BOOT_ALL   = "all"
	BOOT_NAMED = "named"
	BOOT_NODE  = "node"

	NAMEDPORT = ":1111"

	FPORT port.Tport = 1112
	LPORT port.Tport = 11112

	NPORT = 1000
)

var containerIP string
var start bool
var tag string
var ports *port.PortPool

func init() {
	flag.StringVar(&containerIP, "containerIP", "127.0.0.1", "IP addr for container")
	flag.StringVar(&tag, "tag", "", "Docker image tag")
	flag.BoolVar(&start, "start", false, "Start system")
	ports = port.MakePortPool(FPORT, LPORT)
}

func Mbyte(sz sp.Tlength) float64 {
	return float64(sz) / float64(sp.MBYTE)
}

func TputStr(sz sp.Tlength, ms int64) string {
	s := float64(ms) / 1000
	return fmt.Sprintf("%.2fMB/s", Mbyte(sz)/s)
}

func Tput(sz sp.Tlength, ms int64) float64 {
	t := float64(ms) / 1000
	return Mbyte(sz) / t
}

type Tstate struct {
	*sigmaclnt.SigmaClnt
	rc      *realmclnt.RealmClnt
	kclnts  []*bootkernelclnt.Kernel
	killidx int
	T       *testing.T
}

func MakeTstatePath(t *testing.T, path string) *Tstate {
	b, err := makeSysClntPath(t, path)
	if err != nil {
		db.DFatalf("MakeTstatePath: %v\n", err)
	}
	return b
}

func MakeTstate(t *testing.T) *Tstate {
	ts, err := makeSysClnt(t, BOOT_NAMED)
	if err != nil {
		db.DFatalf("MakeTstate: %v\n", err)
	}
	return ts
}

func MakeTstateAll(t *testing.T) *Tstate {
	ts, err := makeSysClnt(t, BOOT_ALL)
	if err != nil {
		db.DFatalf("MakeTstateAll: %v\n", err)
	}
	return ts
}

func MakeTstateWithRealms(t *testing.T) *Tstate {
	ts, err := makeSysClnt(t, BOOT_REALM)
	if err != nil {
		db.DFatalf("MakeTstateRealm: %v\n", err)
	}
	rc, err := realmclnt.MakeRealmClnt(ts.FsLib)
	if err != nil {
		db.DFatalf("MakeTstateRealm make realmclnt: %v\n", err)
	}
	ts.rc = rc
	return ts
}

func makeSysClntPath(t *testing.T, path string) (*Tstate, error) {
	if path == sp.NAMED {
		return makeSysClnt(t, BOOT_NAMED)
	} else {
		ts, err := makeSysClnt(t, BOOT_ALL)
		if err != nil {
			return nil, err
		}
		ts.RmDir(path)
		ts.MkDir(path, 0777)
		return ts, nil
	}
}

func makeSysClnt(t *testing.T, srvs string) (*Tstate, error) {
	namedport := []string{NAMEDPORT}
	kernelid := bootkernelclnt.GenKernelId()
	if start {
		r, err := ports.AllocRange(NPORT)
		if err != nil {
			return nil, err
		}
		ip, err := bootkernelclnt.Start(kernelid, tag, srvs, namedport, r)
		if err != nil {
			return nil, err
		}
		containerIP = ip
	}
	proc.SetPid(proc.Tpid("test-" + proc.GenPid().String()))
	namedAddr, err := kernel.SetNamedIP(containerIP, namedport)
	if err != nil {
		return nil, err
	}
	k, err := bootkernelclnt.MkKernelClnt(kernelid, "test", containerIP, namedAddr)
	if err != nil {
		return nil, err
	}
	return &Tstate{
		SigmaClnt: k.SigmaClnt,
		kclnts:    []*bootkernelclnt.Kernel{k},
		killidx:   0,
		T:         t,
	}, nil
}

func (ts *Tstate) BootNode(n int) error {
	for i := 0; i < n; i++ {
		r, err := ports.AllocRange(NPORT)
		if err != nil {
			return err
		}
		kclnt, err := bootkernelclnt.MkKernelClntStart(tag, "kclnt", BOOT_NODE, ts.NamedAddr(), r)
		if err != nil {
			return err
		}
		ts.kclnts = append(ts.kclnts, kclnt)
	}
	return nil
}

func (ts *Tstate) Boot(s string) error {
	return ts.kclnts[0].Boot(s)
}

func (ts *Tstate) BootFss3d() error {
	return ts.Boot(sp.S3REL)
}

func (ts *Tstate) KillOne(s string) error {
	idx := ts.killidx
	ts.killidx++
	return ts.kclnts[idx].Kill(s)
}

func (ts *Tstate) Shutdown() error {
	db.DPrintf(db.TEST, "Shutdown")
	db.DPrintf(db.TEST, "Done Shutdown")
	db.DPrintf(db.SYSTEM, "Shutdown")
	// Shut down other kernel running named last
	for i := len(ts.kclnts) - 1; i >= 0; i-- {
		if err := ts.kclnts[i].Shutdown(); err != nil {
			return err
		}
	}
	return nil
}
