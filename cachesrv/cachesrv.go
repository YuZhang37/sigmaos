package cachesrv

import (
	"encoding/json"
	"errors"
	"hash/fnv"
	"strconv"
	"sync"

	"sigmaos/cachesrv/proto"
	db "sigmaos/debug"
	"sigmaos/fs"
	"sigmaos/inode"
	"sigmaos/memfssrv"
	"sigmaos/proc"
	"sigmaos/protdevsrv"
	"sigmaos/serr"
	"sigmaos/sessdevsrv"
	"sigmaos/sessp"
	sp "sigmaos/sigmap"
)

const (
	DUMP = "dump"
	NBIN = 13
)

var (
	ErrMiss = errors.New("cache miss")
)

func key2bin(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	bin := h.Sum32() % NBIN
	return bin
}

type cache struct {
	sync.Mutex
	cache map[string][]byte
}

type CacheSrv struct {
	bins []cache
	shrd string
}

func RunCacheSrv(args []string) error {
	s := &CacheSrv{}
	if len(args) > 3 {
		s.shrd = args[3]
	}
	public, err := strconv.ParseBool(args[2])
	if err != nil {
		return err
	}
	s.bins = make([]cache, NBIN)
	for i := 0; i < NBIN; i++ {
		s.bins[i].cache = make(map[string][]byte)
	}
	db.DPrintf(db.CACHESRV, "%v: Run %v\n", proc.GetName(), s.shrd)
	pds, err := protdevsrv.MakeProtDevSrvPublic(sp.CACHE+s.shrd, s, public)
	if err != nil {
		return err
	}
	if err := sessdevsrv.MkSessDev(pds.MemFs, DUMP, s.mkSession, nil); err != nil {
		return err
	}
	return pds.RunServer()
}

// XXX support timeout
func (s *CacheSrv) Set(ctx fs.CtxI, req proto.CacheRequest, rep *proto.CacheResult) error {
	db.DPrintf(db.CACHESRV, "%v: Set %v\n", proc.GetName(), req)

	b := key2bin(req.Key)

	s.bins[b].Lock()
	defer s.bins[b].Unlock()

	s.bins[b].cache[req.Key] = req.Value
	return nil
}

func (s *CacheSrv) Get(ctx fs.CtxI, req proto.CacheRequest, rep *proto.CacheResult) error {
	db.DPrintf(db.CACHESRV, "%v: Get %v\n", proc.GetName(), req)
	b := key2bin(req.Key)

	s.bins[b].Lock()
	defer s.bins[b].Unlock()

	v, ok := s.bins[b].cache[req.Key]
	if ok {
		rep.Value = v
		return nil
	}
	return ErrMiss
}

type cacheSession struct {
	*inode.Inode
	bins []cache
	sid  sessp.Tsession
}

func (s *CacheSrv) mkSession(mfs *memfssrv.MemFs, sid sessp.Tsession) (fs.Inode, *serr.Err) {
	cs := &cacheSession{mfs.MakeDevInode(), s.bins, sid}
	db.DPrintf(db.CACHESRV, "mkSession %v %p\n", cs.bins, cs)
	return cs, nil
}

// XXX incremental read
func (cs *cacheSession) Read(ctx fs.CtxI, off sp.Toffset, cnt sessp.Tsize, v sp.TQversion) ([]byte, *serr.Err) {
	if off > 0 {
		return nil, nil
	}
	db.DPrintf(db.CACHESRV, "Dump cache %p %v\n", cs, cs.bins)
	m := make(map[string][]byte)
	for i, _ := range cs.bins {
		cs.bins[i].Lock()
		for k, v := range cs.bins[i].cache {
			m[k] = v
		}
		cs.bins[i].Unlock()
	}

	b, err := json.Marshal(m)
	if err != nil {
		return nil, serr.MkErrError(err)
	}
	return b, nil
}

func (cs *cacheSession) Write(ctx fs.CtxI, off sp.Toffset, b []byte, v sp.TQversion) (sessp.Tsize, *serr.Err) {
	return 0, serr.MkErr(serr.TErrNotSupported, nil)
}

func (cs *cacheSession) Close(ctx fs.CtxI, m sp.Tmode) *serr.Err {
	db.DPrintf(db.CACHESRV, "%v: Close %v\n", proc.GetName(), cs.sid)
	return nil
}
