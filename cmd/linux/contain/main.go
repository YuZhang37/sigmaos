package main

import (
	"log"
	"os"
	"os/exec"

	"sigmaos/container"
)

var defaultEnvironment = []string{
	"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	"TERM=xterm",
}

const (
	REALM = "test-realm"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("%s: Usage <bin> [args]\n", os.Args[0])
	}
	cmd := &exec.Cmd{
		Path: os.Args[1],
		Args: os.Args[1:],
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := container.MakeEnv()
	for _, s := range defaultEnvironment {
		env = append(env, s)
	}
	cmd.Env = env
	if err := container.RunKernelContainer(cmd, REALM); err != nil {
		log.Fatalf("%s: run container err %v\n", os.Args[0], err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatalf("%s: wait err %v\n", os.Args[0], err)
	}
}
