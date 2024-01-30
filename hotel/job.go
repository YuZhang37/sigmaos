package hotel

import (
	"os"
	"path"
	"strconv"

	"sigmaos/cache"
	"sigmaos/cachedsvc"
	"sigmaos/cachedsvcclnt"
	db "sigmaos/debug"
	"sigmaos/fslib"
	"sigmaos/kv"
	"sigmaos/proc"
	"sigmaos/rpc"
	"sigmaos/sigmaclnt"
	sp "sigmaos/sigmap"
	"sigmaos/test"
)

const (
	HOTEL        = "hotel/"
	HOTELDIR     = "name/" + HOTEL
	HOTELGEO     = HOTELDIR + "geo"
	HOTELRATE    = HOTELDIR + "rate"
	HOTELSEARCH  = HOTELDIR + "search"
	HOTELREC     = HOTELDIR + "rec"
	HOTELRESERVE = HOTELDIR + "reserve"
	HOTELUSER    = HOTELDIR + "user"
	HOTELPROF    = HOTELDIR + "prof"

	MEMFS          = "memfs"
	HTTP_ADDRS     = "http-addr"
	TRACING        = false
	N_RPC_SESSIONS = 10
)

var HOTELSVC = []string{HOTELGEO, HOTELRATE, HOTELSEARCH, HOTELREC, HOTELRESERVE,
	HOTELUSER, HOTELPROF, sp.DB + "~any/"}

var (
	nhotel    int
	imgSizeMB int
)

func init() {
	nh := os.Getenv("NHOTEL")
	if nh == "" {
		// Defaults to 80, same as original DSB.
		nhotel = 80
	} else {
		i, err := strconv.Atoi(nh)
		if err != nil {
			db.DFatalf("Can't parse nhotel: %v", err)
		}
		setNHotel(i)
	}
	isb := os.Getenv("HOTEL_IMG_SZ_MB")
	if isb != "" {
		i, err := strconv.Atoi(isb)
		if err != nil {
			db.DFatalf("Can't parse imgSize: %v", err)
		}
		imgSizeMB = i
	}
}

func setNHotel(nh int) {
	nhotel = nh
}

func JobDir(job string) string {
	return path.Join(HOTELDIR, job)
}

func JobHTTPAddrsPath(job string) string {
	return path.Join(JobDir(job), HTTP_ADDRS)
}

func MemFsPath(job string) string {
	return path.Join(JobDir(job), MEMFS)
}

func NewFsLibs(uname string) ([]*fslib.FsLib, error) {
	pe := proc.GetProcEnv()
	fsls := make([]*fslib.FsLib, 0, N_RPC_SESSIONS)
	for i := 0; i < N_RPC_SESSIONS; i++ {
		pen := proc.NewAddedProcEnv(pe, i)
		fsl, err := sigmaclnt.NewFsLib(pen)
		if err != nil {
			db.DPrintf(db.ERROR, "Error newfsl: %v", err)
			return nil, err
		}
		fsls = append(fsls, fsl)
	}
	return fsls, nil
}

func GetJobHTTPAddrs(fsl *fslib.FsLib, job string) (sp.Taddrs, error) {
	mnt, err := fsl.ReadMount(JobHTTPAddrsPath(job))
	if err != nil {
		return nil, err
	}
	return mnt.Addr, err
}

func InitHotelFs(fsl *fslib.FsLib, jobname string) error {
	fsl.MkDir(HOTELDIR, 0777)
	if err := fsl.MkDir(JobDir(jobname), 0777); err != nil {
		db.DPrintf(db.ERROR, "Mkdir %v err %v\n", JobDir(jobname), err)
		return err
	}
	return nil
}

type Srv struct {
	Name         string
	Public       bool
	Mcpu         proc.Tmcpu
	AllowedPaths []string
}

//	,[]string{sp.NAMED, path.Join(sp.SCHEDD, "*"), path.Join(sp.DB, "*")}

// XXX searchd only needs 2, but in order to make spawns work out we need to have it run with 3.
func NewHotelSvc(public bool) []Srv {
	return []Srv{
		Srv{"hotel-userd", public, 0, []string{sp.NAMED, path.Join(sp.SCHEDD, "*"), path.Join(sp.DB, "*")}},
		Srv{"hotel-rated", public, 2000, []string{sp.NAMED, path.Join(sp.SCHEDD, "*"), path.Join(sp.DB, "*"), path.Join(cache.CACHE, "servers", "*")}},
		Srv{"hotel-geod", public, 2000, []string{sp.NAMED, path.Join(sp.SCHEDD, "*"), path.Join(sp.DB, "*")}},
		Srv{"hotel-profd", public, 2000, []string{sp.NAMED, path.Join(sp.SCHEDD, "*"), path.Join(sp.DB, "*"), path.Join(cache.CACHE, "servers", "*")}},
		Srv{"hotel-searchd", public, 3000, []string{sp.NAMED, path.Join(sp.SCHEDD, "*"), path.Join(sp.DB, "*"), HOTELRATE, HOTELGEO}},
		Srv{"hotel-reserved", public, 3000, []string{sp.NAMED, path.Join(sp.SCHEDD, "*"), path.Join(sp.DB, "*"), path.Join(cache.CACHE, "servers", "*")}},
		Srv{"hotel-recd", public, 0, []string{sp.NAMED, path.Join(sp.SCHEDD, "*"), path.Join(sp.DB, "*")}},
		Srv{"hotel-wwwd", public, 3000, []string{sp.NAMED, path.Join(sp.SCHEDD, "*"), path.Join(sp.DB, "*"), HOTELSEARCH, HOTELUSER, HOTELPROF, HOTELREC, HOTELGEO, HOTELRESERVE}},
	}
}

//var ncores = []int{0, 1,
//	1, 1, 3,
//	3, 0, 2}

//var ncores = []int{0, 2,
//	2, 2, 3,
//	3, 0, 2}

type HotelJob struct {
	*sigmaclnt.SigmaClnt
	cacheClnt       *cachedsvcclnt.CachedSvcClnt
	cacheMgr        *cachedsvc.CacheMgr
	CacheAutoscaler *cachedsvcclnt.Autoscaler
	pids            []sp.Tpid
	cache           string
	kvf             *kv.KVFleet
}

func NewHotelJob(sc *sigmaclnt.SigmaClnt, job string, srvs []Srv, nhotel int, cache string, cacheMcpu proc.Tmcpu, nsrv int, gc bool, imgSizeMB int) (*HotelJob, error) {
	// Set number of hotels before doing anything.
	setNHotel(nhotel)

	var cc *cachedsvcclnt.CachedSvcClnt
	var cm *cachedsvc.CacheMgr
	var ca *cachedsvcclnt.Autoscaler
	var err error
	var kvf *kv.KVFleet

	// Init fs.
	if err := InitHotelFs(sc.FsLib, job); err != nil {
		return nil, err
	}

	// Create a cache clnt.
	if nsrv > 0 {
		switch cache {
		case "cached":
			db.DPrintf(db.ALWAYS, "Hotel running with cached")
			cm, err = cachedsvc.NewCacheMgr(sc, job, nsrv, cacheMcpu, gc, test.Overlays)
			if err != nil {
				db.DPrintf(db.ERROR, "Error NewCacheMgr %v", err)
				return nil, err
			}
			cc, err = cachedsvcclnt.NewCachedSvcClnt([]*fslib.FsLib{sc.FsLib}, job)
			if err != nil {
				db.DPrintf(db.ERROR, "Error NewCachedSvcClnt %v", err)
				return nil, err
			}
			ca = cachedsvcclnt.NewAutoscaler(cm, cc)
		case "kvd":
			db.DPrintf(db.ALWAYS, "Hotel running with kvd")
			kvf, err = kv.NewKvdFleet(sc, job, 0, nsrv, 0, 0, cacheMcpu, "0", "manual")
			if err != nil {
				return nil, err
			}
			err = kvf.Start()
			if err != nil {
				return nil, err
			}
		// XXX Remove
		case "memcached":
			db.DPrintf(db.ALWAYS, "Hotel running with memcached")
		default:
			db.DPrintf(db.ERROR, "Unrecognized hotel cache type: %v", cache)
		}
	}

	pids := make([]sp.Tpid, 0, len(srvs))

	for _, srv := range srvs {
		db.DPrintf(db.TEST, "Hotel spawn %v", srv.Name)
		p := proc.NewProc(srv.Name, []string{job, strconv.FormatBool(srv.Public), cache})
		p.SetAllowedPaths(srv.AllowedPaths)
		p.AppendEnv("NHOTEL", strconv.Itoa(nhotel))
		p.AppendEnv("HOTEL_IMG_SZ_MB", strconv.Itoa(imgSizeMB))
		p.SetMcpu(srv.Mcpu)
		if err := sc.Spawn(p); err != nil {
			db.DPrintf(db.ERROR, "Error burst-spawnn proc %v: %v", p, err)
			return nil, err
		}
		if err = sc.WaitStart(p.GetPid()); err != nil {
			db.DPrintf(db.ERROR, "Error spawn proc %v: %v", p, err)
			return nil, err
		}
		pids = append(pids, p.GetPid())
		db.DPrintf(db.TEST, "Hotel started %v", srv.Name)
	}

	return &HotelJob{sc, cc, cm, ca, pids, cache, kvf}, nil
}

func (hj *HotelJob) Stop() error {
	if hj.CacheAutoscaler != nil {
		hj.CacheAutoscaler.Stop()
	}
	for _, pid := range hj.pids {
		if err := hj.Evict(pid); err != nil {
			return err
		}
		if _, err := hj.WaitExit(pid); err != nil {
			return err
		}
	}
	if hj.cacheMgr != nil {
		hj.cacheMgr.Stop()
	}
	if hj.kvf != nil {
		hj.kvf.Stop()
	}
	return nil
}

func (hj *HotelJob) StatsSrv() ([]*rpc.SigmaRPCStats, error) {
	if hj.cacheClnt != nil {
		return hj.cacheClnt.StatsSrvs()
	}
	db.DPrintf(db.ALWAYS, "No cacheclnt")
	return nil, nil
}
