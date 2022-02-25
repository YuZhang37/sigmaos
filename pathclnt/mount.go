package pathclnt

import (
	"fmt"
	"log"
	"sync"

	np "ulambda/ninep"
)

type Point struct {
	path []string
	fid  np.Tfid
}

func (p *Point) String() string {
	return fmt.Sprintf("{%v, %v}", p.path, p.fid)
}

type MntTable struct {
	sync.Mutex
	mounts []*Point
	exited bool
}

func makeMntTable() *MntTable {
	mnt := &MntTable{}
	mnt.mounts = make([]*Point, 0)
	return mnt
}

// add path, in order of longest path first. if the path
// already exits, return error
func (mnt *MntTable) add(path []string, fid np.Tfid) *np.Err {
	mnt.Lock()
	defer mnt.Unlock()

	point := &Point{path, fid}
	for i, p := range mnt.mounts {
		if np.IsPathEq(path, p.path) {
			return np.MkErr(np.TErrExists, fmt.Sprintf("mount %v", p.path))
		}
		if len(path) > len(p.path) {
			mnts := append([]*Point{point}, mnt.mounts[i:]...)
			mnt.mounts = append(mnt.mounts[:i], mnts...)
			return nil
		}
	}
	mnt.mounts = append(mnt.mounts, point)
	return nil
}

// prefix match
func match(mp []string, path []string) (bool, []string) {
	rest := path
	for _, s := range mp {
		if len(rest) == 0 {
			return false, rest
		}
		if s == rest[0] {
			rest = rest[1:]
		} else {
			return false, rest
		}
	}
	return true, rest
}

func matchexact(mp []string, path []string) bool {
	if len(mp) != len(path) {
		return false
	}
	for i, s := range mp {
		if s != path[i] {
			return false
		}
	}
	return true
}

func (mnt *MntTable) exit() {
	mnt.Lock()
	defer mnt.Unlock()

	mnt.exited = true
}

// XXX Right now, we return EOF once we've "exited". Perhaps it makes more
// sense to return "unknown mount" or something along those lines.
func (mnt *MntTable) resolve(path []string) (np.Tfid, []string, *np.Err) {
	mnt.Lock()
	defer mnt.Unlock()

	if mnt.exited {
		return np.NoFid, path, np.MkErr(np.TErrEOF, path)
	}

	for _, p := range mnt.mounts {
		ok, rest := match(p.path, path)
		if ok {
			return p.fid, rest, nil
		}
	}
	return np.NoFid, path, np.MkErr(np.TErrNotfound, fmt.Sprintf("no mount %v\n", path))
}

func (mnt *MntTable) umount(path []string) (np.Tfid, *np.Err) {
	mnt.Lock()
	defer mnt.Unlock()

	for i, p := range mnt.mounts {
		ok := matchexact(p.path, path)
		if ok {
			mnt.mounts = append(mnt.mounts[:i], mnt.mounts[i+1:]...)
			return p.fid, nil
		}
	}
	return np.NoFid, np.MkErr(np.TErrNotfound, fmt.Sprintf("no mount %v\n", path))
}

func (mnt *MntTable) close() error {
	// Forbid any more (auto)mounting
	mnt.exit()

	// now iterate over mount points and umount them (without
	// holding mnt lock).  XXX do the actually work.
	for p := range mnt.mounts {
		log.Printf("mnt p %v\n", p)
	}
	return nil
}
