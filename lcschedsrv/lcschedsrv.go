package lcschedsrv

import (
	"path"
	"sync"

	db "sigmaos/debug"
	"sigmaos/fs"
	proto "sigmaos/lcschedsrv/proto"
	"sigmaos/memfssrv"
	"sigmaos/perf"
	"sigmaos/proc"
	pqproto "sigmaos/procqsrv/proto"
	"sigmaos/rpcclnt"
	schedd "sigmaos/schedd/proto"
	"sigmaos/scheddclnt"
	sp "sigmaos/sigmap"
	"sigmaos/sigmasrv"
)

type LCSched struct {
	sync.Mutex
	cond       *sync.Cond
	mfs        *memfssrv.MemFs
	scheddclnt *scheddclnt.ScheddClnt
	qs         map[sp.Trealm]*Queue
	schedds    map[string]*Resources
}

func NewLCSched(mfs *memfssrv.MemFs) *LCSched {
	lcs := &LCSched{
		mfs:        mfs,
		scheddclnt: scheddclnt.NewScheddClnt(mfs.SigmaClnt().FsLib),
		qs:         make(map[sp.Trealm]*Queue),
		schedds:    make(map[string]*Resources),
	}
	lcs.cond = sync.NewCond(&lcs.Mutex)
	return lcs
}

func (lcs *LCSched) Enqueue(ctx fs.CtxI, req pqproto.EnqueueRequest, res *pqproto.EnqueueResponse) error {
	p := proc.NewProcFromProto(req.ProcProto)
	db.DPrintf(db.LCSCHED, "[%v] Enqueued %v", p.GetRealm(), p)

	ch := lcs.addProc(p)
	res.KernelID = <-ch
	return nil
}

func (lcs *LCSched) RegisterSchedd(ctx fs.CtxI, req proto.RegisterScheddRequest, res *proto.RegisterScheddResponse) error {
	lcs.Lock()
	defer lcs.Unlock()

	db.DPrintf(db.LCSCHED, "Register Schedd id:%v mcpu:%v mem:%v", req.KernelID, req.McpuInt, req.MemInt)
	if _, ok := lcs.schedds[req.KernelID]; ok {
		db.DFatalf("Double-register schedd %v", req.KernelID)
	}
	lcs.schedds[req.KernelID] = newResources(req.McpuInt, req.MemInt)
	lcs.cond.Broadcast()
	return nil
}

func (lcs *LCSched) schedule() {
	lcs.Lock()
	defer lcs.Unlock()

	// Keep scheduling forever.
	for {
		var success bool
		for realm, q := range lcs.qs {
			db.DPrintf(db.LCSCHED, "Try to schedule realm %v", realm)
			for kid, r := range lcs.schedds {
				p, ch, ok := q.Dequeue(r.mcpu, r.mem)
				if ok {
					lcs.runProc(kid, p, ch, r)
					success = true
				}
			}
		}
		// If scheduling was unsuccessful, wait.
		if !success {
			lcs.cond.Wait()
		}
	}
}

// Caller holds lock
func (lcs *LCSched) runProc(kernelID string, p *proc.Proc, ch chan string, r *Resources) {
	// Alloc resources for the proc
	r.alloc(p)
	rpcc, err := lcs.scheddclnt.GetScheddClnt(kernelID)
	if err != nil {
		db.DFatalf("Error getScheddClnt: %v", err)
	}
	// Start the proc on the chosen schedd.
	req := &schedd.ForceRunRequest{
		ProcProto: p.GetProto(),
	}
	res := &schedd.ForceRunResponse{}
	if err := rpcc.RPC("Schedd.ForceRun", req, res); err != nil {
		db.DFatalf("Schedd.Run %v err %v", kernelID, err)
	}
	// Notify the spawner that a schedd has been chosen.
	ch <- kernelID

	go lcs.waitProcExit(rpcc, kernelID, p, r)
}

func (lcs *LCSched) waitProcExit(rpcc *rpcclnt.RPCClnt, kernelID string, p *proc.Proc, r *Resources) {
	// RPC the schedd this proc was spawned on to wait for the proc to exit.
	db.DPrintf(db.LCSCHED, "WaitExit %v RPC", p.GetPid())
	req := &schedd.WaitRequest{
		PidStr: p.GetPid().String(),
	}
	res := &schedd.WaitResponse{}
	if err := rpcc.RPC("Schedd.WaitExit", req, res); err != nil {
		db.DFatalf("Error Schedd WaitExit: %v", err)
	}
	// Lock to modify resource allocations
	lcs.Lock()
	defer lcs.Unlock()

	r.free(p)
	// Notify that more resources have become available (and thus procs may now
	// be schedulable).
	lcs.cond.Broadcast()
}

func (lcs *LCSched) addProc(p *proc.Proc) chan string {
	lcs.Lock()
	defer lcs.Unlock()

	q, ok := lcs.qs[p.GetRealm()]
	if !ok {
		q = lcs.addRealmQueueL(p.GetRealm())
	}
	// Enqueue the proc according to its realm
	ch := q.Enqueue(p)
	// Signal that a new proc may be runnable.
	lcs.cond.Signal()
	return ch
}

// Caller must hold lock.
func (lcs *LCSched) addRealmQueueL(realm sp.Trealm) *Queue {
	q := newQueue()
	lcs.qs[realm] = q
	return q
}

// Run an LCSched
func Run() {
	pcfg := proc.GetProcEnv()
	mfs, err := memfssrv.NewMemFs(path.Join(sp.LCSCHED, pcfg.GetPID().String()), pcfg)

	if err != nil {
		db.DFatalf("Error NewMemFs: %v", err)
	}

	lcs := NewLCSched(mfs)
	ssrv, err := sigmasrv.NewSigmaSrvMemFs(mfs, lcs)

	if err != nil {
		db.DFatalf("Error PDS: %v", err)
	}

	setupMemFsSrv(ssrv.MemFs)
	setupFs(ssrv.MemFs)
	// Perf monitoring
	p, err := perf.NewPerf(pcfg, perf.LCSCHED)

	if err != nil {
		db.DFatalf("Error NewPerf: %v", err)
	}

	defer p.Done()
	go lcs.schedule()

	ssrv.RunServer()
}