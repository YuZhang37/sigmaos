package cachesrvrepl

import (
	"sync"

	// "google.golang.org/protobuf/proto"

	//cacheproto "sigmaos/cache/proto"
	replproto "sigmaos/cache/replproto"
	// replproto "sigmaos/repl/proto"

	"sigmaos/ctx"
	db "sigmaos/debug"
	"sigmaos/fs"
	"sigmaos/proc"
	"sigmaos/repl"
	"sigmaos/replraft"
	"sigmaos/rpcsrv"
	// "sigmaos/serr"
	// sp "sigmaos/sigmap"
)

//
// Replicated CacheSrv with same RPC interface (CacheSrv) has unreplicated CacheSrv
//

type CacheSrvRepl struct {
	mu      sync.Mutex
	raftcfg *replraft.RaftConfig
	replSrv repl.Server
	rpcs    *rpcsrv.RPCSrv
	rt      *ReplyTable
}

func NewCacheSrvRepl(raftcfg *replraft.RaftConfig, svci any) *CacheSrvRepl {
	cs := &CacheSrvRepl{
		raftcfg: raftcfg,
		rpcs:    rpcsrv.NewRPCSrv(svci, nil),
		rt:      NewReplyTable(),
	}
	cs.replSrv = raftcfg.MakeServer(cs.applyOp)
	cs.replSrv.Start()
	db.DPrintf(db.ALWAYS, "%v: Starting repl server: %v %v", proc.GetName(), svci, raftcfg)
	return cs
}

func (cs *CacheSrvRepl) applyOp(req *replproto.ReplOpRequest, rep *replproto.ReplOpReply) error {
	db.DPrintf(db.CACHESRV_REPL, "ApplyOp %v\n", req)
	duplicate, err, b := cs.rt.IsDuplicate(req.TclntId(), req.Tseqno())
	if duplicate {
		db.DPrintf(db.CACHESRV_REPL, "ApplyOp duplicate %v\n", req)
		rep.Msg = b
		return err
	}
	if b, err := cs.rpcs.ServeRPC(ctx.MkCtxNull(), req.Method, req.Msg); err != nil {
		cs.rt.PutReply(req.TclntId(), req.Tseqno(), err, nil)
		return err
	} else {
		if rep != nil {
			rep.Msg = b
		}
		cs.rt.PutReply(req.TclntId(), req.Tseqno(), nil, b)
	}
	return nil
}

func (cs *CacheSrvRepl) SubmitOp(ctx fs.CtxI, req replproto.ReplOpRequest, rep *replproto.ReplOpReply) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	db.DPrintf(db.CACHESRV_REPL, "SubmitOp %v\n", req)
	if err := cs.replSrv.Process(&req, rep); err != nil {
		db.DPrintf(db.CACHESRV_REPL, "Process req %v err %v\n", req, err)
		return err
	}
	db.DPrintf(db.CACHESRV_REPL, "Process req %v done rep %v\n", req, rep)
	return nil
}