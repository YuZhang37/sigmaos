package main

import (
	"fmt"
	"log"
	"os"

	"ulambda/crash"
	db "ulambda/debug"
	"ulambda/delay"
	"ulambda/fslib"
	np "ulambda/ninep"
	"ulambda/proc"
	"ulambda/procinit"
	usync "ulambda/sync"
)

const (
	N   = 1000
	DIR = "name/locktest"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "%v: Usage: <partition?> <dir>\n", os.Args[0])
		os.Exit(1)
	}
	fsl := fslib.MakeFsLib("locker-" + proc.GetPid())

	pclnt := procinit.MakeProcClnt(fsl, procinit.GetProcLayersMap())

	fsl.Mkdir(DIR, 0777)

	lock := usync.MakeLock(fsl, DIR, "lock", true)

	delay.SetDelay(100)

	pclnt.Started(proc.GetPid())

	for i := 0; i < N; i++ {
		lock.Lock()
		fn := os.Args[2] + "/" + proc.GetPid()
		if err := fsl.MakeFile(fn, 0777, np.OWRITE, []byte{}); err != nil {
			log.Fatalf("makefile %v failed %v\n", fn, err)
		}

		if os.Args[1] == "YES" {
			crash.MaybePartition(fsl)
			delay.Delay()
		}

		if sts, err := fsl.ReadDir(os.Args[2]); err != nil {
			log.Fatalf("readdir %v failed %v\n", os.Args[2], err)
		} else {
			if len(sts) == 2 {
				log.Printf("%v: len 2 %v\n", db.GetName(), sts)
				pclnt.Exited(proc.GetPid(), "Two holders")
				os.Exit(0)
			}
		}
		if err := fsl.Remove(fn); err != nil {
			log.Fatalf("remove %v failed %v\n", fn, err)
		}
		lock.Unlock()
	}
	pclnt.Exited(proc.GetPid(), "OK")
}
