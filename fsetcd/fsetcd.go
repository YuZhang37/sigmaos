package fsetcd

import (
	"context"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"

	db "sigmaos/debug"
	"sigmaos/serr"
	sp "sigmaos/sigmap"
)

const (
	DialTimeout = 5 * time.Second
	SessionTTL  = 5
	LeaseTTL    = SessionTTL // 30
	NCLNT       = 1
)

var (
	endpointsBase = []string{":2379", ":2380", ":2381"}
)

type FsEtcd struct {
	sync.Mutex
	clnts    []*clientv3.Client
	next     int
	fencekey string
	fencerev int64
	realm    sp.Trealm
}

func NewFsEtcd(realm sp.Trealm, etcdIP string) (*FsEtcd, error) {
	endpoints := []string{}
	for i := range endpointsBase {
		endpoints = append(endpoints, etcdIP+endpointsBase[i])
	}
	db.DPrintf(db.FSETCD, "FsEtcd etcd endpoints: %v", endpoints)
	clnts := make([]*clientv3.Client, 0, NCLNT)
	for i := 0; i < NCLNT; i++ {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   endpoints,
			DialTimeout: DialTimeout,
		})
		if err != nil {
			return nil, err
		}
		clnts = append(clnts, cli)
	}
	fs := &FsEtcd{clnts: clnts, realm: realm}
	return fs, nil
}

func (fs *FsEtcd) Close() error {
	var err error
	for _, clnt := range fs.clnts {
		if e := clnt.Close(); e != nil {
			err = e
		}
	}
	return err
}

func (fs *FsEtcd) incNext() int {
	fs.Lock()
	defer fs.Unlock()
	n := fs.next
	fs.next = (fs.next + 1) % NCLNT
	return n
}

func (fs *FsEtcd) Clnt() *clientv3.Client {
	n := fs.incNext()
	return fs.clnts[n]
}

func (fs *FsEtcd) Fence(key string, rev int64) {
	db.DPrintf(db.FSETCD, "Fence key %v rev %d\n", key, rev)
	fs.fencekey = key
	fs.fencerev = rev
}

func (fs *FsEtcd) Detach(cid sp.TclntId) {
}

func (fs *FsEtcd) SetRootNamed(mnt sp.Tmount) *serr.Err {
	db.DPrintf(db.FSETCD, "SetRootNamed %v", mnt)
	d, err := mnt.Marshal()
	if err != nil {
		return serr.NewErrError(err)
	}
	nf := NewEtcdFile(sp.DMSYMLINK, sp.NoClntId, sp.NoLeaseId, d)
	if b, err := proto.Marshal(nf); err != nil {
		return serr.NewErrError(err)
	} else {
		cmp := []clientv3.Cmp{
			clientv3.Compare(clientv3.CreateRevision(fs.fencekey), "=", fs.fencerev),
		}
		ops := []clientv3.Op{
			clientv3.OpPut(fs.path2key(sp.ROOTREALM, BOOT), string(b)),
		}
		resp, err := fs.Clnt().Txn(context.TODO()).If(cmp...).Then(ops...).Commit()
		if err != nil {
			db.DPrintf(db.FSETCD, "SetNamed txn %v err %v\n", nf, err)
			return serr.NewErrError(err)
		}
		db.DPrintf(db.FSETCD, "SetNamed txn %v %v\n", nf, resp)
		return nil
	}
}

func GetRootNamed(realm sp.Trealm, etcdIP string) (sp.Tmount, *serr.Err) {
	fs, err := NewFsEtcd(realm, etcdIP)
	if err != nil {
		return sp.Tmount{}, serr.NewErrError(err)
	}
	defer fs.Close()

	nf, _, sr := fs.getFile(fs.path2key(sp.ROOTREALM, sp.Tpath(BOOT)))
	if sr != nil {
		db.DPrintf(db.FSETCD, "GetFile %v nf %v err %v realm %v etcdIP %v", BOOT, nf, sr, realm, etcdIP)
		return sp.Tmount{}, sr
	}
	mnt, sr := sp.NewMount(nf.Data)
	if sr != nil {
		db.DPrintf(db.FSETCD, "NewMount %v err %v\n", BOOT, err)
		return sp.Tmount{}, sr
	}
	db.DPrintf(db.FSETCD, "GetNamed mnt %v\n", mnt)
	fs.Close()
	return mnt, nil
}
