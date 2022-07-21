package mr

import (
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"ulambda/crash"
	db "ulambda/debug"
	"ulambda/electclnt"
	"ulambda/fslib"
	np "ulambda/ninep"
	"ulambda/proc"
	"ulambda/procclnt"
	"ulambda/procdclnt"
)

const (
	MR       = "/mr/"
	MRDIRTOP = "name/" + MR

	TIP  = "-tip/"
	DONE = "-done/"
	NEXT = "-next/"

	NCOORD = 3

	MLOCALSRV = np.UX + "/~ip" // must end without /
	MLOCALDIR = MLOCALSRV + MR

	RESTART = "restart" // restart message from reducer
)

// XXX update
//
// mr_test puts names of input files in MDIR and launches a
// coordinator proc to run the MR job.  The actual input files live in
// MIN.
//
// The coordinator creates one thread per input file, which looks for
// a file name in MDIR. If thread finds a name, it claims it by
// renaming it into MDIR+TIP to record that a task for name is in
// progress.  Then, the thread creates a mapper proc (task) to process
// the input file.  Mapper i creates <r> output shards, one for each
// reducer.  Once the mapper completes an output shard, it creates a
// symlink in dir RIN+/r, which contains the pathname for the output
// shard.  If the mapper proc successfully exits, the thread renames
// the task from MDIR+TIP into the dir MDIR+DONE, to record that this
// mapper task has completed.
//
// The coordinator creates one thread per reducer, which grabs <r>
// from RDIR, and records in RDIR+TIP that reducer <r> is in progress.
// The thread creates a reducer proc that looks in dir RIN+/r for
// symlinks to process (one symlink per mapper task). The symlinks
// contain the pathname where the mapper puts its shard for this
// reducer.  The reducer writes it output to ROUT+<r>.  If the reducer
// task exits successfully, the thread renames RDIR+TIP+r into
// RDIR+DONE, to record that this reducer task has completed.
//

type Coord struct {
	*fslib.FsLib
	*procclnt.ProcClnt
	job         string
	nmaptask    int
	nreducetask int
	crash       int
	linesz      string
	mapperbin   string
	reducerbin  string
	electclnt   *electclnt.ElectClnt
	done        int32
}

func MakeCoord(args []string) (*Coord, error) {
	if len(args) != 7 {
		return nil, errors.New("MakeCoord: wrong number of arguments")
	}
	c := &Coord{}
	c.FsLib = fslib.MakeFsLib("coord-" + proc.GetPid().String())

	c.job = args[0]
	m, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, fmt.Errorf("MakeCoord: nmaptask %v isn't int", args[1])
	}
	n, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, fmt.Errorf("MakeCoord: nreducetask %v isn't int", args[2])
	}
	c.nmaptask = m
	c.nreducetask = n
	c.mapperbin = args[3]
	c.reducerbin = args[4]

	ctime, err := strconv.Atoi(args[5])
	if err != nil {
		return nil, fmt.Errorf("MakeCoord: crash %v isn't int", args[5])
	}
	c.crash = ctime

	c.linesz = args[6]

	c.ProcClnt = procclnt.MakeProcClnt(c.FsLib)

	c.Started()

	c.electclnt = electclnt.MakeElectClnt(c.FsLib, JobDir(c.job)+"/coord-leader", 0)

	crash.Crasher(c.FsLib)

	return c, nil
}

func (c *Coord) makeTask(bin string, args []string) *proc.Proc {
	p := proc.MakeProc(bin, args)
	if c.crash > 0 {
		p.AppendEnv("SIGMACRASH", strconv.Itoa(c.crash))
	}
	return p
}

func (c *Coord) mapperProc(task string) *proc.Proc {
	input := MapTask(c.job) + TIP + task
	return c.makeTask(c.mapperbin, []string{c.job, strconv.Itoa(c.nreducetask), input, c.linesz})
}

func (c *Coord) reducerProc(task string) *proc.Proc {
	in := ReduceIn(c.job) + "/" + task
	out := ReduceOut(c.job) + task
	return c.makeTask(c.reducerbin, []string{in, out, strconv.Itoa(c.nmaptask)})
}

func (c *Coord) claimEntry(dir string, st *np.Stat) (string, error) {
	from := dir + "/" + st.Name
	if err := c.Rename(from, dir+TIP+"/"+st.Name); err != nil {
		if np.IsErrUnreachable(err) { // partitioned?
			return "", err
		}
		// another thread claimed the task before us
		return "", nil
	}
	return st.Name, nil
}

func (c *Coord) getTask(dir string) (string, error) {
	t := ""
	stopped, err := c.ProcessDir(dir, func(st *np.Stat) (bool, error) {
		t1, err := c.claimEntry(dir, st)
		if err != nil {
			return false, err
		}
		if t1 == "" {
			return false, nil
		}
		t = t1
		return true, nil

	})
	if err != nil {
		return "", err
	}
	if stopped {
		return t, nil
	}
	return "", nil
}

type Tresult struct {
	t   string
	ok  bool
	ms  int64
	msg string
	res *Result
}

func (c *Coord) doneTasks(dir string) int {
	sts, err := c.GetDir(dir)
	if err != nil {
		db.DFatalf("doneTasks err %v\n", err)
	}
	return len(sts)
}

func (c *Coord) waitForTask(start time.Time, ch chan Tresult, dir string, p *proc.Proc, t string) {
	// Wait for the task to exit.
	status, err := c.WaitExit(p.Pid)
	// Record end time.
	ms := time.Since(start).Milliseconds()
	if err == nil && status.IsStatusOK() {
		// mark task as done
		s := dir + TIP + "/" + t
		d := dir + DONE + "/" + t
		if err := c.Rename(s, d); err != nil {
			db.DFatalf("rename task done %v to %v err %v\n", s, d, err)
		}
		r := mkResult(status.Data())
		ch <- Tresult{t, true, ms, status.Msg(), r}
	} else { // task failed; make it runnable again
		if status != nil && status.Msg() == RESTART {
			// reducer indicates to run some mappers again
			s := mkStringSlice(status.Data().([]interface{}))
			c.restart(s, t)
		} else { // if failure but not restart, rerun task immediately again
			to := dir + "/" + t
			db.DPrintf("MR", "task %v failed %v err %v\n", t, status, err)
			if err := c.Rename(dir+TIP+"/"+t, to); err != nil {
				db.DFatalf("rename to runnable %v err %v\n", to, err)
			}
		}
		ch <- Tresult{t, false, ms, "", nil}
	}
}

func (c *Coord) runTasks(ch chan Tresult, dir string, taskNames []string, f func(string) *proc.Proc) {
	tasks := make([]*proc.Proc, len(taskNames))
	// Make task proc structures.
	for i, tn := range taskNames {
		tasks[i] = f(tn)
		db.DPrintf("MR", "prep to burst-spawn task %v %v\n", tasks[i].Pid, tasks[i].Args)
	}
	start := time.Now()
	// Burst-spawn procs.
	failed, errs := c.SpawnBurst(tasks)
	if len(failed) > 0 {
		db.DFatalf("Couldn't burst-spawn some tasks %v, errs: %v", failed, errs)
	}
	for i := range tasks {
		go c.waitForTask(start, ch, dir, tasks[i], taskNames[i])
	}
}

func mkStringSlice(data []interface{}) []string {
	s := make([]string, 0, len(data))
	for _, o := range data {
		s = append(s, o.(string))
	}
	return s
}

func (c *Coord) startTasks(ch chan Tresult, dir string, f func(string) *proc.Proc) int {
	taskNames := []string{}
	for {
		t, err := c.getTask(dir)
		if err != nil {
			db.DFatalf("getTask %v err %v\n", dir, err)
		}
		if t == "" {
			break
		}
		taskNames = append(taskNames, t)
	}
	go c.runTasks(ch, dir, taskNames, f)
	return len(taskNames)
}

// A reducer failed because it couldn't read its input file; we must
// restart mapper.  We let all mappers and reducers finish, before
// starting a new round for the mappers and reducers that must be
// restarted, which avoids restarting a mapper several times (because
// several reducers may ask the mapper to be restarted).
func (c *Coord) restart(files []string, task string) {
	db.DPrintf(db.ALWAYS, "restart %v and %v\n", files, task)
	for _, f := range files {
		// Remove symfile so that when coordinator restarts
		// reducers, they wait for the mappers to make new
		// symfiles.
		sym := symname(c.job, task, f)
		if err := c.Remove(sym); err != nil {
			db.DPrintf(db.ALWAYS, "remove %v err %v\n", sym, err)
		}
		// Record that we have to rerun mapper f.  Mapper may
		// still be in progress, is done, or already has been
		// moved to d.
		s := MapTask(c.job) + DONE + "/" + f
		s1 := MapTask(c.job) + TIP + "/" + f
		d := MapTask(c.job) + NEXT + "/" + f
		if err := c.Rename(s1, d); err != nil {
			db.DPrintf(db.ALWAYS, "rename next  %v to %v err %v\n", s, d, err)
		}
		if err := c.Rename(s, d); err != nil {
			db.DPrintf(db.ALWAYS, "rename next  %v to %v err %v\n", s, d, err)
		}
	}
	// Record that we have to rerun reducer t
	n := ReduceTask(c.job) + NEXT + "/" + task
	if _, err := c.PutFile(n, 0777, np.OWRITE, []byte(n)); err != nil {
		db.DPrintf(db.ALWAYS, "PutFile %v err %v\n", n, err)
	}
}

// Mark all restart tasks as runnable
func (c *Coord) doRestart() bool {
	n, err := c.MoveFiles(MapTask(c.job)+NEXT, MapTask(c.job))
	if err != nil {
		db.DFatalf("MoveFiles %v err %v\n", MapTask(c.job), err)
	}
	m, err := c.MoveFiles(ReduceTask(c.job)+NEXT, ReduceTask(c.job))
	if err != nil {
		db.DFatalf("MoveFiles %v err %v\n", ReduceTask(c.job), err)
	}
	return n+m > 0
}

// Consider all tasks in progress as failed (too aggressive, but
// correct), and make them runnable
func (c *Coord) recover(dir string) {
	if _, err := c.MoveFiles(dir+TIP, dir); err != nil {
		db.DFatalf("MoveFiles %v err %v\n", dir, err)
	}
}

// XXX do something for stragglers?
func (c *Coord) Round() {
	ch := make(chan Tresult)
	for m := 0; ; m-- {
		m += c.startTasks(ch, MapTask(c.job), c.mapperProc)
		m += c.startTasks(ch, ReduceTask(c.job), c.reducerProc)
		if m <= 0 {
			break
		}
		res := <-ch
		db.DPrintf("MR", "%v ok %v ms %d msg %v res %v\n", res.t, res.ok, res.ms, res.msg, res.res)
		if res.ok {
			if err := c.AppendFileJson(MRstats(c.job), res.res); err != nil {
				db.DFatalf("Appendfile %v err %v\n", MRstats(c.job), err)
			}
			db.DPrintf(db.ALWAYS, "tasks left %d/%d\n", m-1, c.nmaptask+c.nreducetask)
		}
	}
}

func (c *Coord) monitorProcds() {
	pdc := procdclnt.MakeProcdClnt(c.FsLib)
	n := 0
	for atomic.LoadInt32(&c.done) == 0 {
		m, err := pdc.WaitProcdChange(n)
		if err != nil && atomic.LoadInt32(&c.done) == 0 {
			db.DFatalf("WaitProcdChange err %v\n", err)
		}

		db.DPrintf(db.ALWAYS, "nprocd = %d\n", m)
		n = m
	}
}

func (c *Coord) monitorProcs() {
	pdc := procdclnt.MakeProcdClnt(c.FsLib)
	for atomic.LoadInt32(&c.done) == 0 {
		n, load, err := pdc.Nprocd()
		if err != nil && atomic.LoadInt32(&c.done) == 0 {
			db.DFatalf("Nprocd err %v\n", err)
		}
		db.DPrintf(db.ALWAYS, "nprocd = %d %v\n", n, load)
		time.Sleep(10 * time.Second)
	}
}

func (c *Coord) Work() {
	// Try to become the leading coordinator.  If we get
	// partitioned, we cannot write the todo directories either,
	// so need to set a fence.
	c.electclnt.AcquireLeadership(nil)

	db.DPrintf(db.ALWAYS, "leader %s nmap %v nreduce %v\n", c.job, c.nmaptask, c.nreducetask)

	go c.monitorProcds()
	go c.monitorProcs()

	c.recover(MapTask(c.job))
	c.recover(ReduceTask(c.job))
	c.doRestart()

	for n := 0; ; {
		db.DPrintf(db.ALWAYS, "run round %d\n", n)
		c.Round()
		if !c.doRestart() {
			break
		}
	}

	// double check we are done
	n := c.doneTasks(MapTask(c.job) + DONE)
	n += c.doneTasks(ReduceTask(c.job) + DONE)
	if n < c.nmaptask+c.nreducetask {
		db.DFatalf("job isn't done %v", n)
	}

	db.DPrintf(db.ALWAYS, "job done\n")

	atomic.StoreInt32(&c.done, 1)

	c.Exited(proc.MakeStatus(proc.StatusOK))
}
