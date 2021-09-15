package main

import (
	"log"
	"os"
	"strconv"

	"ulambda/fsclnt"
	"ulambda/replica"
)

func main() {
	if len(os.Args) < 6 {
		log.Fatalf("Usage: %v pid port config-path union-dir-path symlink-path", os.Args[0])
	}
	args := os.Args[1:]

	relayPort := args[1]
	portNum, err := strconv.Atoi(relayPort)
	if err != nil {
		log.Fatalf("Relay port must be an integer")
	}

	// Server port is relay port + 100
	srvPort := strconv.Itoa(100 + portNum)
	configPath := args[2]
	unionDirPath := args[3]
	symlinkPath := args[4]
	ip, err := fsclnt.LocalIP()
	if err != nil {
		log.Fatalf("%v: no IP %v\n", args, err)
	}
	relayAddr := ip + ":" + relayPort
	srvAddr := ip + ":" + srvPort

	// Get config
	config := replica.GetChainReplConfig("memfsd", relayPort, configPath, relayAddr, unionDirPath, symlinkPath)
	if len(args) == 6 && args[5] == "log-ops" {
		config.LogOps = true
	}

	// Start the replica server
	r := replica.MakeMemfsdReplica(args, srvAddr, unionDirPath, config)
	r.Work()
}
