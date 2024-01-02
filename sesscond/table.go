package sesscond

import (
	"sync"

	db "sigmaos/debug"
	"sigmaos/sessstatesrv"
	sp "sigmaos/sigmap"
)

type SessCondTable struct {
	//	deadlock.Mutex
	sync.Mutex
	conds  map[*SessCond]bool
	St     *sessstatesrv.SessionTable
	closed bool
}

func NewSessCondTable(st *sessstatesrv.SessionTable) *SessCondTable {
	t := &SessCondTable{}
	t.conds = make(map[*SessCond]bool)
	t.St = st
	return t
}

func (sct *SessCondTable) NewSessCond(lock sync.Locker) *SessCond {
	sct.Lock()
	defer sct.Unlock()

	sc := newSessCond(sct, lock)
	sct.conds[sc] = true
	sc.nref++
	return sc
}

func (sct *SessCondTable) FreeSessCond(sc *SessCond) {
	sct.Lock()
	defer sct.Unlock()
	sc.nref--
	if sc.nref != 0 {
		db.DFatalf("freesesscond %v\n", sc)
	}
	delete(sct.conds, sc)
}

func (sct *SessCondTable) toSlice() []*SessCond {
	sct.Lock()
	defer sct.Unlock()

	sct.closed = true
	t := make([]*SessCond, 0, len(sct.conds))
	for sc, _ := range sct.conds {
		t = append(t, sc)
	}
	return t
}

// Close all sess conds for sessid, which wakes up waiting threads.  A
// thread may delete a sess cond from sct, if the thread is the last
// user.  So we need, a lock around sct.conds.  But, DeleteSess
// violates lock order, which is lock sc.lock first (e.g., watch on
// directory), then acquire sct.lock (if file watch must create sess
// cond in sct).  To avoid order violation, DeleteSess news copy
// first, then close() sess conds.  Threads many add new sess conds to
// sct while bailing out (e.g., to remove an emphemeral file), but
// threads shouldn't wait on these sess conds, so we don't have to
// close those.
func (sct *SessCondTable) DeleteClnt(cid sp.TclntId) {
	t := sct.toSlice()
	db.DPrintf(db.ALWAYS, "DeleteClnt cid %v %v\n", cid, t)
	for _, sc := range t {
		sc.closed(cid)
	}
}
