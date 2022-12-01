package proxy

import (
	"os/user"
	"sync"

	db "sigmaos/debug"
	"sigmaos/fcall"
	"sigmaos/fidclnt"
	"sigmaos/fslib"
	"sigmaos/path"
	"sigmaos/pathclnt"
	"sigmaos/protclnt"
	"sigmaos/sessstatesrv"
	np "sigmaos/sigmap"
	"sigmaos/threadmgr"
)

type Npd struct {
	named []string
	st    *sessstatesrv.SessionTable
}

func MakeNpd() *Npd {
	npd := &Npd{fslib.Named(), nil}
	tm := threadmgr.MakeThreadMgrTable(nil, false)
	npd.st = sessstatesrv.MakeSessionTable(npd.mkProtServer, npd, tm)
	return npd
}

func (npd *Npd) mkProtServer(sesssrv np.SessServer, sid fcall.Tsession) np.Protsrv {
	return makeNpConn(npd.named)
}

func (npd *Npd) serve(fm *np.FcallMsg) {
	s := fcall.Tsession(fm.Fc.Session)
	sess, _ := npd.st.Lookup(s)
	reply, _, rerror := sess.Dispatch(fm.Msg)
	if rerror != nil {
		reply = rerror
	}
	fm1 := np.MakeFcallMsg(reply, fcall.Tclient(fm.Fc.Client), s, nil, nil, np.MakeFenceNull())
	fm1.Fc.Tag = fm.Fc.Tag
	sess.SendConn(fm1)
}

func (npd *Npd) Register(cid fcall.Tclient, sid fcall.Tsession, conn np.Conn) *fcall.Err {
	sess := npd.st.Alloc(cid, sid)
	sess.SetConn(conn)
	return nil
}

// Disassociate a connection with a session, and let it close gracefully.
func (npd *Npd) Unregister(cid fcall.Tclient, sid fcall.Tsession, conn np.Conn) {
	// If this connection hasn't been associated with a session yet, return.
	if sid == fcall.NoSession {
		return
	}
	sess := npd.st.Alloc(cid, sid)
	sess.UnsetConn(conn)
}

func (npd *Npd) SrvFcall(fcall fcall.Fcall) {
	fm := fcall.(*np.FcallMsg)
	go npd.serve(fm)
}

func (npd *Npd) Snapshot() []byte {
	return nil
}

func (npd *Npd) Restore(b []byte) {
}

// The connection from the kernel/client
type NpConn struct {
	mu    sync.Mutex
	clnt  *protclnt.Clnt
	uname string
	fidc  *fidclnt.FidClnt
	pc    *pathclnt.PathClnt
	named []string
	fm    *fidMap
}

func makeNpConn(named []string) *NpConn {
	npc := &NpConn{}
	npc.clnt = protclnt.MakeClnt()
	npc.named = named
	npc.fidc = fidclnt.MakeFidClnt()
	npc.pc = pathclnt.MakePathClnt(npc.fidc, np.Tsize(1_000_000))
	npc.fm = mkFidMap()
	return npc
}

// Make Wire-compatible Rerror
func MkRerrorWC(ec fcall.Terror) *np.Rerror {
	return &np.Rerror{ec.String()}
}

func (npc *NpConn) Version(args *np.Tversion, rets *np.Rversion) *np.Rerror {
	rets.Msize = args.Msize
	rets.Version = "9P2000"
	return nil
}

func (npc *NpConn) Auth(args *np.Tauth, rets *np.Rauth) *np.Rerror {
	return MkRerrorWC(fcall.TErrNotSupported)
}

func (npc *NpConn) Attach(args *np.Tattach, rets *np.Rattach) *np.Rerror {
	u, error := user.Current()
	if error != nil {
		return &np.Rerror{error.Error()}
	}
	npc.uname = u.Uid

	fid, err := npc.fidc.Attach(npc.uname, npc.named, "", "")
	if err != nil {
		db.DPrintf("PROXY", "Attach args %v err %v\n", args, err)
		return np.MkRerror(err)
	}
	if err := npc.pc.Mount(fid, np.NAMED); err != nil {
		db.DPrintf("PROXY", "Attach args %v mount err %v\n", args, err)
		return &np.Rerror{error.Error()}
	}
	rets.Qid = npc.fidc.Qid(fid)
	npc.fm.mapTo(args.Fid, fid)
	npc.fidc.Lookup(fid).SetPath(path.Split(np.NAMED))
	db.DPrintf("PROXY", "Attach args %v rets %v fid %v\n", args, rets, fid)
	return nil
}

func (npc *NpConn) Detach(rets *np.Rdetach, detach np.DetachF) *np.Rerror {
	db.DPrintf("PROXY", "Detach\n")
	return nil
}

func (npc *NpConn) Walk(args *np.Twalk, rets *np.Rwalk) *np.Rerror {
	fid, ok := npc.fm.lookup(args.Fid)
	if !ok {
		return MkRerrorWC(fcall.TErrNotfound)
	}
	fid1, err := npc.pc.Walk(fid, args.Wnames)
	if err != nil {
		db.DPrintf("PROXY", "Walk args %v err: %v\n", args, err)
		return MkRerrorWC(err.Code())
	}
	qids := npc.pc.Qids(fid1)
	rets.Qids = qids[len(qids)-len(args.Wnames):]
	npc.fm.mapTo(args.NewFid, fid1)
	db.DPrintf("PROXY", "Walk args %v rets: %v\n", args, rets)
	return nil
}

func (npc *NpConn) Open(args *np.Topen, rets *np.Ropen) *np.Rerror {
	fid, ok := npc.fm.lookup(args.Fid)
	if !ok {
		return MkRerrorWC(fcall.TErrNotfound)
	}
	qid, err := npc.fidc.Open(fid, args.Mode)
	if err != nil {
		db.DPrintf("PROXY", "Open args %v err: %v\n", args, err)
		return MkRerrorWC(err.Code())
	}
	rets.Qid = qid
	db.DPrintf("PROXY", "Open args %v rets: %v\n", args, rets)
	return nil
}

func (npc *NpConn) Watch(args *np.Twatch, rets *np.Ropen) *np.Rerror {
	return nil
}

func (npc *NpConn) Create(args *np.Tcreate, rets *np.Rcreate) *np.Rerror {
	fid, ok := npc.fm.lookup(args.Fid)
	if !ok {
		return MkRerrorWC(fcall.TErrNotfound)
	}
	fid1, err := npc.fidc.Create(fid, args.Name, args.Perm, args.Mode)
	if err != nil {
		db.DPrintf("PROXY", "Create args %v err: %v\n", args, err)
		return MkRerrorWC(err.Code())
	}
	if fid != fid1 {
		db.DPrintf(db.ALWAYS, "Create fid %v fid1 %v\n", fid, fid1)
	}
	rets.Qid = npc.pc.Qid(fid1)
	db.DPrintf("PROXY", "Create args %v rets: %v\n", args, rets)
	return nil
}

func (npc *NpConn) Clunk(args *np.Tclunk, rets *np.Rclunk) *np.Rerror {
	fid, ok := npc.fm.lookup(args.Fid)
	if !ok {
		return MkRerrorWC(fcall.TErrNotfound)
	}
	err := npc.fidc.Clunk(fid)
	if err != nil {
		db.DPrintf("PROXY", "Clunk: args %v err %v\n", args, err)
		return MkRerrorWC(err.Code())
	}
	db.DPrintf("PROXY", "Clunk: args %v rets %v\n", args, rets)
	return nil
}

func (npc *NpConn) Flush(args *np.Tflush, rets *np.Rflush) *np.Rerror {
	return nil
}

func (npc *NpConn) Read(args *np.Tread, rets *np.Rread) *np.Rerror {
	fid, ok := npc.fm.lookup(args.Fid)
	if !ok {
		return MkRerrorWC(fcall.TErrNotfound)
	}
	d, err := npc.fidc.ReadVU(fid, args.Offset, args.Count, np.NoV)
	if err != nil {
		db.DPrintf("PROXY", "Read: args %v err %v\n", args, err)
		return MkRerrorWC(err.Code())
	}
	rets.Data = d
	db.DPrintf("PROXY", "Read: args %v rets %v\n", args, rets)
	return nil
}

func (npc *NpConn) Write(args *np.Twrite, rets *np.Rwrite) *np.Rerror {
	fid, ok := npc.fm.lookup(args.Fid)
	if !ok {
		return MkRerrorWC(fcall.TErrNotfound)
	}
	n, err := npc.fidc.WriteV(fid, args.Offset, args.Data, np.NoV)
	if err != nil {
		db.DPrintf("PROXY", "Write: args %v err %v\n", args, err)
		return MkRerrorWC(err.Code())
	}
	rets.Count = n
	db.DPrintf("PROXY", "Write: args %v rets %v\n", args, rets)
	return nil
}

func (npc *NpConn) Remove(args *np.Tremove, rets *np.Rremove) *np.Rerror {
	fid, ok := npc.fm.lookup(args.Fid)
	if !ok {
		return MkRerrorWC(fcall.TErrNotfound)
	}
	err := npc.fidc.Remove(fid)
	if err != nil {
		db.DPrintf("PROXY", "Remove: args %v err %v\n", args, err)
		return MkRerrorWC(err.Code())
	}
	db.DPrintf("PROXY", "Remove: args %v rets %v\n", args, rets)
	return nil
}

func (npc *NpConn) RemoveFile(args *np.Tremovefile, rets *np.Rremove) *np.Rerror {
	return nil
}

func (npc *NpConn) Stat(args *np.Tstat, rets *np.Rstat) *np.Rerror {
	fid, ok := npc.fm.lookup(args.Fid)
	if !ok {
		return MkRerrorWC(fcall.TErrNotfound)
	}
	st, err := npc.fidc.Stat(fid)
	if err != nil {
		db.DPrintf("PROXY", "Stats: args %v err %v\n", args, err)
		return MkRerrorWC(err.Code())
	}
	rets.Stat = *st
	db.DPrintf("PROXY", "Stat: req %v rets %v\n", args, rets)
	return nil
}

func (npc *NpConn) Wstat(args *np.Twstat, rets *np.Rwstat) *np.Rerror {
	fid, ok := npc.fm.lookup(args.Fid)
	if !ok {
		return MkRerrorWC(fcall.TErrNotfound)
	}
	err := npc.fidc.Wstat(fid, &args.Stat)
	if err != nil {
		db.DPrintf("PROXY", "Wstats: args %v err %v\n", args, err)
		return MkRerrorWC(err.Code())
	}
	db.DPrintf("PROXY", "Wstat: req %v rets %v\n", args, rets)
	return nil
}

func (npc *NpConn) Renameat(args *np.Trenameat, rets *np.Rrenameat) *np.Rerror {
	return MkRerrorWC(fcall.TErrNotSupported)
}

func (npc *NpConn) ReadV(args *np.TreadV, rets *np.Rread) *np.Rerror {
	return MkRerrorWC(fcall.TErrNotSupported)
}

func (npc *NpConn) WriteV(args *np.TwriteV, rets *np.Rwrite) *np.Rerror {
	return MkRerrorWC(fcall.TErrNotSupported)
}

func (npc *NpConn) GetFile(args *np.Tgetfile, rets *np.Rgetfile) *np.Rerror {
	return nil
}

func (npc *NpConn) SetFile(args *np.Tsetfile, rets *np.Rwrite) *np.Rerror {
	return nil
}

func (npc *NpConn) PutFile(args *np.Tputfile, rets *np.Rwrite) *np.Rerror {
	return nil
}

func (npc *NpConn) WriteRead(args *np.Twriteread, rets *np.Rwriteread) *np.Rerror {
	return nil
}

func (npc *NpConn) Snapshot() []byte {
	return nil
}
