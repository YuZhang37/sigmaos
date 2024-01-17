package lcschedsrv

import (
	"fmt"
	"sync"

	"sigmaos/proc"
	sp "sigmaos/sigmap"
)

const (
	DEF_Q_SZ = 10
)

type Qitem struct {
	p     *proc.Proc
	kidch chan string
}

func newQitem(p *proc.Proc, kidch chan string) *Qitem {
	return &Qitem{
		p:     p,
		kidch: kidch,
	}
}

type Queue struct {
	sync.Mutex
	procs []*Qitem
	pmap  map[sp.Tpid]*proc.Proc
}

func newQueue() *Queue {
	return &Queue{
		procs: make([]*Qitem, 0, DEF_Q_SZ),
		pmap:  make(map[sp.Tpid]*proc.Proc, 0),
	}
}

func (q *Queue) Enqueue(p *proc.Proc, kidch chan string) {
	q.Lock()
	defer q.Unlock()

	q.pmap[p.GetPid()] = p
	qi := newQitem(p, kidch)
	q.procs = append(q.procs, qi)
}

// Dequeue a proc with certain resource requirements.
func (q *Queue) Dequeue(mcpu proc.Tmcpu, mem proc.Tmem) (*proc.Proc, chan string, bool) {
	q.Lock()
	defer q.Unlock()

	for i := range q.procs {
		qi := q.procs[i]
		if qi.p.GetMem() <= mem && qi.p.GetMcpu() <= mcpu {
			q.procs = append(q.procs[:i], q.procs[i+1:]...)
			delete(q.pmap, qi.p.GetPid())
			return qi.p, qi.kidch, true
		}
	}
	return nil, nil, false
}

func (q *Queue) String() string {
	q.Lock()
	defer q.Unlock()

	return fmt.Sprintf("{ procs:%v }", q.procs)
}
