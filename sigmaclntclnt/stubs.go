package sigmaclntclnt

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	db "sigmaos/debug"
	"sigmaos/path"
	rpcproto "sigmaos/rpc/proto"
	"sigmaos/serr"
	"sigmaos/sessp"
	scproto "sigmaos/sigmaclntsrv/proto"
	sos "sigmaos/sigmaos"
	sp "sigmaos/sigmap"
)

func (scc *SigmaClntClnt) rpcErr(method string, req proto.Message, rep *scproto.SigmaErrReply) error {
	err := scc.rpcc.RPC(method, req, rep)
	if err != nil {
		return err
	}
	if rep.Err.TErrCode() != serr.TErrNoError {
		return sp.NewErr(rep.Err)
	}
	return nil
}

func (scc *SigmaClntClnt) rpcFd(method string, req proto.Message, rep *scproto.SigmaFdReply) (int, error) {
	err := scc.rpcc.RPC(method, req, rep)
	if err != nil {
		return -1, err
	}
	if rep.Err.TErrCode() != serr.TErrNoError {
		return -1, sp.NewErr(rep.Err)
	}
	return int(rep.Fd), nil
}

func (scc *SigmaClntClnt) rpcData(method string, req proto.Message, rep *scproto.SigmaDataReply) (sessp.IoVec, error) {
	err := scc.rpcc.RPC(method, req, rep)
	if err != nil {
		return nil, err
	}
	if rep.Err.TErrCode() != serr.TErrNoError {
		return nil, sp.NewErr(rep.Err)
	}
	return sessp.NewIoVec(rep.Blob.Iov), nil
}

func (scc *SigmaClntClnt) rpcSize(method string, req proto.Message, rep *scproto.SigmaSizeReply) (sp.Tsize, error) {
	err := scc.rpcc.RPC(method, req, rep)
	if err != nil {
		return 0, err
	}
	if rep.Err.TErrCode() == serr.TErrNoError {
		return sp.Tsize(rep.Size), nil
	} else {
		return 0, sp.NewErr(rep.Err)
	}
}

func (scc *SigmaClntClnt) Init() error {
	req := scproto.SigmaInitRequest{ProcEnvProto: scc.pe.GetProto()}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.Init", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Init %v %v %v", req, rep, err)
	return err
}

func (scc *SigmaClntClnt) CloseFd(fd int) error {
	req := scproto.SigmaCloseRequest{Fd: uint32(fd)}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.CloseFd", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "CloseFd %v %v %v", req, rep, err)
	return err
}

func (scc *SigmaClntClnt) Stat(path string) (*sp.Stat, error) {
	req := scproto.SigmaPathRequest{Path: path}
	rep := scproto.SigmaStatReply{}
	err := scc.rpcc.RPC("SigmaClntSrvAPI.Stat", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Stat %v %v %v", req, rep, err)
	if err != nil {
		return nil, err
	}
	if rep.Err.TErrCode() != serr.TErrNoError {
		return nil, sp.NewErr(rep.Err)
	}
	return rep.Stat, nil
}

func (scc *SigmaClntClnt) Create(path string, p sp.Tperm, m sp.Tmode) (int, error) {
	req := scproto.SigmaCreateRequest{Path: path, Perm: uint32(p), Mode: uint32(m)}
	rep := scproto.SigmaFdReply{}
	fd, err := scc.rpcFd("SigmaClntSrvAPI.Create", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Create %v %v fd %v err %v", req, rep, fd, err)
	return fd, err
}

func (scc *SigmaClntClnt) Open(path string, m sp.Tmode, w sos.Twait) (int, error) {
	req := scproto.SigmaCreateRequest{Path: path, Mode: uint32(m), Wait: bool(w)}
	rep := scproto.SigmaFdReply{}
	fd, err := scc.rpcFd("SigmaClntSrvAPI.Open", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Open %v %v %v %v", req, rep, fd, err)
	return fd, err
}

func (scc *SigmaClntClnt) Rename(src, dst string) error {
	req := scproto.SigmaRenameRequest{Src: src, Dst: dst}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.Rename", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Rename %v %v %v", req, rep, err)
	return err
}

func (scc *SigmaClntClnt) Remove(path string) error {
	req := scproto.SigmaPathRequest{Path: path}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.Remove", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Remove %v %v %v", req, rep, err)
	return err

}

func (scc *SigmaClntClnt) GetFile(path string) ([]byte, error) {
	req := scproto.SigmaPathRequest{Path: path}
	rep := scproto.SigmaDataReply{}
	rep.Blob = &rpcproto.Blob{Iov: [][]byte{nil}}
	d, err := scc.rpcData("SigmaClntSrvAPI.GetFile", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "GetFile %v %v %v", req, d, err)
	if err != nil {
		return nil, err
	}
	return d[0], nil
}

func (scc *SigmaClntClnt) PutFile(path string, p sp.Tperm, m sp.Tmode, data []byte, o sp.Toffset, l sp.TleaseId) (sp.Tsize, error) {
	blob := &rpcproto.Blob{Iov: [][]byte{data}}
	req := scproto.SigmaPutFileRequest{Path: path, Perm: uint32(p), Mode: uint32(m), Offset: uint64(o), LeaseId: uint64(l), Blob: blob}
	rep := scproto.SigmaSizeReply{}
	sz, err := scc.rpcSize("SigmaClntSrvAPI.PutFile", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "PutFile %q %v %v sz %v %v", req.Path, len(data), rep, sz, err)
	return sz, err
}

func (scc *SigmaClntClnt) Read(fd int, b []byte) (sp.Tsize, error) {
	req := scproto.SigmaReadRequest{Fd: uint32(fd), Size: uint64(len(b))}
	rep := scproto.SigmaDataReply{}
	rep.Blob = &rpcproto.Blob{Iov: [][]byte{b}}
	db.DPrintf(db.ALWAYS, "Outblob caller %p", rep.Blob.Iov[0])
	d, err := scc.rpcData("SigmaClntSrvAPI.Read", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Read %v len %v err %v", req, len(d), err)
	if err != nil {
		return 0, err
	}
	db.DPrintf(db.SIGMACLNTCLNT, "Read %v len %v", req, len(d[0]))
	db.DPrintf(db.ALWAYS, "Read %v %v d: %v\nb: %v", req, len(d[0]), d[0], b)
	return sp.Tsize(len(d[0])), nil
}

func (scc *SigmaClntClnt) Write(fd int, data []byte) (sp.Tsize, error) {
	blob := &rpcproto.Blob{Iov: [][]byte{data}}
	req := scproto.SigmaWriteRequest{Fd: uint32(fd), Blob: blob}
	rep := scproto.SigmaSizeReply{}
	sz, err := scc.rpcSize("SigmaClntSrvAPI.Write", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Write %v %v %v %v", req.Fd, len(data), rep, err)
	return sz, err
}

func (scc *SigmaClntClnt) Seek(fd int, off sp.Toffset) error {
	req := scproto.SigmaSeekRequest{Fd: uint32(fd), Offset: uint64(off)}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.Seek", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Seek %v %v %v", req, rep, err)
	return err
}

func (scc *SigmaClntClnt) CreateEphemeral(path string, p sp.Tperm, m sp.Tmode, l sp.TleaseId, f sp.Tfence) (int, error) {
	req := scproto.SigmaCreateRequest{Path: path, Perm: uint32(p), Mode: uint32(m), LeaseId: uint64(l), Fence: f.FenceProto()}
	rep := scproto.SigmaFdReply{}
	fd, err := scc.rpcFd("SigmaClntSrvAPI.CreateEphemeral", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "CreateEphemeral %v %v %v", req, rep, err)
	return fd, err
}

func (scc *SigmaClntClnt) ClntId() sp.TclntId {
	req := scproto.SigmaNullRequest{}
	rep := scproto.SigmaClntIdReply{}
	err := scc.rpcc.RPC("SigmaClntSrvAPI.ClntId", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "ClntId %v %v", rep, err)
	if err != nil {
		return 0
	}
	if rep.Err.TErrCode() != serr.TErrNoError {
		return 0
	}
	return sp.TclntId(rep.ClntId)
}

func (scc *SigmaClntClnt) FenceDir(path string, f sp.Tfence) error {
	req := scproto.SigmaFenceRequest{Path: path, Fence: f.FenceProto()}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcc.RPC("SigmaClntSrvAPI.FenceDir", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "FenceDir %v %v %v", req, rep, err)
	return err
}

func (scc *SigmaClntClnt) WriteFence(fd int, d []byte, f sp.Tfence) (sp.Tsize, error) {
	blob := &rpcproto.Blob{Iov: [][]byte{d}}
	req := scproto.SigmaWriteRequest{Fd: uint32(fd), Blob: blob, Fence: f.FenceProto()}
	rep := scproto.SigmaSizeReply{}
	sz, err := scc.rpcSize("SigmaClntSrvAPI.Write", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "WriteFence %v %v %v", req, rep, err)
	return sz, err
}

func (scc *SigmaClntClnt) WriteRead(fd int, iniov sessp.IoVec, outiov sessp.IoVec) error {
	inblob := rpcproto.NewBlob(iniov)
	outblob := rpcproto.NewBlob(outiov)
	req := scproto.SigmaWriteRequest{Fd: uint32(fd), Blob: inblob, NOutVec: uint32(len(outiov))}
	rep := scproto.SigmaDataReply{Blob: outblob}
	d, err := scc.rpcData("SigmaClntSrvAPI.WriteRead", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "WriteRead %v %v %v %v", req.Fd, len(iniov), len(d), err)
	if err == nil && len(outiov) != len(d) {
		return fmt.Errorf("sigmaclntclnt outiov len wrong: supplied %v != %v returned", len(outiov), len(d))
	}
	copy(outiov, d)
	return err
}

func (scc *SigmaClntClnt) DirWait(fd int) error {
	req := scproto.SigmaReadRequest{Fd: uint32(fd)}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.DirWait", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "DirWait %v %v %v", req, rep, err)
	return err
}

func (scc *SigmaClntClnt) MountTree(addrs sp.Taddrs, tree, mount string) error {
	req := scproto.SigmaMountTreeRequest{Addr: addrs, Tree: tree, Mount: mount}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.MountTree", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "MountTree %v %v %v", req, rep, err)
	return err
}

func (scc *SigmaClntClnt) IsLocalMount(mnt sp.Tmount) (bool, error) {
	req := scproto.SigmaMountRequest{Mount: mnt.TmountProto}
	rep := scproto.SigmaMountReply{}
	err := scc.rpcc.RPC("SigmaClntSrvAPI.IsLocalMount", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "IsLocalMount %v %v %v", req, rep, err)
	if err != nil {
		return false, err
	}
	if rep.Err.TErrCode() != serr.TErrNoError {
		return false, sp.NewErr(rep.Err)
	}
	return rep.Local, nil
}

func (scc *SigmaClntClnt) PathLastMount(pn string) (path.Path, path.Path, error) {
	req := scproto.SigmaPathRequest{Path: pn}
	rep := scproto.SigmaLastMountReply{}
	err := scc.rpcc.RPC("SigmaClntSrvAPI.PathLastMount", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "PathLastMount %v %v %v", req, rep, err)
	if err != nil {
		return nil, nil, err
	}
	if rep.Err.TErrCode() != serr.TErrNoError {
		return nil, nil, sp.NewErr(rep.Err)
	}
	return rep.Path1, rep.Path2, nil
}

func (scc *SigmaClntClnt) GetNamedMount() (sp.Tmount, error) {
	req := scproto.SigmaNullRequest{}
	rep := scproto.SigmaMountReply{}
	err := scc.rpcc.RPC("SigmaClntSrvAPI.GetNamedMount", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "GetNamedMount %v %v %v", req, rep, err)
	if err != nil {
		return sp.NullMount(), nil
	}
	if rep.Err.TErrCode() != serr.TErrNoError {
		return sp.NullMount(), nil
	}
	return sp.Tmount{rep.Mount}, nil
}

func (scc *SigmaClntClnt) NewRootMount(pn, mntname string) error {
	req := scproto.SigmaMountTreeRequest{Tree: pn, Mount: mntname}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.NewRootMount", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "NewRootMount %v %v", req, rep)
	return err
}

func (scc *SigmaClntClnt) Mounts() []string {
	req := scproto.SigmaNullRequest{}
	rep := scproto.SigmaMountsReply{}
	err := scc.rpcc.RPC("SigmaClntSrvAPI.Mounts", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Mounts %v %v %v", req, rep, err)
	if err != nil {
		return nil
	}
	if rep.Err.TErrCode() != serr.TErrNoError {
		return nil
	}
	return rep.Mounts
}

func (scc *SigmaClntClnt) SetLocalMount(mnt *sp.Tmount, port sp.Tport) {
	db.DFatalf("SetLocalMount %v", mnt)
}

func (scc *SigmaClntClnt) Detach(path string) error {
	req := scproto.SigmaPathRequest{Path: path}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.Detach", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Detach %v %v %v", req, rep, err)
	return err
}

func (scc *SigmaClntClnt) Disconnected() bool {
	return scc.disconnected
}

func (scc *SigmaClntClnt) Disconnect(path string) error {
	scc.disconnected = true
	req := scproto.SigmaPathRequest{Path: path}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.Disconnect", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Disconnect %v %v %v", req, rep, err)
	return err
}

func (scc *SigmaClntClnt) Close() error {
	req := scproto.SigmaNullRequest{}
	rep := scproto.SigmaErrReply{}
	err := scc.rpcErr("SigmaClntSrvAPI.Close", &req, &rep)
	db.DPrintf(db.SIGMACLNTCLNT, "Close %v %v %v", req, rep, err)
	scc.close()
	return err
}
