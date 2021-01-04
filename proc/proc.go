package proc

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	"ulambda/fsclnt"
	np "ulambda/ninep"
)

func randPid(clnt *fsclnt.FsClient) string {
	pid := rand.Int()
	return strconv.Itoa(pid)
}

func makeAttr(clnt *fsclnt.FsClient, fddir int, key string, value []byte) error {
	fd, err := clnt.CreateAt(fddir, key, 0, np.OWRITE)
	if err != nil {
		return err
	}
	defer clnt.Close(fd)
	_, err = clnt.Write(fd, 0, []byte(value))
	return err
}

func setAttr(clnt *fsclnt.FsClient, fddir int, key string, value []byte) error {
	fd, err := clnt.OpenAt(fddir, key, np.OWRITE)
	if err != nil {
		return err
	}
	defer clnt.Close(fd)
	_, err = clnt.Write(fd, 0, []byte(value))
	return err
}

// XXX clean up on failure
// XXX maybe one RPC?
func Spawn(clnt *fsclnt.FsClient, program string, args []string, fids []string) (string, error) {
	pid := randPid(clnt)
	pname := "name/procd/" + pid

	fd, err := clnt.Mkdir(pname, np.ORDWR)
	if err != nil {
		return "", err
	}
	defer clnt.Close(fd)
	err = makeAttr(clnt, fd, "program", []byte(program))
	if err != nil {
		//clnt.RmDir(pname)
		return "", err
	}

	args = append([]string{pname}, args...)
	b, err := json.Marshal(args)
	if err != nil {
		//clnt.RmDir(pname)
		return "", err
	}
	err = makeAttr(clnt, fd, "args", b)
	if err != nil {
		//clnt.RmDir(pname)
		return "", err
	}

	b, err = json.Marshal(fids)
	if err != nil {
		//clnt.RmDir(pname)
		return "", err
	}
	err = makeAttr(clnt, fd, "fds", b)
	if err != nil {
		//clnt.RmDir(pname)
		return "", err
	}
	err = clnt.Pipe(clnt.Proc+"/"+"exit"+pid, np.ORDWR)
	if err != nil {
		//clnt.RmDir(pname)
		return "", err
	}
	err = clnt.SymlinkAt(fd, clnt.Proc, "parent")
	if err != nil {
		//clnt.RmDir(clnt.Proc + "/" + "exit" + pid)
		//clnt.RmDir(pname)
		return "", err
	}

	fd1, err := clnt.Open("name/procd/makeproc", np.OWRITE)
	if err != nil {
		//clnt.RmDir(clnt.Proc + "/" + "exit" + pid)
		//clnt.RmDir(pname)
		return "", err
	}
	defer clnt.Close(fd1)
	_, err = clnt.Write(fd1, 0, []byte("Start "+pid))
	if err != nil {
		//clnt.RmDir(clnt.Proc + "/" + "exit" + pid)
		//clnt.RmDir(pname)
		return "", err
	}
	return pname, err
}

func Exit(clnt *fsclnt.FsClient, v ...interface{}) error {
	log.Printf("Exit %v\n", clnt.Proc)
	pid := filepath.Base(clnt.Proc)
	defer func() {
		//clnt.RmDir(clnt.Proc)
		os.Exit(0)
	}()
	fd, err := clnt.Open(clnt.Proc+"/parent/exit"+pid, np.OWRITE)
	if err != nil {
		return err
	}
	defer clnt.Close(fd)
	_, err = clnt.Write(fd, 0, []byte(fmt.Sprint(v...)))
	return err
}

func Wait(clnt *fsclnt.FsClient, child string) ([]byte, error) {
	log.Printf("Wait %v\n", child)
	pid := filepath.Base(child)
	fd, err := clnt.Open(clnt.Proc+"/exit"+pid, np.OREAD)
	if err != nil {
		return nil, err
	}
	// defer clnt.Remove(clnt.Proc + "/exit" + pid)
	defer clnt.Close(fd)
	b, err := clnt.Read(fd, 0, 1024)
	return b, err
}

func Getpid(clnt *fsclnt.FsClient) (string, error) {
	return "name/procd/" + clnt.Proc, nil
}
