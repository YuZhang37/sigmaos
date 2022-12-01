package pathclnt

import (
	"fmt"
	"strings"

	db "sigmaos/debug"
	"sigmaos/fcall"
	"sigmaos/fidclnt"
	"sigmaos/path"
	"sigmaos/reader"
	np "sigmaos/sigmap"
	"sigmaos/writer"
)

//
// The Sigma file system API at the level of pathnames.  The
// pathname-based operations are implemented here and support dynamic
// mounts, sfs-like pathnames, etc. All fid-based operations are
// inherited from FidClnt.
//

type Watch func(string, error)

type PathClnt struct {
	*fidclnt.FidClnt
	mnt     *MntTable
	chunkSz np.Tsize
}

func MakePathClnt(fidc *fidclnt.FidClnt, sz np.Tsize) *PathClnt {
	pathc := &PathClnt{}
	pathc.mnt = makeMntTable()
	pathc.chunkSz = sz
	if fidc == nil {
		pathc.FidClnt = fidclnt.MakeFidClnt()
	} else {
		pathc.FidClnt = fidc
	}
	return pathc
}

func (pathc *PathClnt) String() string {
	str := fmt.Sprintf("Pathclnt mount table:\n")
	str += fmt.Sprintf("%v\n", pathc.mnt)
	return str
}

func (pathc *PathClnt) SetChunkSz(sz np.Tsize) {
	pathc.chunkSz = sz
}

func (pathc *PathClnt) GetChunkSz() np.Tsize {
	return pathc.chunkSz
}

func (pathc *PathClnt) Mounts() []string {
	return pathc.mnt.mountedPaths()
}

// Exit the path client, closing all sessions
func (pathc *PathClnt) Exit() error {
	return pathc.FidClnt.Exit()
}

// Detach from server. XXX Mixes up umount a file system at server and
// closing session; if two mounts point to the same server; the first
// detach will close the session regardless of the second mount point.
func (pathc *PathClnt) Detach(pn string) error {
	fid, err := pathc.mnt.umount(path.Split(pn))
	if err != nil {
		return err
	}
	defer pathc.FidClnt.Free(fid)
	if err := pathc.FidClnt.Detach(fid); err != nil {
		return err
	}
	return nil
}

// Simulate network partition to server that exports path
func (pathc *PathClnt) Disconnect(pn string) error {
	fid, err := pathc.mnt.umount(path.Split(pn))
	if err != nil {
		return err
	}
	defer pathc.FidClnt.Free(fid)
	if err := pathc.FidClnt.Lookup(fid).Disconnect(); err != nil {
		return err
	}
	return nil
}

func (pathc *PathClnt) MakeReader(fid np.Tfid, path string, chunksz np.Tsize) *reader.Reader {
	return reader.MakeReader(pathc.FidClnt, path, fid, chunksz)
}

func (pathc *PathClnt) MakeWriter(fid np.Tfid) *writer.Writer {
	return writer.MakeWriter(pathc.FidClnt, fid)
}

func (pathc *PathClnt) readlink(fid np.Tfid) (string, *fcall.Err) {
	qid := pathc.Qid(fid)
	if qid.Type&np.QTSYMLINK == 0 {
		return "", fcall.MkErr(fcall.TErrNotSymlink, qid.Type)
	}
	_, err := pathc.FidClnt.Open(fid, np.OREAD)
	if err != nil {
		return "", err
	}
	rdr := reader.MakeReader(pathc.FidClnt, "", fid, pathc.chunkSz)
	b, err := rdr.GetDataErr()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (pathc *PathClnt) mount(fid np.Tfid, pn string) *fcall.Err {
	if err := pathc.mnt.add(path.Split(pn), fid); err != nil {
		if err.Code() == fcall.TErrExists {
			// Another thread may already have mounted
			// path; clunk the fid and don't return an
			// error.
			pathc.Clunk(fid)
			return nil
		} else {
			return err
		}
	}
	return nil
}

func (pathc *PathClnt) Mount(fid np.Tfid, path string) error {
	if err := pathc.mount(fid, path); err != nil {
		return err
	}
	return nil
}

func (pathc *PathClnt) Create(p string, perm np.Tperm, mode np.Tmode) (np.Tfid, error) {
	db.DPrintf("PATHCLNT", "Create %v perm %v\n", p, perm)
	path := path.Split(p)
	dir := path.Dir()
	base := path.Base()
	fid, err := pathc.WalkPath(dir, true, nil)
	if err != nil {
		return np.NoFid, err
	}
	fid, err = pathc.FidClnt.Create(fid, base, perm, mode)
	if err != nil {
		return np.NoFid, err
	}
	return fid, nil
}

// Rename using renameat() for across directories or using wstat()
// for within a directory.
func (pathc *PathClnt) Rename(old string, new string) error {
	db.DPrintf("PATHCLNT", "Rename %v %v\n", old, new)
	opath := path.Split(old)
	npath := path.Split(new)

	if len(opath) != len(npath) {
		if err := pathc.renameat(old, new); err != nil {
			return err
		}
		return nil
	}
	for i, n := range opath[:len(opath)-1] {
		if npath[i] != n {
			if err := pathc.renameat(old, new); err != nil {
				return err
			}
			return nil
		}
	}
	fid, err := pathc.WalkPath(opath, path.EndSlash(old), nil)
	if err != nil {
		return err
	}
	defer pathc.FidClnt.Clunk(fid)
	st := &np.Stat{}
	st.Name = npath[len(npath)-1]
	err = pathc.FidClnt.Wstat(fid, st)
	if err != nil {
		return err
	}
	return nil
}

// Rename across directories of a single server using Renameat
func (pathc *PathClnt) renameat(old, new string) *fcall.Err {
	db.DPrintf("PATHCLNT", "Renameat %v %v\n", old, new)
	opath := path.Split(old)
	npath := path.Split(new)
	o := opath[len(opath)-1]
	n := npath[len(npath)-1]
	fid, err := pathc.WalkPath(opath[:len(opath)-1], path.EndSlash(old), nil)
	if err != nil {
		return err
	}
	defer pathc.FidClnt.Clunk(fid)
	fid1, err := pathc.WalkPath(npath[:len(npath)-1], path.EndSlash(old), nil)
	if err != nil {
		return err
	}
	defer pathc.FidClnt.Clunk(fid1)
	return pathc.FidClnt.Renameat(fid, o, fid1, n)
}

func (pathc *PathClnt) umountFree(path []string) *fcall.Err {
	if fid, err := pathc.mnt.umount(path); err != nil {
		return err
	} else {
		pathc.FidClnt.Free(fid)
		return nil
	}
}

func (pathc *PathClnt) Remove(name string) error {
	db.DPrintf("PATHCLNT", "Remove %v\n", name)
	pn := path.Split(name)
	fid, rest, err := pathc.mnt.resolve(pn)
	if err != nil {
		return err
	}
	// Optimistcally remove obj without doing a pathname
	// walk; this may fail if rest contains an automount
	// symlink.
	err = pathc.FidClnt.RemoveFile(fid, rest, path.EndSlash(name))
	if err != nil {
		if fcall.IsMaybeSpecialElem(err) || fcall.IsErrUnreachable(err) {
			fid, err = pathc.WalkPath(pn, path.EndSlash(name), nil)
			if err != nil {
				return err
			}
			defer pathc.FidClnt.Clunk(fid)
			err = pathc.FidClnt.Remove(fid)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (pathc *PathClnt) Stat(name string) (*np.Stat, error) {
	db.DPrintf("PATHCLNT", "Stat %v\n", name)
	pn := path.Split(name)
	// XXX ignore err?
	target, rest, _ := pathc.mnt.resolve(pn)
	if len(rest) == 0 && !path.EndSlash(name) {
		st := &np.Stat{}
		st.Name = strings.Join(pathc.FidClnt.Lookup(target).Servers(), ",")
		return st, nil
	} else {
		fid, err := pathc.WalkPath(path.Split(name), path.EndSlash(name), nil)
		if err != nil {
			return nil, err
		}
		defer pathc.FidClnt.Clunk(fid)
		st, err := pathc.FidClnt.Stat(fid)
		if err != nil {
			return nil, err
		}
		return st, nil
	}
}

func (pathc *PathClnt) OpenWatch(pn string, mode np.Tmode, w Watch) (np.Tfid, error) {
	db.DPrintf("PATHCLNT", "Open %v %v\n", pn, mode)
	p := path.Split(pn)
	fid, err := pathc.WalkPath(p, path.EndSlash(pn), w)
	if err != nil {
		return np.NoFid, err
	}
	_, err = pathc.FidClnt.Open(fid, mode)
	if err != nil {
		return np.NoFid, err
	}
	return fid, nil
}

func (pathc *PathClnt) Open(path string, mode np.Tmode) (np.Tfid, error) {
	return pathc.OpenWatch(path, mode, nil)
}

func (pathc *PathClnt) SetDirWatch(fid np.Tfid, path string, w Watch) error {
	db.DPrintf("PATHCLNT", "SetDirWatch %v\n", fid)
	go func() {
		err := pathc.FidClnt.Watch(fid)
		db.DPrintf("PATHCLNT", "SetDirWatch: Watch returns %v %v\n", fid, err)
		if err == nil {
			w(path, nil)
		} else {
			w(path, err)
		}
	}()
	return nil
}

func (pathc *PathClnt) SetRemoveWatch(pn string, w Watch) error {
	db.DPrintf("PATHCLNT", "SetRemoveWatch %v", pn)
	p := path.Split(pn)
	fid, err := pathc.WalkPath(p, path.EndSlash(pn), nil)
	if err != nil {
		return err
	}
	if w == nil {
		return fcall.MkErr(fcall.TErrInval, "watch")
	}
	go func() {
		err := pathc.FidClnt.Watch(fid)
		db.DPrintf("PATHCLNT", "SetRemoveWatch: Watch %v %v err %v\n", fid, pn, err)
		if err == nil {
			w(pn, nil)
		} else {
			w(pn, err)
		}
		pathc.Clunk(fid)
	}()
	return nil
}

func (pathc *PathClnt) GetFile(pn string, mode np.Tmode, off np.Toffset, cnt np.Tsize) ([]byte, error) {
	db.DPrintf("PATHCLNT", "GetFile %v %v\n", pn, mode)
	p := path.Split(pn)
	fid, rest, err := pathc.mnt.resolve(p)
	if err != nil {
		return nil, err
	}
	// Optimistcally GetFile without doing a pathname
	// walk; this may fail if rest contains an automount
	// symlink.
	data, err := pathc.FidClnt.GetFile(fid, rest, mode, off, cnt, path.EndSlash(pn))
	if err != nil {
		if fcall.IsMaybeSpecialElem(err) {
			fid, err = pathc.WalkPath(p, path.EndSlash(pn), nil)
			if err != nil {
				return nil, err
			}
			defer pathc.FidClnt.Clunk(fid)
			data, err = pathc.FidClnt.GetFile(fid, []string{}, mode, off, cnt, false)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return data, nil
}

// Write file
func (pathc *PathClnt) SetFile(pn string, mode np.Tmode, data []byte, off np.Toffset) (np.Tsize, error) {
	db.DPrintf("PATHCLNT", "SetFile %v %v\n", pn, mode)
	p := path.Split(pn)
	fid, rest, err := pathc.mnt.resolve(p)
	if err != nil {
		return 0, err
	}
	// Optimistcally SetFile without doing a pathname walk; this
	// may fail if rest contains an automount symlink.
	// XXX On EOF try another replica for TestMaintainReplicationLevelCrashProcd
	cnt, err := pathc.FidClnt.SetFile(fid, rest, mode, off, data, path.EndSlash(pn))
	if err != nil {
		if fcall.IsMaybeSpecialElem(err) || fcall.IsErrUnreachable(err) {
			fid, err = pathc.WalkPath(p, path.EndSlash(pn), nil)
			if err != nil {
				return 0, err
			}
			defer pathc.FidClnt.Clunk(fid)
			cnt, err = pathc.FidClnt.SetFile(fid, []string{}, mode, off, data, false)
			if err != nil {
				return 0, err
			}
		} else {
			return 0, err
		}
	}
	return cnt, nil
}

// Create file
func (pathc *PathClnt) PutFile(pn string, mode np.Tmode, perm np.Tperm, data []byte, off np.Toffset) (np.Tsize, error) {
	db.DPrintf("PATHCLNT", "PutFile %v %v\n", pn, mode)
	p := path.Split(pn)
	fid, rest, err := pathc.mnt.resolve(p)
	if err != nil {
		return 0, err
	}
	// Optimistcally PutFile without doing a pathname
	// walk; this may fail if rest contains an automount
	// symlink.
	cnt, err := pathc.FidClnt.PutFile(fid, rest, mode, perm, off, data)
	if err != nil {
		if fcall.IsMaybeSpecialElem(err) || fcall.IsErrUnreachable(err) {
			dir := p.Dir()
			base := path.Path{p.Base()}
			fid, err = pathc.WalkPath(dir, true, nil)
			if err != nil {
				return 0, err
			}
			defer pathc.FidClnt.Clunk(fid)
			cnt, err = pathc.FidClnt.PutFile(fid, base, mode, perm, off, data)
			if err != nil {
				return 0, err
			}
		} else {
			return 0, err
		}
	}
	return cnt, nil
}

// Return path to the root directory for last server on path
func (pathc *PathClnt) PathServer(pn string) (string, path.Path, error) {
	if _, err := pathc.Stat(pn + "/"); err != nil {
		db.DPrintf("PATHCLNT_ERR", "PathServer: stat %v err %v\n", pn, err)
		return "", nil, err
	}
	p := path.Split(pn)
	_, left, err := pathc.mnt.resolve(p)
	if err != nil {
		db.DPrintf("PATHCLNT_ERR", "resolve  %v err %v\n", pn, err)
		return "", nil, err
	}
	p = p[0 : len(p)-len(left)]
	return p.String(), left, nil
}

// Return path to server, replacing ~ip with the IP address of the mounted server
func (pathc *PathClnt) AbsPathServer(pn string) (path.Path, path.Path, error) {
	srv, left, err := pathc.PathServer(pn)
	if err != nil {
		return nil, nil, err
	}
	p := path.Split(srv)
	if path.IsUnionElem(p.Base()) {
		st, err := pathc.Stat(srv)
		if err != nil {
			return path.Path{}, left, err
		}
		p[len(p)-1] = st.Name
		return p, left, nil
	}
	return p, left, nil
}
