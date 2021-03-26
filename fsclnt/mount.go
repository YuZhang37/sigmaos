package fsclnt

import (
	"fmt"

	db "ulambda/debug"
	np "ulambda/ninep"
)

type Point struct {
	path []string
	fid  np.Tfid
}

func (p *Point) String() string {
	return fmt.Sprintf("{%v, %v}", p.path, p.fid)
}

type Mount struct {
	mounts []*Point
}

func makeMount() *Mount {
	return &Mount{make([]*Point, 0)}
}

// add path, in order of longest path first
func (mnt *Mount) add(path []string, fid np.Tfid) {
	point := &Point{path, fid}
	for i, p := range mnt.mounts {
		if len(path) > len(p.path) {
			mnts := append([]*Point{point}, mnt.mounts[i:]...)
			mnt.mounts = append(mnt.mounts[:i], mnts...)
			return
		}
	}
	mnt.mounts = append(mnt.mounts, point)
}

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

func (mnt *Mount) resolve(path []string) (np.Tfid, []string) {
	for _, p := range mnt.mounts {
		ok, rest := match(p.path, path)
		if ok {
			return p.fid, rest
		}
	}
	return np.NoFid, path
}

func (mnt *Mount) umount(path []string) (np.Tfid, error) {
	db.DLPrintf("FSCLNT", "Umount %v\n", path)
	for i, p := range mnt.mounts {
		ok, _ := match(p.path, path)
		if ok {
			mnt.mounts = append(mnt.mounts[:i], mnt.mounts[i+1:]...)
			return p.fid, nil
		}
	}
	return np.NoFid, fmt.Errorf("umount: unknown mount %v\n", path)
}
