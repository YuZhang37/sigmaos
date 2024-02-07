package inode

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"sigmaos/fs"
	"sigmaos/serr"
	sp "sigmaos/sigmap"
)

type Inode struct {
	mu     sync.Mutex
	inum   sp.Tpath
	perm   sp.Tperm
	mtime  int64
	parent fs.Dir
	owner  *sp.Tprincipal
}

var NextInum = uint64(0)

func NewInode(ctx fs.CtxI, p sp.Tperm, parent fs.Dir) *Inode {
	i := &Inode{}
	i.inum = sp.Tpath(atomic.AddUint64(&NextInum, 1))
	i.perm = p
	i.mtime = time.Now().Unix()
	i.parent = parent
	if ctx == nil {
		i.owner = sp.NoPrincipal()
	} else {
		i.owner = ctx.Principal()
	}
	return i
}

func (inode *Inode) String() string {
	str := fmt.Sprintf("{ino %p %v %v}", inode, inode.inum, inode.perm)
	return str
}

func (inode *Inode) Path() sp.Tpath {
	return inode.inum
}

func (inode *Inode) Perm() sp.Tperm {
	return inode.perm
}

func (inode *Inode) Parent() fs.Dir {
	inode.mu.Lock()
	defer inode.mu.Unlock()
	return inode.parent
}

func (inode *Inode) SetParent(p fs.Dir) {
	inode.mu.Lock()
	defer inode.mu.Unlock()
	inode.parent = p
}

func (inode *Inode) Mtime() int64 {
	inode.mu.Lock()
	defer inode.mu.Unlock()
	return inode.mtime
}

func (inode *Inode) SetMtime(m int64) {
	inode.mu.Lock()
	defer inode.mu.Unlock()
	inode.mtime = m
}

func (i *Inode) Size() (sp.Tlength, *serr.Err) {
	return 0, nil
}

func (i *Inode) Open(ctx fs.CtxI, mode sp.Tmode) (fs.FsObj, *serr.Err) {
	return nil, nil
}

func (i *Inode) Close(ctx fs.CtxI, mode sp.Tmode) *serr.Err {
	return nil
}

func (i *Inode) Unlink() {
}

func (inode *Inode) Mode() sp.Tperm {
	perm := sp.Tperm(0777)
	if inode.perm.IsDir() {
		perm |= sp.DMDIR
	}
	return perm
}

func (inode *Inode) Stat(ctx fs.CtxI) (*sp.Stat, *serr.Err) {
	inode.mu.Lock()
	defer inode.mu.Unlock()

	st := sp.NewStat(sp.NewQidPerm(inode.perm, 0, inode.inum),
		inode.Mode(), uint32(inode.mtime), "", inode.owner.GetID().String())
	return st, nil
}
