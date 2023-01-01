package container

import (
	"fmt"
	"os"

	db "sigmaos/debug"
)

//
// exec-container enters here
//

const (
	KERNEL = "KERNEL"
	PROC   = "PROC"
)

var envvar = []string{"SIGMADEBUG", "SIGMAPERF", "SIGMANAMED", "SIGMAROOTFS"}

func SIGMAROOTFS() (string, error) {
	fs := os.Getenv("SIGMAROOTFS")
	if fs == "" {
		return "", fmt.Errorf("%v: ExecContainer: SIGMAROOTFS isn't set; `run source env/init.sh`\n", os.Args[0])
	}
	return fs, nil
}

func MakeEnv() []string {
	env := []string{}
	for _, s := range envvar {
		if e := os.Getenv(s); e != "" {
			env = append(env, fmt.Sprintf("%s=%s", s, e))
		}
	}
	return env
}

func ExecContainer() error {
	db.DPrintf(db.CONTAINER, "execContainer: %v\n", os.Args)

	var r error
	switch os.Args[0] {
	case KERNEL:
		r = execKContainer()
	case PROC:
		r = execPContainer()
	default:
		r = fmt.Errorf("ExecContainer: unknown container type: %s", os.Args[0])
	}
	return r
}
