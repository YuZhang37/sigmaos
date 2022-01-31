package session

import (
	"log"
	"sync"

	//	"github.com/sasha-s/go-deadlock"

	// db "ulambda/debug"
	"ulambda/fence"
	"ulambda/fslib"
	np "ulambda/ninep"
	"ulambda/protsrv"
)

//
// A session identifies a client across TCP connections.  For each
// session, sigmaos has a protsrv.
//
// The sess lock is to serialize requests on a session.  The calls in
// this file assume the calling threads holds the sess lock.
//

type Session struct {
	sync.Mutex // to serialize requests on a session
	cond       *sync.Cond
	threads    sync.WaitGroup
	protsrv    protsrv.Protsrv
	seenFences *fence.FenceTable
	myFences   *fence.FenceTable
	Sid        np.Tsession
}

func makeSession(protsrv protsrv.Protsrv, sid np.Tsession, fm *fence.FenceTable) *Session {
	sess := &Session{}
	sess.protsrv = protsrv
	sess.cond = sync.NewCond(&sess.Mutex)
	sess.Sid = sid
	sess.seenFences = fm
	sess.myFences = fence.MakeFenceTable()
	return sess
}

func (sess *Session) Fence(req np.Tregfence) error {
	sess.myFences.Insert(req.Fence)
	return nil
}

func (sess *Session) Unfence(idf np.Tfenceid) error {
	return sess.myFences.Del(idf)
}

func (sess *Session) CheckFences(fsl *fslib.FsLib) error {
	fences := sess.myFences.Fences()
	if len(fences) > 0 {
		log.Printf("%v: CheckFences %v\n", sess.Sid, fences)
	}
	for _, f := range fences {
		err := sess.seenFences.IsRecent(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sess *Session) IncThreads() {
	sess.threads.Add(1)
}

func (sess *Session) DecThreads() {
	sess.threads.Done()
}

func (sess *Session) WaitThreads() {
	sess.threads.Wait()
}
