package main

import (
	"bufio"
	"log"
	"os"

	"ulambda/fslib"
	"ulambda/memfs"
	"ulambda/memfsd"
	np "ulambda/ninep"
	"ulambda/npsrv"
)

const (
	Stdin  = memfs.RootInum + 1
	Stdout = memfs.RootInum + 2
)

type Consoled struct {
	clnt   *fslib.FsLib
	srv    *npsrv.NpServer
	memfsd *memfsd.Fsd
	done   chan bool
}

func makeConsoled() *Consoled {
	cons := &Consoled{}
	cons.clnt = fslib.MakeFsLib(false)
	cons.memfsd = memfsd.MakeFsd(false)
	cons.srv = npsrv.MakeNpServer(cons.memfsd, ":0", false)
	cons.done = make(chan bool)
	return cons
}

type Console struct {
	stdin  *bufio.Reader
	stdout *bufio.Writer
}

func makeConsole() *Console {
	cons := &Console{}
	cons.stdin = bufio.NewReader(os.Stdin)
	cons.stdout = bufio.NewWriter(os.Stdout)
	return cons

}

func (cons *Console) Write(off np.Toffset, data []byte) (np.Tsize, error) {
	n, err := cons.stdout.Write(data)
	cons.stdout.Flush()
	return np.Tsize(n), err
}

func (cons *Console) Read(off np.Toffset, n np.Tsize) ([]byte, error) {
	b, err := cons.stdin.ReadBytes('\n')
	return b, err
}

func (cons *Console) Len() np.Tlength { return 0 }

func (cons *Consoled) FsInit() {
	fs := cons.memfsd.Root()
	_, err := fs.MkNod(fs.RootInode(), "console", makeConsole())
	if err != nil {
		log.Fatal("Create error: ", err)
	}
}

func main() {
	cons := makeConsoled()
	cons.FsInit()
	if fd, err := cons.clnt.Attach(":1111", ""); err == nil {
		err := cons.clnt.Mount(fd, "name")
		if err != nil {
			log.Fatal("Mount error: ", err)
		}
		err = cons.clnt.Remove("name/consoled")
		if err != nil {
			log.Print("name/consoled didn't exist")
		}
		name := cons.srv.MyAddr()
		err = cons.clnt.Symlink(name+":pubkey:console", "name/consoled", 0777)
		if err != nil {
			log.Fatal("Symlink error: ", err)
		}
	} else {
		log.Fatal("Attach error: ", err)
	}
	<-cons.done
	// cons.clnt.Close(fd)
	log.Printf("Consoled: finished\n")
}
