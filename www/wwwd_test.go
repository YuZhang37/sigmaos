package www_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	db "sigmaos/debug"
	"sigmaos/proc"
	rd "sigmaos/rand"
	sp "sigmaos/sigmap"
	"sigmaos/test"
	"sigmaos/www"
)

type Tstate struct {
	*test.Tstate
	*www.WWWClnt
	pid sp.Tpid
	job string
}

func spawn(t *testing.T, ts *Tstate) sp.Tpid {
	a := proc.NewProc("wwwd", []string{ts.job, ""})
	err := ts.Spawn(a)
	assert.Nil(t, err, "Spawn")
	return a.GetPid()
}

func newTstate(t *testing.T) *Tstate {
	var err error
	ts := &Tstate{}
	ts.Tstate = test.NewTstateAll(t)

	ts.Tstate.MkDir(www.TMP, 0777)

	ts.job = rd.String(4)

	www.InitWwwFs(ts.Tstate.FsLib, ts.job)

	ts.pid = spawn(t, ts)

	err = ts.WaitStart(ts.pid)
	assert.Nil(t, err)

	ts.WWWClnt = www.NewWWWClnt(ts.Tstate.FsLib, ts.job)

	return ts
}

func (ts *Tstate) waitWww() {
	err := ts.StopServer(ts.ProcClnt, ts.pid)
	assert.Nil(ts.T, err)
	ts.Shutdown()
}

func TestCompile(t *testing.T) {
}

func TestSandbox(t *testing.T) {
	ts := newTstate(t)
	ts.waitWww()
}

func TestStatic(t *testing.T) {
	ts := newTstate(t)

	out, err := ts.GetStatic("hello.html")
	assert.Nil(t, err)
	assert.Contains(t, string(out), "hello")

	out, err = ts.GetStatic("nonexist.html")
	assert.NotNil(t, err, "Out: %v", string(out)) // wget return error because of HTTP not found

	ts.waitWww()
}

func matmulClnt(ts *Tstate, matsize, clntid, nreq int, avgslp time.Duration, done chan bool) {
	clnt := www.NewWWWClnt(ts.Tstate.FsLib, ts.job)
	for i := 0; i < nreq; i++ {
		slp := avgslp * time.Duration(rd.Uint64()%100) / 100
		db.DPrintf(db.TEST, "[%v] iteration %v Random sleep %v", clntid, i, slp)
		time.Sleep(slp)
		err := clnt.MatMul(matsize)
		assert.Nil(ts.T, err, "Error matmul: %v", err)
	}
	db.DPrintf(db.TEST, "[%v] done", clntid)
	done <- true
}

const N = 100

func TestMatMul(t *testing.T) {
	ts := newTstate(t)

	done := make(chan bool)
	go matmulClnt(ts, N, 0, 1, 0, done)
	<-done

	ts.waitWww()
}

func TestMatMulConcurrent(t *testing.T) {
	ts := newTstate(t)

	N_CLNT := 5
	done := make(chan bool)
	for i := 0; i < N_CLNT; i++ {
		go matmulClnt(ts, N, i, 10, 500*time.Millisecond, done)
	}
	for i := 0; i < N_CLNT; i++ {
		<-done
	}

	ts.waitWww()
}
