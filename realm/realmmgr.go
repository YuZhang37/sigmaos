package realm

import (
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	"ulambda/config"
	"ulambda/ctx"
	db "ulambda/debug"
	"ulambda/dir"
	"ulambda/fs"
	"ulambda/fslib"
	"ulambda/fslibsrv"
	"ulambda/kernel"
	np "ulambda/ninep"
	"ulambda/stats"
	"ulambda/sync"
)

const (
	SCAN_INTERVAL_MS          = 50
	RESIZE_INTERVAL_MS        = 100
	GROW_CPU_UTIL_THRESHOLD   = 50
	SHRINK_CPU_UTIL_THRESHOLD = 25
)

const (
	free_machineds  = "free-machineds"
	realm_create    = "realm_create"
	realm_destroy   = "realm_destroy"
	FREE_MACHINEDS  = np.REALM_MGR + "/" + free_machineds // Unassigned machineds
	REALM_CREATE    = np.REALM_MGR + "/" + realm_create   // Realm allocation requests
	REALM_DESTROY   = np.REALM_MGR + "/" + realm_destroy  // Realm destruction requests
	REALMS          = "name/realms"                       // List of realms, with machineds registered under them
	REALM_CONFIG    = "name/realm-config"                 // Store of realm configs
	MACHINED_CONFIG = "name/machined-config"              // Store of machined configs
	REALM_NAMEDS    = "name/realm-nameds"                 // Symlinks to realms' nameds
)

type RealmMgr struct {
	freeMachineds chan string
	realmCreate   chan string
	realmDestroy  chan string
	root          fs.Dir
	*config.ConfigClnt
	*fslib.FsLib
	*fslibsrv.MemFs
}

func MakeRealmMgr() *RealmMgr {
	m := &RealmMgr{}
	m.freeMachineds = make(chan string)
	m.realmCreate = make(chan string)
	m.realmDestroy = make(chan string)
	var err error
	m.MemFs, _, err = fslibsrv.MakeMemFs(np.REALM_MGR, "realmmgr")
	if err != nil {
		log.Fatalf("Error MakeMemFs in MakeRealmMgr: %v", err)
	}
	m.FsLib = m.MemFs.FsLib
	m.ConfigClnt = config.MakeConfigClnt(m.FsLib)
	m.makeInitFs()
	m.makeCtlFiles()

	return m
}

func (m *RealmMgr) makeInitFs() {
	if err := m.Mkdir(REALMS, 0777); err != nil {
		log.Fatalf("Error Mkdir REALMS in RealmMgr.makeInitFs: %v", err)
	}
	if err := m.Mkdir(REALM_CONFIG, 0777); err != nil {
		log.Fatalf("Error Mkdir REALM_CONFIG in RealmMgr.makeInitFs: %v", err)
	}
	if err := m.Mkdir(MACHINED_CONFIG, 0777); err != nil {
		log.Fatalf("Error Mkdir MACHINED_CONFIG in RealmMgr.makeInitFs: %v", err)
	}
	if err := m.Mkdir(REALM_NAMEDS, 0777); err != nil {
		log.Fatalf("Error Mkdir REALM_NAMEDS in RealmMgr.makeInitFs: %v", err)
	}
}

func (m *RealmMgr) makeCtlFiles() {
	// Set up control files
	realmCreate := makeCtlFile(m.realmCreate, nil, m.Root())
	err := dir.MkNod(ctx.MkCtx("", 0, nil), m.Root(), realm_create, realmCreate)
	if err != nil {
		log.Fatalf("Error MkNod in RealmMgr.makeCtlFiles 1: %v", err)
	}

	realmDestroy := makeCtlFile(m.realmDestroy, nil, m.Root())
	err = dir.MkNod(ctx.MkCtx("", 0, nil), m.Root(), realm_destroy, realmDestroy)
	if err != nil {
		log.Fatalf("Error MkNod in RealmMgr.makeCtlFiles 2: %v", err)
	}

	freeMachineds := makeCtlFile(m.freeMachineds, nil, m.Root())
	err = dir.MkNod(ctx.MkCtx("", 0, nil), m.Root(), free_machineds, freeMachineds)
	if err != nil {
		log.Fatalf("Error MkNod in RealmMgr.makeCtlFiles 3: %v", err)
	}
}

// Handle realm creation requests.
func (m *RealmMgr) createRealms() {
	for {
		// Get a realm creation request
		realmId := <-m.realmCreate

		realmLock := sync.MakeLock(m.FsLib, np.LOCKS, REALM_LOCK+realmId, true)
		realmLock.Lock()

		// Unmarshal the realm config file.
		cfg := &RealmConfig{}
		cfg.Rid = realmId

		// Make a directory for this realm.
		if err := m.Mkdir(path.Join(REALMS, realmId), 0777); err != nil {
			log.Fatalf("Error Mkdir in RealmMgr.createRealms: %v", err)
		}

		// Make a directory for this realm's nameds.
		if err := m.Mkdir(path.Join(REALM_NAMEDS, realmId), 0777); err != nil {
			log.Fatalf("Error Mkdir in RealmMgr.createRealms: %v", err)
		}

		// Make the realm config file.
		m.WriteConfig(path.Join(REALM_CONFIG, realmId), cfg)

		realmLock.Unlock()
	}
}

// Deallocate a machined from a realm.
func (m *RealmMgr) deallocMachined(realmId string, machinedId string) {
	rdCfg := &MachinedConfig{}
	rdCfg.Id = machinedId
	rdCfg.RealmId = kernel.NO_REALM

	// Update the machined config file.
	m.WriteConfig(path.Join(MACHINED_CONFIG, machinedId), rdCfg)

	// Note machined de-registration
	rCfg := &RealmConfig{}
	m.ReadConfig(path.Join(REALM_CONFIG, realmId), rCfg)
	rCfg.NMachineds -= 1
	rCfg.LastResize = time.Now()
	m.WriteConfig(path.Join(REALM_CONFIG, realmId), rCfg)
}

func (m *RealmMgr) deallocAllMachineds(realmId string) {
	rds, err := m.ReadDir(path.Join(REALMS, realmId))
	if err != nil {
		log.Fatalf("Error ReadDir in RealmMgr.deallocRealms: %v", err)
	}

	for _, machined := range rds {
		m.deallocMachined(realmId, machined.Name)
	}
}

func (m *RealmMgr) destroyRealms() {
	for {
		// Get a realm creation request
		realmId := <-m.realmDestroy

		realmLock := sync.MakeLock(m.FsLib, np.LOCKS, REALM_LOCK+realmId, true)
		realmLock.Lock()

		m.deallocAllMachineds(realmId)

		cfg := &RealmConfig{}
		m.ReadConfig(path.Join(REALM_CONFIG, realmId), cfg)
		cfg.Shutdown = true
		m.WriteConfig(path.Join(REALM_CONFIG, realmId), cfg)

		realmLock.Unlock()
	}
}

// Get & alloc a machined to this realm. Retur true if successful
func (m *RealmMgr) allocMachined(realmId string) bool {
	// Get a free machined
	select {
	// If there is a machined available...
	case machinedId := <-m.freeMachineds:
		// Update the machined's config
		rdCfg := &MachinedConfig{}
		rdCfg.Id = machinedId
		rdCfg.RealmId = realmId
		m.WriteConfig(path.Join(MACHINED_CONFIG, machinedId), rdCfg)

		// Update the realm's config
		rCfg := &RealmConfig{}
		m.ReadConfig(path.Join(REALM_CONFIG, realmId), rCfg)
		rCfg.NMachineds += 1
		rCfg.LastResize = time.Now()
		m.WriteConfig(path.Join(REALM_CONFIG, realmId), rCfg)
		return true
		// If no machined is available...
	default:
		return false
	}
}

// Get all a realm's procd's stats
func (m *RealmMgr) getRealmProcdStats(nameds []string, realmId string) map[string]*stats.StatInfo {
	// XXX For now we assume all the nameds are up
	stat := make(map[string]*stats.StatInfo)
	if len(nameds) == 0 {
		return stat
	}
	// XXX May fail if this named crashed
	procds, err := m.ReadDir(path.Join(REALM_NAMEDS, realmId, "~ip", np.PROCDREL))
	if err != nil {
		log.Fatalf("Error ReadDir 2 in RealmMgr.getRealmProcdStats: %v", err)
	}
	for _, pd := range procds {
		s := &stats.StatInfo{}
		// XXX May fail if this named crashed
		err := m.ReadFileJson(path.Join(REALM_NAMEDS, realmId, "~ip", np.PROCDREL, pd.Name, "statsd"), s)
		if err != nil {
			log.Fatalf("Error ReadFileJson in RealmMgr.getRealmProcdStats: %v", err)
		}
		stat[pd.Name] = s
	}
	return stat
}

func (m *RealmMgr) getRealmConfig(realmId string) (*RealmConfig, error) {
	// If the realm is being shut down, the realm config file may not be there
	// anymore. In this case, another machined is not needed.
	if _, err := m.Stat(path.Join(REALM_CONFIG, realmId)); err != nil && strings.Contains(err.Error(), "file not found") {
		return nil, fmt.Errorf("Realm not found")
	}
	cfg := &RealmConfig{}
	m.ReadConfig(path.Join(REALM_CONFIG, realmId), cfg)
	return cfg, nil
}

func (m *RealmMgr) getRealmUtil(realmId string, cfg *RealmConfig) (float64, map[string]float64) {
	// Get stats
	utilMap := make(map[string]float64)
	procdStats := m.getRealmProcdStats(cfg.NamedAddr, realmId)
	avgUtil := 0.0
	for machinedId, stat := range procdStats {
		avgUtil += stat.Util
		utilMap[machinedId] = stat.Util
	}
	avgUtil /= float64(len(procdStats))
	return avgUtil, utilMap
}

func (m *RealmMgr) adjustRealm(realmId string) {
	// Get the realm's config
	realmCfg, err := m.getRealmConfig(realmId)
	if err != nil {
		db.DLPrintf("REALMMGR", "Error RealmMgr.getRealmConfig in RealmMgr.adjustRealm: %v", err)
		return
	}

	// If the realm is shutting down, return
	if realmCfg.Shutdown {
		return
	}

	// If we are below the target replication level
	if realmCfg.NMachineds < nReplicas() {
		// Start enough machineds to reach the target replication level
		for i := realmCfg.NMachineds; i < nReplicas(); i++ {
			if ok := m.allocMachined(realmId); !ok {
				log.Fatalf("Error in adjustRealm: not enough machineds to meet minimum replication level")
			}
		}
		return
	}

	// If we have resized too recently, return
	if time.Now().Sub(realmCfg.LastResize).Milliseconds() < RESIZE_INTERVAL_MS {
		return
	}

	//	log.Printf("Avg util pre: %v", realmCfg)
	avgUtil, procdUtils := m.getRealmUtil(realmId, realmCfg)
	//	log.Printf("Avg util post: %v, %v", realmCfg, avgUtil)
	if avgUtil > GROW_CPU_UTIL_THRESHOLD {
		m.allocMachined(realmId)
	} else if avgUtil < SHRINK_CPU_UTIL_THRESHOLD {
		// If there are replicas to spare
		if realmCfg.NMachineds > nReplicas() {
			// Find least utilized procd
			//			min := 100.0
			//			minMachinedId := ""
			//			for machinedId, util := range procdUtils {
			//				if min > util {
			//					min = util
			//					minMachinedId = machinedId
			//				}
			//			}
			// XXX A hack for now, since we don't have a good way of linking a procd to a machined
			_ = procdUtils
			machinedIds, err := m.ReadDir(path.Join(REALMS, realmId))
			if err != nil {
				log.Printf("Error ReadDir in RealmMgr.adjustRealm: %v", err)
			}
			minMachinedId := machinedIds[1].Name
			// Deallocate least utilized procd
			m.deallocMachined(realmId, minMachinedId)
		}
	}
}

// Balance machineds across realms.
func (m *RealmMgr) balanceMachineds() {
	for {
		realms, err := m.ReadDir(REALMS)
		if err != nil {
			log.Fatalf("Error ReadDir in RealmMgr.balanceMachineds: %v", err)
		}

		for _, realm := range realms {
			realmId := realm.Name
			// XXX Currently we assume there are always enough machineds for the number
			// of realms we have. If that assumption is broken, this may deadlock when
			// a realm is trying to exit & we're trying to assign a machined to it.
			realmLock := sync.MakeLock(m.FsLib, np.LOCKS, REALM_LOCK+realmId, true)
			realmLock.Lock()

			m.adjustRealm(realmId)

			realmLock.Unlock()
		}

		time.Sleep(SCAN_INTERVAL_MS * time.Millisecond)
	}
}

func (m *RealmMgr) Work() {
	go m.createRealms()
	go m.destroyRealms()
	go m.balanceMachineds()
	m.Serve()
	m.Done()
}
