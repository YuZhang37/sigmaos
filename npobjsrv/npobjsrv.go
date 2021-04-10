package npobjsrv

import (
	"fmt"
	"log"
	"net"
	"sync"

	db "ulambda/debug"
	np "ulambda/ninep"
	"ulambda/npcodec"
)

type NpObjSrv interface {
	// Maybe pass uname to RootAttach()
	RootAttach(string) (NpObj, CtxI)
	Done()
	WatchTable() *WatchTable
	ConnTable() *ConnTable
}

type CtxI interface {
	Uname() string
}

type NpObjDir interface {
	Lookup(CtxI, []string) ([]NpObj, []string, error)
	Create(CtxI, string, np.Tperm, np.Tmode) (NpObj, error)
	ReadDir(CtxI, np.Toffset, np.Tsize, np.TQversion) ([]*np.Stat, error)
	WriteDir(CtxI, np.Toffset, []byte, np.TQversion) (np.Tsize, error)
}

type NpObjFile interface {
	Read(CtxI, np.Toffset, np.Tsize, np.TQversion) ([]byte, error)
	Write(CtxI, np.Toffset, []byte, np.TQversion) (np.Tsize, error)
}

type NpObj interface {
	Qid() np.Tqid
	Perm() np.Tperm
	Version() np.TQversion
	Size() np.Tlength
	Open(CtxI, np.Tmode) error
	Close(CtxI, np.Tmode) error
	Remove(CtxI, string) error
	Stat(CtxI) (*np.Stat, error)
	Rename(CtxI, string, string) error
}

type Fid struct {
	path []string
	obj  NpObj
	vers np.TQversion
	ctx  CtxI
}

func (f *Fid) String() string {
	return fmt.Sprintf("p %v", f.path)
}

func (f *Fid) Ctx() CtxI {
	return f.ctx
}

func (f *Fid) Path() []string {
	return f.path
}

func (f *Fid) Close() {
	f.obj = nil
}

func (f *Fid) Write(off np.Toffset, b []byte, v np.TQversion) (np.Tsize, error) {
	switch i := f.obj.(type) {
	case NpObjFile:
		return i.Write(f.ctx, off, b, v)
	case NpObjDir:
		return i.WriteDir(f.ctx, off, b, v)
	default:
		log.Fatalf("Write: unknown obj type %v\n", f.obj)
		return 0, nil
	}
}

func (f *Fid) readDir(off np.Toffset, count np.Tsize, v np.TQversion, rets *np.Rread) *np.Rerror {
	var dirents []*np.Stat
	var err error
	if f.obj.Size() > 0 && off >= np.Toffset(f.obj.Size()) {
		dirents = []*np.Stat{}
	} else {
		d := f.obj.(NpObjDir)
		dirents, err = d.ReadDir(f.ctx, off, count, v)

	}
	b, err := npcodec.Dir2Byte(off, count, dirents)
	if err != nil {
		return &np.Rerror{err.Error()}
	}
	rets.Data = b
	return nil
}

func (f *Fid) Read(off np.Toffset, count np.Tsize, v np.TQversion, rets *np.Rread) *np.Rerror {
	switch i := f.obj.(type) {
	case NpObjDir:
		return f.readDir(off, count, v, rets)
	case NpObjFile:
		b, err := i.Read(f.ctx, off, count, v)
		if err != nil {
			return &np.Rerror{err.Error()}
		}
		rets.Data = b
		return nil
	default:
		log.Fatalf("Read: unknown obj type %v\n", f.obj)
		return nil
	}
}

type NpConn struct {
	mu        sync.Mutex // for Fids
	conn      net.Conn
	fids      map[np.Tfid]*Fid
	osrv      NpObjSrv
	ephemeral map[NpObj]*Fid
	wt        *WatchTable
	ct        *ConnTable
}

func MakeNpConn(osrv NpObjSrv, conn net.Conn) *NpConn {
	npc := &NpConn{}
	npc.conn = conn
	npc.osrv = osrv
	npc.fids = make(map[np.Tfid]*Fid)
	npc.ephemeral = make(map[NpObj]*Fid)
	npc.wt = osrv.WatchTable()
	npc.ct = osrv.ConnTable()
	if npc.ct != nil {
		npc.ct.Add(npc)
	}
	db.DLPrintf("NPOBJ", "MakeNpConn %v -> %v", conn, npc)
	return npc
}

func (npc *NpConn) Addr() string {
	return npc.conn.LocalAddr().String()
}

func (npc *NpConn) lookup(fid np.Tfid) (*Fid, bool) {
	npc.mu.Lock()
	defer npc.mu.Unlock()
	f, ok := npc.fids[fid]
	return f, ok
}

func (npc *NpConn) add(fid np.Tfid, f *Fid) {
	npc.mu.Lock()
	defer npc.mu.Unlock()
	npc.fids[fid] = f
}

func (npc *NpConn) del(fid np.Tfid) {
	npc.mu.Lock()
	defer npc.mu.Unlock()
	o := npc.fids[fid].obj
	delete(npc.fids, fid)
	delete(npc.ephemeral, o)
}

func (npc *NpConn) Version(args np.Tversion, rets *np.Rversion) *np.Rerror {
	rets.Msize = args.Msize
	rets.Version = "9P2000"
	return nil
}

func (npc *NpConn) Auth(args np.Tauth, rets *np.Rauth) *np.Rerror {
	return np.ErrUnknownMsg
}

func (npc *NpConn) Attach(args np.Tattach, rets *np.Rattach) *np.Rerror {
	root, ctx := npc.osrv.RootAttach(args.Uname)
	npc.add(args.Fid, &Fid{[]string{}, root, root.Version(), ctx})
	rets.Qid = root.Qid()
	return nil
}

func (npc *NpConn) Detach() {
	db.DLPrintf("9POBJ", "Detach %v\n", npc.ephemeral)

	// Delete ephemeral files created on this connection
	for o, f := range npc.ephemeral {
		o.Remove(f.ctx, f.path[len(f.path)-1])
	}

	if npc.wt != nil {
		npc.wt.DeleteConn(npc)
	}
	if npc.ct != nil {
		npc.ct.Del(npc)
	}
}

func makeQids(os []NpObj) []np.Tqid {
	var qids []np.Tqid
	for _, o := range os {
		qids = append(qids, o.Qid())
	}
	return qids
}

func (npc *NpConn) Walk(args np.Twalk, rets *np.Rwalk) *np.Rerror {
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	db.DLPrintf("9POBJ", "Walk o %v args %v (%v)\n", f, args, len(args.Wnames))
	if len(args.Wnames) == 0 { // clone args.Fid?
		npc.add(args.NewFid, &Fid{f.path, f.obj, f.obj.Version(), f.ctx})
	} else {
		if !f.obj.Perm().IsDir() {
			return np.ErrNotfound
		}
		d := f.obj.(NpObjDir)
		os, rest, err := d.Lookup(f.ctx, args.Wnames)
		if err != nil {
			return &np.Rerror{err.Error()}
		}
		n := len(args.Wnames) - len(rest)
		p := append(f.path, args.Wnames[:n]...)
		lo := os[len(os)-1]
		npc.add(args.NewFid, &Fid{p, lo, lo.Version(), f.ctx})
		rets.Qids = makeQids(os)
	}
	return nil
}

// XXX call close? keep refcnt per obj?
func (npc *NpConn) Clunk(args np.Tclunk, rets *np.Rclunk) *np.Rerror {
	db.DLPrintf("9POBJ", "Clunk %v\n", args)
	_, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	npc.mu.Lock()
	delete(npc.fids, args.Fid)
	npc.mu.Unlock()
	return nil
}

func (npc *NpConn) Open(args np.Topen, rets *np.Ropen) *np.Rerror {
	db.DLPrintf("9POBJ", "Open %v\n", args)
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	db.DLPrintf("9POBJ", "f %v\n", f)
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}
	err := f.obj.Open(f.ctx, args.Mode)
	if err != nil {
		return &np.Rerror{err.Error()}
	}
	rets.Qid = f.obj.Qid()
	return nil
}

func (npc *NpConn) OpenV(args np.Topenv, rets *np.Ropen) *np.Rerror {
	db.DLPrintf("9POBJ", "Openv %v\n", args)
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}
	if args.Version != np.NoV && args.Version != f.obj.Version() {
		return &np.Rerror{"Version mismatch"}
	}
	npc.wt.Watch(npc, f.path)
	return nil
}

func (npc *NpConn) Create(args np.Tcreate, rets *np.Rcreate) *np.Rerror {
	db.DLPrintf("9POBJ", "Create %v\n", args)
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	db.DLPrintf("9POBJ", "f %v\n", f)
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}

	names := []string{args.Name}
	if !f.obj.Perm().IsDir() {
		return &np.Rerror{fmt.Sprintf("Not a directory")}
	}
	for {
		// XXX make create and setting watch atomic (hold lock on fid?)
		d := f.obj.(NpObjDir)
		o1, err := d.Create(f.ctx, names[0], args.Perm, args.Mode)
		db.DLPrintf("9POBJ", "Create %v %v %v\n", names[0], o1, err)
		if err == nil {
			nf := &Fid{append(f.path, names[0]), o1, o1.Version(), f.ctx}
			if args.Perm.IsEphemeral() {
				npc.mu.Lock()
				npc.ephemeral[o1] = nf
				npc.mu.Unlock()
			}
			npc.add(args.Fid, nf)
			rets.Qid = o1.Qid()
			break
		} else {
			if npc.wt != nil && err.Error() == "Name exists" && args.Mode&np.OWATCH == np.OWATCH { // retry?
				p := append(f.path, names[0])
				db.DLPrintf("9POBJ", "Watch %v\n", p)
				npc.wt.Watch(npc, p)
				db.DLPrintf("9POBJ", "Retry create %v\n", p)
			} else {
				return &np.Rerror{err.Error()}
			}
		}
	}
	return nil
}

func (npc *NpConn) Flush(args np.Tflush, rets *np.Rflush) *np.Rerror {
	return nil
}

func (npc *NpConn) Read(args np.Tread, rets *np.Rread) *np.Rerror {
	db.DLPrintf("9POBJ", "Read %v\n", args)
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	db.DLPrintf("9POBJ", "ReadFid %v %v\n", args, f)
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}
	return f.Read(args.Offset, args.Count, np.NoV, rets)
}

func (npc *NpConn) ReadV(args np.Treadv, rets *np.Rread) *np.Rerror {
	db.DLPrintf("9POBJ", "ReadV %v\n", args)
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}
	return f.Read(args.Offset, args.Count, f.vers, rets)
}

func (npc *NpConn) Write(args np.Twrite, rets *np.Rwrite) *np.Rerror {
	db.DLPrintf("9POBJ", "Write %v\n", args)
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}
	var err error
	rets.Count, err = f.Write(args.Offset, args.Data, np.NoV)
	if err != nil {
		return &np.Rerror{err.Error()}
	}
	return nil
}

func (npc *NpConn) WriteV(args np.Twritev, rets *np.Rwrite) *np.Rerror {
	db.DLPrintf("9POBJ", "WriteV %v\n", args)
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}
	var err error
	rets.Count, err = f.Write(args.Offset, args.Data, f.vers)
	if err != nil {
		return &np.Rerror{err.Error()}
	}
	return nil
}

func (npc *NpConn) Remove(args np.Tremove, rets *np.Rremove) *np.Rerror {
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}
	if len(f.path) == 0 { // exit?
		db.DLPrintf("9POBJ", "Done\n")
		npc.osrv.Done()
		return nil
	}
	db.DLPrintf("9POBJ", "Remove f %v\n", f)
	err := f.obj.Remove(f.ctx, f.path[len(f.path)-1])
	if err != nil {
		return &np.Rerror{err.Error()}
	}
	if npc.wt != nil {
		npc.wt.WakeupWatch(f.path)
	}
	npc.del(args.Fid)
	return nil
}

func (npc *NpConn) Stat(args np.Tstat, rets *np.Rstat) *np.Rerror {
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	db.DLPrintf("9POBJ", "Stat %v\n", f)
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}
	st, err := f.obj.Stat(f.ctx)
	if err != nil {
		return &np.Rerror{err.Error()}
	}
	rets.Stat = *st
	return nil
}

func (npc *NpConn) Wstat(args np.Twstat, rets *np.Rwstat) *np.Rerror {
	f, ok := npc.lookup(args.Fid)
	if !ok {
		return np.ErrUnknownfid
	}
	db.DLPrintf("9POBJ", "Wstat %v %v\n", f, args)
	if f.obj == nil {
		return &np.Rerror{"Closed by server"}
	}
	if args.Stat.Name != "" {
		// XXX if dst exists run watch?
		err := f.obj.Rename(f.ctx, f.path[len(f.path)-1], args.Stat.Name)
		if err != nil {
			return &np.Rerror{err.Error()}
		}
		dst := append(f.path[:len(f.path)-1], np.Split(args.Stat.Name)...)
		if npc.wt != nil {
			npc.wt.WakeupWatch(dst)
		}
		db.DLPrintf("9POBJ", "dst %v %v %v\n", dst, f.path[len(f.path)-1], args.Stat.Name)
		f.path = dst
	}
	// XXX ignore other Wstat for now
	return nil
}
