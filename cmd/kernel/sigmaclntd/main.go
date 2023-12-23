package main

import (
	"os"

	db "sigmaos/debug"
	"sigmaos/sigmaclntsrv"
)

func main() {
	if len(os.Args) != 1 {
		db.DFatalf("Usage: %v", os.Args[0])
	}
	if err := sigmaclntsrv.RunSigmaClntSrv(os.Args); err != nil {
		db.DFatalf("Fatal start: %v %v\n", os.Args[0], err)
	}
}
