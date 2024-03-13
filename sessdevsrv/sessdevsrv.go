package sessdevsrv

import (
	"path"

	"sigmaos/clonedev"
	db "sigmaos/debug"
	"sigmaos/fs"
	"sigmaos/memfssrv"
	"sigmaos/serr"
	"sigmaos/sessdev"
	"sigmaos/sessp"
	sp "sigmaos/sigmap"
)

type NewSessionF func(*memfssrv.MemFs, sessp.Tsession) (fs.Inode, *serr.Err)

type SessDev struct {
	mfs  *memfssrv.MemFs
	dir  string
	news NewSessionF
}

// Make a SessDev in mfs in the directory pn
func NewSessDev(mfs *memfssrv.MemFs, dir string, news NewSessionF, wctl clonedev.WriteCtlF) error {
	db.DPrintf(db.SESSDEV, "NewSessDev: %v\n", dir)
	sd := &SessDev{mfs, dir, news}
	if err := clonedev.NewCloneDev(mfs, dir, sd.newSession, sd.detachSession, wctl); err != nil {
		return err
	}
	return nil
}

// XXX clean up in case of error
func (sd *SessDev) newSession(mfs *memfssrv.MemFs, sid sessp.Tsession) *serr.Err {
	sess, err := sd.news(mfs, sid)
	if err != nil {
		return err
	}
	fn := path.Join(sd.dir, sid.String(), sessdev.DATA)
	db.DPrintf(db.SESSDEV, "newSession %v\n", fn)
	if err := mfs.NewDev(fn, sess); err != nil {
		db.DPrintf(db.SESSDEV, "newSession %v err %v\n", fn, err)
		return err
	}
	return nil
}

func (sd *SessDev) detachSession(sid sessp.Tsession) {
	fn := path.Join(sd.dir, sid.String(), sessdev.DATA)
	if err := sd.mfs.Remove(fn); err != nil {
		db.DPrintf(db.SESSDEV, "detachSession %v err %v\n", fn, err)
	}
}

func (sd *SessDev) Close(ctx fs.CtxI, m sp.Tmode) *serr.Err {
	fn := path.Join(sd.dir, ctx.SessionId().String(), sessdev.DATA)
	db.DPrintf(db.SESSDEV, "Close %v\n", fn)
	return nil
}
