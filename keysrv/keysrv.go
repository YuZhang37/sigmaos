package keysrv

import (
	"path"
	"sync"

	"sigmaos/auth"
	db "sigmaos/debug"
	"sigmaos/fs"
	"sigmaos/keysrv/proto"
	"sigmaos/perf"
	"sigmaos/proc"
	"sigmaos/rpc"
	"sigmaos/sigmaclnt"
	sp "sigmaos/sigmap"
	"sigmaos/sigmasrv"
)

type rOnlyKeySrv struct {
	mu   sync.Mutex
	keys map[sp.Tsigner][]byte
}

type rwKeySrv struct {
	*rOnlyKeySrv
}

type KeySrv struct {
	*rwKeySrv
}

func newrOnlyKeySrv(masterPubKey auth.PublicKey) *rOnlyKeySrv {
	return &rOnlyKeySrv{
		keys: map[sp.Tsigner][]byte{
			auth.SIGMA_DEPLOYMENT_MASTER_SIGNER: masterPubKey.B64(),
		},
	}
}

func newrwKeySrv(masterPubKey auth.PublicKey) *rwKeySrv {
	return &rwKeySrv{
		rOnlyKeySrv: newrOnlyKeySrv(masterPubKey),
	}
}

func NewKeySrv(masterPubKey auth.PublicKey) *KeySrv {
	return &KeySrv{
		rwKeySrv: newrwKeySrv(masterPubKey),
	}
}

func (ks rOnlyKeySrv) GetKey(ctx fs.CtxI, req proto.GetKeyRequest, res *proto.GetKeyResponse) error {
	db.DFatalf("Unimplemented")
	return nil
}

func (ks rwKeySrv) SetKey(ctx fs.CtxI, req proto.SetKeyRequest, res *proto.SetKeyResponse) error {
	db.DFatalf("Unimplemented")
	return nil
}

func RunKeySrv(masterPubKey auth.PublicKey) {
	sc, err := sigmaclnt.NewSigmaClnt(proc.GetProcEnv())
	if err != nil {
		db.DFatalf("Error NewSigmaClnt: %v", err)
	}
	ks := NewKeySrv(masterPubKey)
	ssrv, err := sigmasrv.NewSigmaSrvClnt(sp.KEYD, sc, ks)
	if err != nil {
		db.DFatalf("Error NewSigmaSrv: %v", err)
	}
	// Add a directory for the RW rpc device
	if _, err := ssrv.Create(sp.RW_REL, sp.DMDIR|0777, sp.ORDWR, sp.NoLeaseId); err != nil {
		db.DFatalf("Error Create RW rpc dev dir: %v", err)
	}
	if _, err := ssrv.Create(sp.RONLY_REL, sp.DMDIR|0777, sp.ORDWR, sp.NoLeaseId); err != nil {
		db.DFatalf("Error Create RONLY rpc dev dir: %v", err)
	}
	if err := ssrv.AddRPCSrv(path.Join(sp.RW_REL, rpc.RPC), ks.rwKeySrv); err != nil {
		db.DFatalf("Error add RW rpc dev: %v", err)
	}
	if err := ssrv.AddRPCSrv(path.Join(sp.RONLY_REL, rpc.RPC), ks.rOnlyKeySrv); err != nil {
		db.DFatalf("Error add RONLY rpc dev: %v", err)
	}
	// Perf monitoring
	p, err := perf.NewPerf(sc.ProcEnv(), perf.KEYD)
	if err != nil {
		db.DFatalf("Error NewPerf: %v", err)
	}
	defer p.Done()
	db.DPrintf(db.KEYD, "Keyd running!")
	ssrv.RunServer()
}
