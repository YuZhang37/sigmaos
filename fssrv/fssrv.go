package fssrv

import (
	"log"
	"reflect"
	"runtime/debug"

	"ulambda/ctx"
	"ulambda/dir"
	"ulambda/fences"
	"ulambda/fs"
	"ulambda/fslib"
	"ulambda/netsrv"
	np "ulambda/ninep"
	"ulambda/proc"
	"ulambda/procclnt"
	"ulambda/protsrv"
	"ulambda/repl"
	"ulambda/sesscond"
	"ulambda/session"
	"ulambda/snapshot"
	"ulambda/stats"
	"ulambda/threadmgr"
	"ulambda/watch"
)

//
// There is one FsServer per server. The FsServer has one ProtSrv per
// 9p channel (i.e., TCP connection). Each channel may multiplex
// several users/clients.
//
// FsServer has a table with all sess conds in use so that it can
// unblock threads that are waiting in a sess cond when a session
// closes.
//

type FsServer struct {
	addr       string
	root       fs.Dir
	mkps       protsrv.MkProtServer
	rps        protsrv.RestoreProtServer
	stats      *stats.Stats
	snapDev    *snapshot.Dev
	st         *session.SessionTable
	sm         *session.SessionMgr
	sct        *sesscond.SessCondTable
	tmt        *threadmgr.ThreadMgrTable
	wt         *watch.WatchTable
	rft        *fences.RecentTable
	srv        *netsrv.NetServer
	replSrv    repl.Server
	rc         *repl.ReplyCache
	pclnt      *procclnt.ProcClnt
	snap       *snapshot.Snapshot
	done       bool
	replicated bool
	ch         chan bool
	fsl        *fslib.FsLib
}

func MakeFsServer(root fs.Dir, addr string, fsl *fslib.FsLib,
	mkps protsrv.MkProtServer, rps protsrv.RestoreProtServer, pclnt *procclnt.ProcClnt,
	config repl.Config) *FsServer {
	fssrv := &FsServer{}
	fssrv.replicated = config != nil && !reflect.ValueOf(config).IsNil()
	fssrv.root = root
	fssrv.addr = addr
	fssrv.mkps = mkps
	fssrv.rps = rps
	fssrv.stats = stats.MkStats(fssrv.root)
	fssrv.rft = fences.MakeRecentTable()
	fssrv.tmt = threadmgr.MakeThreadMgrTable(fssrv.process, fssrv.replicated)
	fssrv.sm = session.MakeSessionMgr(fssrv.Process)
	fssrv.st = session.MakeSessionTable(mkps, fssrv, fssrv.rft, fssrv.tmt, fssrv.sm)
	fssrv.sct = sesscond.MakeSessCondTable(fssrv.st)
	fssrv.wt = watch.MkWatchTable(fssrv.sct)
	fssrv.srv = netsrv.MakeNetServer(fssrv, addr)
	if !fssrv.replicated {
		fssrv.replSrv = nil
	} else {
		fssrv.snapDev = snapshot.MakeDev(fssrv, nil, fssrv.root)
		fssrv.rc = repl.MakeReplyCache()
		fssrv.replSrv = config.MakeServer(fssrv.tmt.AddThread())
		fssrv.replSrv.Start()
		log.Printf("Starting repl server")
	}
	fssrv.pclnt = pclnt
	fssrv.ch = make(chan bool)
	fssrv.fsl = fsl
	fssrv.stats.MonitorCPUUtil()
	return fssrv
}

func (fssrv *FsServer) SetFsl(fsl *fslib.FsLib) {
	fssrv.fsl = fsl
}

func (fssrv *FsServer) GetSessCondTable() *sesscond.SessCondTable {
	return fssrv.sct
}

func (fssrv *FsServer) GetRecentFences() *fences.RecentTable {
	return fssrv.rft
}

func (fssrv *FsServer) Root() fs.Dir {
	return fssrv.root
}

func (fssrv *FsServer) Snapshot() []byte {
	log.Printf("Snapshot %v", proc.GetPid())
	if !fssrv.replicated {
		log.Fatalf("FATAL: Tried to snapshot an unreplicated server %v", proc.GetName())
	}
	fssrv.snap = snapshot.MakeSnapshot(fssrv)
	return fssrv.snap.Snapshot(fssrv.root.(*dir.DirImpl), fssrv.st, fssrv.tmt, fssrv.rft, fssrv.rc)
}

func (fssrv *FsServer) Restore(b []byte) {
	if !fssrv.replicated {
		log.Fatal("FATAL: Tried to restore an unreplicated server %v", proc.GetName())
	}
	// Store snapshot for later use during restore.
	fssrv.snap = snapshot.MakeSnapshot(fssrv)
	// XXX How do we install the sct and wt? How do we sunset old state when
	// installing a snapshot on a running server?
	var root fs.FsObj
	root, fssrv.st, fssrv.tmt, fssrv.rft, fssrv.rc = fssrv.snap.Restore(fssrv.mkps, fssrv.rps, fssrv, fssrv.tmt.AddThread(), fssrv.process, fssrv.rc, b)
	fssrv.sct.St = fssrv.st
	fssrv.root = root.(fs.Dir)

}

func (fssrv *FsServer) Sess(sid np.Tsession) *session.Session {
	sess, ok := fssrv.st.Lookup(sid)
	if !ok {
		log.Fatalf("FATAL %v: no sess %v\n", proc.GetName(), sid)
		return nil
	}
	return sess
}

// The server using fssrv is ready to take requests. Keep serving
// until fssrv is told to stop using Done().
func (fssrv *FsServer) Serve() {
	// Non-intial-named services wait on the pclnt infrastructure. Initial named waits on the channel.
	if fssrv.pclnt != nil {
		if err := fssrv.pclnt.Started(proc.GetPid()); err != nil {
			debug.PrintStack()
			log.Printf("%v: Error Started: %v", proc.GetName(), err)
		}
		if err := fssrv.pclnt.WaitEvict(proc.GetPid()); err != nil {
			debug.PrintStack()
			log.Printf("%v: Error WaitEvict: %v", proc.GetName(), err)
		}
	} else {
		<-fssrv.ch
	}
}

// The server using fssrv is done; exit.
func (fssrv *FsServer) Done() {
	if fssrv.pclnt != nil {
		fssrv.pclnt.Exited(proc.GetPid(), proc.MakeStatus(proc.StatusEvicted))
	} else {
		if !fssrv.done {
			fssrv.done = true
			fssrv.ch <- true

		}
	}
	fssrv.stats.Done()
}

func (fssrv *FsServer) MyAddr() string {
	return fssrv.srv.MyAddr()
}

func (fssrv *FsServer) GetStats() *stats.Stats {
	return fssrv.stats
}

func (fssrv *FsServer) GetWatchTable() *watch.WatchTable {
	return fssrv.wt
}

func (fssrv *FsServer) GetSnapshotter() *snapshot.Snapshot {
	return fssrv.snap
}

func (fssrv *FsServer) AttachTree(uname string, aname string, sessid np.Tsession) (fs.Dir, fs.CtxI) {
	return fssrv.root, ctx.MkCtx(uname, sessid, fssrv.sct)
}

func (fssrv *FsServer) Process(fc *np.Fcall, replies chan *np.Fcall) {
	// The replies channel will be set here.
	sess := fssrv.st.Alloc(fc.Session, replies)
	// New thread about to start
	sess.IncThreads()
	if !fssrv.replicated {
		sess.GetThread().Process(fc)
	} else {
		fssrv.replSrv.Process(fc)
	}
}

func (fssrv *FsServer) sendReply(request *np.Fcall, reply np.Tmsg, replies chan *np.Fcall) {
	fcall := np.MakeFcall(reply, 0, nil)
	fcall.Session = request.Session
	fcall.Seqno = request.Seqno
	fcall.Tag = request.Tag
	// Store the reply in the reply cache if this is a replicated server.
	if fssrv.replicated {
		fssrv.rc.Put(request, fcall)
	}
	// The replies channel may be nil if this is a replicated op which came
	// through raft. In this case, a reply is not needed.
	if replies != nil {
		replies <- fcall
	}
}

func (fssrv *FsServer) process(fc *np.Fcall) {
	// If this is a replicated op received through raft (not directly from a
	// client), the first time Alloc is called will be in this function, so the
	// reply channel will be set to nil. If it came from the client, the reply
	// channel will already be set.
	sess := fssrv.st.Alloc(fc.Session, nil)
	if fssrv.replicated {
		// Reply cache needs to live under the replication layer in order to
		// handle duplicate requests. These may occur if, for example:
		//
		// 1. A client connects to replica A and issues a request.
		// 2. Replica A pushes the request through raft.
		// 3. Before responding to the client, replica A crashes.
		// 4. The client connects to replica B, and retries the request *before*
		//    replica B hears about the request through raft.
		// 5. Replica B pushes the request through raft.
		// 6. Replica B now receives the same request twice through raft's apply
		//    channel, and will try to execute the request twice.
		//
		// In order to handle this, we can use the reply cache to deduplicate
		// requests. Since requests execute sequentially, one of the requests will
		// register itself first in the reply cache. The other request then just
		// has to wait on the reply future in order to send the reply. This can
		// happen asynchronously since it doesn't affect server state, and the
		// asynchrony is necessary in order to allow other ops on the thread to
		// make progress. We coulld optionally use sessconds, but they're kind of
		// overkill since we don't care about ordering in this case.
		if replyFuture, ok := fssrv.rc.Get(fc); ok {
			go func() {
				fssrv.sendReply(fc, replyFuture.Await().GetMsg(), sess.GetRepliesC())
			}()
			return
		}
		// If this request has not been registered with the reply cache yet, register
		// it.
		fssrv.rc.Register(fc)
	}
	fssrv.stats.StatInfo().Inc(fc.Msg.Type())
	fssrv.serve(sess, fc)
}

func (fssrv *FsServer) serve(sess *session.Session, fc *np.Fcall) {
	reply, rerror := sess.Dispatch(fc.Msg)
	// We decrement the number of waiting threads if this request was made to
	// this server (it didn't come through raft), which will only be the case
	// when replies is not nil
	if sess.GetRepliesC() != nil {
		defer sess.DecThreads()
	}
	if rerror != nil {
		reply = *rerror
	}
	// Send reply will drop the reply if the replies channel is nil, but it will
	// make sure to insert the reply into the reply cache.
	fssrv.sendReply(fc, reply, sess.GetRepliesC())
}

func (fssrv *FsServer) CloseSession(sid np.Tsession, replies chan *np.Fcall) {
	sess, ok := fssrv.st.Lookup(sid)
	if !ok {
		// client start TCP connection, but then failed before sending
		// any messages.
		return
	}

	// XXX remove fence from sess, so that fence maybe free from seen table

	// Wait until nthread == 0. Detach is guaranteed to have been processed since
	// it was enqueued by the reader function before calling CloseSession
	// (incrementing nthread). We need to process Detaches (and sess cond closes)
	// through the session thread manager since they generate wakeups and need to
	// be properly serialized (especially for replication).
	sess.WaitThreads()

	// Stop sess thread.
	fssrv.st.KillSessThread(sid)
}
