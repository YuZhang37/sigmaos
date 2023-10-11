package main

import (
	"os"

	db "sigmaos/debug"
	"sigmaos/named"
)

func main() {
	if err := named.Run(os.Args); err != nil {
		db.DFatalf("%v: err %v\n", os.Args[0], err)
	}
}
