package container

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	db "sigmaos/debug"
	"sigmaos/linuxsched"
	"sigmaos/proc"
)

//
// Contain user procs using exec-uproc trampoline
//

func RunUProc(uproc *proc.Proc) error {
	db.DPrintf(db.CONTAINER, "RunUProc %v env %v\n", uproc, os.Environ())
	var cmd *exec.Cmd
	straceProcs := proc.GetLabels(uproc.GetProcEnv().GetStrace())
	// Optionally strace the proc
	if straceProcs[uproc.GetProgram()] {
		cmd = exec.Command("strace", append([]string{"-f", "exec-uproc-rs", uproc.GetPid().String(), uproc.GetProgram()}, uproc.Args...)...)
	} else {
		cmd = exec.Command("exec-uproc-rs", append([]string{uproc.GetPid().String(), uproc.GetProgram()}, uproc.Args...)...)
	}
	uproc.AppendEnv("PATH", "/bin:/bin2:/usr/bin:/home/sigmaos/bin/kernel")
	uproc.AppendEnv("SIGMA_EXEC_TIME", strconv.FormatInt(time.Now().UnixMicro(), 10))
	uproc.AppendEnv(proc.SIGMAPERF, uproc.GetProcEnv().GetPerf())
	uproc.AppendEnv("RUST_BACKTRACE", "1")
	cmd.Env = uproc.GetEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Set up new namespaces
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}
	db.DPrintf(db.CONTAINER, "exec cmd %v", cmd)
	defer cleanupJail(uproc.GetPid())
	s := time.Now()
	if err := cmd.Start(); err != nil {
		db.DPrintf(db.CONTAINER, "Error start %v %v", cmd, err)
		return err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc cmd.Start %v", uproc.GetPid(), time.Since(s))
	if uproc.GetType() == proc.T_BE {
		s := time.Now()
		setSchedPolicy(cmd.Process.Pid, linuxsched.SCHED_IDLE)
		db.DPrintf(db.SPAWN_LAT, "[%v] Uproc Get/Set sched attr %v", uproc.GetPid(), time.Since(s))
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	db.DPrintf(db.CONTAINER, "ExecUProc done  %v\n", uproc)
	return nil
}

func setSchedPolicy(pid int, policy linuxsched.SchedPolicy) {
	attr, err := linuxsched.SchedGetAttr(pid)
	if err != nil {
		db.DPrintf(db.ALWAYS, "Error Getattr %v: %v", pid, err)
		return
	}
	attr.Policy = policy
	err = linuxsched.SchedSetAttr(pid, attr)
	if err != nil {
		db.DPrintf(db.ALWAYS, "Error Setattr %v: %v", pid, err)
	}
}

//
// The exec-uproc trampoline enters here
//

func ExecUProc() error {
	db.DPrintf(db.CONTAINER, "ExecUProc: %v\n", os.Args)
	args := os.Args[1:]
	program := args[0]
	s := time.Now()
	pcfg := proc.GetProcEnv()
	// Isolate the user proc.
	pn, err := isolateUserProc(pcfg.GetPID(), program)
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc isolation %v", pcfg.GetPID(), time.Since(s))
	if err != nil {
		return err
	}
	os.Setenv("SIGMA_EXEC_TIME", strconv.FormatInt(time.Now().UnixMicro(), 10))
	db.DPrintf(db.CONTAINER, "exec %v %v", pn, args)
	if err := syscall.Exec(pn, args, os.Environ()); err != nil {
		db.DPrintf(db.CONTAINER, "Error exec %v", err)
		return err
	}
	defer finishIsolation()
	return nil
}

// For debugging
func ls(dir string) error {
	db.DPrintf(db.ALWAYS, "== ls %s\n", dir)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		db.DPrintf(db.ALWAYS, "ls err %v", err)
		return nil
	}
	for _, file := range files {
		db.DPrintf(db.ALWAYS, "file %v isdir %v", file.Name(), file.IsDir())
	}
	return nil
}
