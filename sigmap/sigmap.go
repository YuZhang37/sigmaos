package sigmap

//
// Go structures for sigmap protocol, which is based on 9P.
//

import (
	"fmt"
	"log"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"

	"sigmaos/fcall"
	"sigmaos/path"
)

type Tsize uint32
type Ttag uint16
type Tfid uint32
type Tiounit uint32
type Tperm uint32
type Toffset uint64
type Tlength uint64
type Tgid uint32

func (fid Tfid) String() string {
	if fid == NoFid {
		return "-1"
	}
	return fmt.Sprintf("fid %d", fid)
}

//
// Augmentated types for sigmaOS
//

type Tseqno uint64

// NoSeqno signifies the fcall came from a wire-compatible peer
const NoSeqno Tseqno = ^Tseqno(0)

// Atomically increment pointer and return result
func (n *Tseqno) Next() Tseqno {
	next := atomic.AddUint64((*uint64)(n), 1)
	return Tseqno(next)
}

type Tepoch uint64

const NoEpoch Tepoch = ^Tepoch(0)

func (e Tepoch) String() string {
	return strconv.FormatUint(uint64(e), 16)
}

func String2Epoch(epoch string) (Tepoch, error) {
	e, err := strconv.ParseUint(epoch, 16, 64)
	if err != nil {
		return Tepoch(0), err
	}
	return Tepoch(e), nil
}

//
//  End augmentated types
//

// NoTag is the tag for Tversion and Rversion requests.
const NoTag Ttag = ^Ttag(0)

// NoFid is a reserved fid used in a Tattach request for the afid
// field, that indicates that the client does not wish to authenticate
// this session.
const NoFid Tfid = ^Tfid(0)
const NoOffset Toffset = ^Toffset(0)

// If need more than MaxGetSet, use Open/Read/Close interface
const MAXGETSET Tsize = 1_000_000

type Tpath uint64

func (p Tpath) String() string {
	return strconv.FormatUint(uint64(p), 16)
}

func String2Path(path string) (Tpath, error) {
	p, err := strconv.ParseUint(path, 16, 64)
	if err != nil {
		return Tpath(p), err
	}
	return Tpath(p), nil
}

type Qtype uint32
type TQversion uint32

const NoPath Tpath = ^Tpath(0)
const NoV TQversion = ^TQversion(0)

func VEq(v1, v2 TQversion) bool {
	return v1 == NoV || v1 == v2
}

// A Qid's type field represents the type of a file, the high 8 bits of
// the file's permission.
const (
	QTDIR     Qtype = 0x80 // directories
	QTAPPEND  Qtype = 0x40 // append only files
	QTEXCL    Qtype = 0x20 // exclusive use files
	QTMOUNT   Qtype = 0x10 // mounted channel
	QTAUTH    Qtype = 0x08 // authentication file (afid)
	QTTMP     Qtype = 0x04 // non-backed-up file
	QTSYMLINK Qtype = 0x02
	QTFILE    Qtype = 0x00
)

func (qt Qtype) String() string {
	s := ""
	if qt&QTDIR == QTDIR {
		s += "d"
	}
	if qt&QTAPPEND == QTAPPEND {
		s += "a"
	}
	if qt&QTEXCL == QTEXCL {
		s += "e"
	}
	if qt&QTMOUNT == QTMOUNT {
		s += "m"
	}
	if qt&QTAUTH == QTAUTH {
		s += "auth"
	}
	if qt&QTTMP == QTTMP {
		s += "t"
	}
	if qt&QTSYMLINK == QTSYMLINK {
		s += "s"
	}
	if s == "" {
		s = "f"
	}
	return s
}

func MakeQid(t Qtype, v TQversion, p Tpath) *Tqid {
	return &Tqid{Type: uint32(t), Version: uint32(v), Path: uint64(p)}
}

func MakeQidPerm(perm Tperm, v TQversion, p Tpath) *Tqid {
	return MakeQid(Qtype(perm>>QTYPESHIFT), v, p)
}

func (qid *Tqid) Tversion() TQversion {
	return TQversion(qid.Version)
}

func (qid *Tqid) Tpath() Tpath {
	return Tpath(qid.Path)
}

func (qid *Tqid) Ttype() Qtype {
	return Qtype(qid.Type)
}

type Tmode uint32

// Flags for the mode field in Topen and Tcreate messages
const (
	OREAD   Tmode = 0    // read-only
	OWRITE  Tmode = 0x01 // write-only
	ORDWR   Tmode = 0x02 // read-write
	OEXEC   Tmode = 0x03 // execute (implies OREAD)
	OTRUNC  Tmode = 0x10 // or truncate file first
	OCEXEC  Tmode = 0x20 // or close on exec
	ORCLOSE Tmode = 0x40 // remove on close
	OAPPEND Tmode = 0x80 // append

	// sigmaP extension: a client uses OWATCH to block at the
	// server until a file/directiory is create or removed, or a
	// directory changes.  OWATCH with Tcreate will block if the
	// file exists and a remove() will unblock that create.
	// OWATCH with Open() and a closure will invoke the closure
	// when a client creates or removes the file.  OWATCH on open
	// for a directory and a closure will invoke the closure if
	// the directory changes.
	OWATCH Tmode = OCEXEC // overleads OEXEC; maybe ORCLOSe better?
)

func (m Tmode) String() string {
	return fmt.Sprintf("m %x", uint8(m&0xFF))
}

// Permissions
const (
	DMDIR    Tperm = 0x80000000 // directory
	DMAPPEND Tperm = 0x40000000 // append only file
	DMEXCL   Tperm = 0x20000000 // exclusive use file
	DMMOUNT  Tperm = 0x10000000 // mounted channel
	DMAUTH   Tperm = 0x08000000 // authentication file

	// DMTMP is ephemeral in sigmaP
	DMTMP Tperm = 0x04000000 // non-backed-up file

	DMREAD  = 0x4 // mode bit for read permission
	DMWRITE = 0x2 // mode bit for write permission
	DMEXEC  = 0x1 // mode bit for execute permission

	// 9P2000.u extensions
	// A few are used by ulambda, but not supported in driver/proxy,
	// so ulambda mounts on Linux without these extensions.
	DMSYMLINK   Tperm = 0x02000000
	DMLINK      Tperm = 0x01000000
	DMDEVICE    Tperm = 0x00800000
	DMREPL      Tperm = 0x00400000
	DMNAMEDPIPE Tperm = 0x00200000
	DMSOCKET    Tperm = 0x00100000
	DMSETUID    Tperm = 0x00080000
	DMSETGID    Tperm = 0x00040000
	DMSETVTX    Tperm = 0x00010000
)

const (
	QTYPESHIFT = 24
	TYPESHIFT  = 16
	TYPEMASK   = 0xFF
)

func (p Tperm) IsDir() bool        { return p&DMDIR == DMDIR }
func (p Tperm) IsSymlink() bool    { return p&DMSYMLINK == DMSYMLINK }
func (p Tperm) IsReplicated() bool { return p&DMREPL == DMREPL }
func (p Tperm) IsDevice() bool     { return p&DMDEVICE == DMDEVICE }
func (p Tperm) IsPipe() bool       { return p&DMNAMEDPIPE == DMNAMEDPIPE }
func (p Tperm) IsEphemeral() bool  { return p&DMTMP == DMTMP }
func (p Tperm) IsFile() bool       { return (p>>QTYPESHIFT)&0xFF == 0 }

func (p Tperm) String() string {
	qt := Qtype(p >> QTYPESHIFT)
	return fmt.Sprintf("qt %v qp %x", qt, uint8(p&TYPEMASK))
}

func MkInterval(start, end uint64) *Tinterval {
	return &Tinterval{
		Start: start,
		End:   end,
	}
}

func (iv *Tinterval) Size() Tsize {
	return Tsize(iv.End - iv.Start)
}

// XXX should atoi be uint64?
func (iv *Tinterval) Unmarshal(s string) {
	idxs := strings.Split(s[1:len(s)-1], ", ")
	start, err := strconv.Atoi(idxs[0])
	if err != nil {
		debug.PrintStack()
		log.Fatalf("FATAL unmarshal interval: %v", err)
	}
	iv.Start = uint64(start)
	end, err := strconv.Atoi(idxs[1])
	if err != nil {
		debug.PrintStack()
		log.Fatalf("FATAL unmarshal interval: %v", err)
	}
	iv.End = uint64(end)
}

func (iv *Tinterval) Marshal() string {
	return fmt.Sprintf("[%d, %d)", iv.Start, iv.End)
}

type FcallMsg struct {
	Fc  *Fcall
	Msg fcall.Tmsg
}

func (fcm *FcallMsg) Session() fcall.Tsession {
	return fcall.Tsession(fcm.Fc.Session)
}

func (fcm *FcallMsg) Client() fcall.Tclient {
	return fcall.Tclient(fcm.Fc.Client)
}

func (fcm *FcallMsg) Type() fcall.Tfcall {
	return fcall.Tfcall(fcm.Fc.Type)
}

func (fc *Fcall) Tseqno() Tseqno {
	return Tseqno(fc.Seqno)
}

func (fcm *FcallMsg) Seqno() Tseqno {
	return fcm.Fc.Tseqno()
}

func (fcm *FcallMsg) Tag() Ttag {
	return Ttag(fcm.Fc.Tag)
}

func MakeFenceNull() *Tfence {
	return &Tfence{Fenceid: &Tfenceid{}}
}

func MakeFcallMsgNull() *FcallMsg {
	fc := &Fcall{Received: &Tinterval{}, Fence: MakeFenceNull()}
	return &FcallMsg{fc, nil}
}

func (fi *Tfenceid) Tpath() Tpath {
	return Tpath(fi.Path)
}

func (f *Tfence) Tepoch() Tepoch {
	return Tepoch(f.Epoch)
}

func MakeFcallMsg(msg fcall.Tmsg, cli fcall.Tclient, sess fcall.Tsession, seqno *Tseqno, rcv *Tinterval, f *Tfence) *FcallMsg {
	if rcv == nil {
		rcv = &Tinterval{}
	}
	fcall := &Fcall{
		Type:     uint32(msg.Type()),
		Tag:      0,
		Client:   uint64(cli),
		Session:  uint64(sess),
		Received: rcv,
		Fence:    f,
	}
	if seqno != nil {
		fcall.Seqno = uint64(seqno.Next())
	}
	return &FcallMsg{fcall, msg}
}

func MakeFcallMsgReply(req *FcallMsg, reply fcall.Tmsg) *FcallMsg {
	fm := MakeFcallMsg(reply, fcall.Tclient(req.Fc.Client), fcall.Tsession(req.Fc.Session), nil, nil, MakeFenceNull())
	fm.Fc.Seqno = req.Fc.Seqno
	fm.Fc.Received = req.Fc.Received
	fm.Fc.Tag = req.Fc.Tag
	return fm
}

func (fm *FcallMsg) String() string {
	return fmt.Sprintf("%v t %v s %v seq %v recv %v msg %v f %v", fm.Msg.Type(), fm.Fc.Tag, fm.Fc.Session, fm.Fc.Seqno, fm.Fc.Received, fm.Msg, fm.Fc.Fence)
}

func (fm *FcallMsg) GetType() fcall.Tfcall {
	return fcall.Tfcall(fm.Fc.Type)
}

func (fm *FcallMsg) GetMsg() fcall.Tmsg {
	return fm.Msg
}

func MkRerror(err *fcall.Err) *Rerror {
	return &Rerror{Ename: err.Error()}
}

func MkTwalk(fid, nfid Tfid, p path.Path) *Twalk {
	return &Twalk{Fid: uint32(fid), NewFid: uint32(nfid), Wnames: p}
}

func (w *Twalk) Tfid() Tfid {
	return Tfid(w.Fid)
}

func (w *Twalk) Tnewfid() Tfid {
	return Tfid(w.NewFid)
}

func MkTattach(fid, afid Tfid, uname string, path path.Path) *Tattach {
	return &Tattach{Fid: uint32(fid), Afid: uint32(afid), Uname: uname, Aname: path.String()}
}

func (a *Tattach) Tfid() Tfid {
	return Tfid(a.Fid)
}

func MkTopen(fid Tfid, mode Tmode) *Topen {
	return &Topen{Fid: uint32(fid), Mode: uint32(mode)}
}

func (o *Topen) Tfid() Tfid {
	return Tfid(o.Fid)
}

func (o *Topen) Tmode() Tmode {
	return Tmode(o.Mode)
}

func MkTcreate(fid Tfid, n string, p Tperm, mode Tmode) *Tcreate {
	return &Tcreate{Fid: uint32(fid), Name: n, Perm: uint32(p), Mode: uint32(mode)}
}

func (c *Tcreate) Tfid() Tfid {
	return Tfid(c.Fid)
}

func (c *Tcreate) Tperm() Tperm {
	return Tperm(c.Perm)
}

func (c *Tcreate) Tmode() Tmode {
	return Tmode(c.Mode)
}

func MkReadV(fid Tfid, o Toffset, c Tsize, v TQversion) *TreadV {
	return &TreadV{Fid: uint32(fid), Offset: uint64(o), Count: uint32(c), Version: uint32(v)}
}

func (r *TreadV) Tfid() Tfid {
	return Tfid(r.Fid)
}

func (r *TreadV) Tversion() TQversion {
	return TQversion(r.Version)
}

func (r *TreadV) Toffset() Toffset {
	return Toffset(r.Offset)
}

func (r *TreadV) Tcount() Tsize {
	return Tsize(r.Count)
}

func MkTwriteV(fid Tfid, o Toffset, v TQversion, d []byte) *TwriteV {
	return &TwriteV{Fid: uint32(fid), Offset: uint64(o), Version: uint32(v), Data: d}
}

func (w *TwriteV) Tfid() Tfid {
	return Tfid(w.Fid)
}

func (w *TwriteV) Toffset() Toffset {
	return Toffset(w.Offset)
}

func (w *TwriteV) Tversion() TQversion {
	return TQversion(w.Version)
}

func (wr *Rwrite) Tcount() Tsize {
	return Tsize(wr.Count)
}

func MkTwatch(fid Tfid) *Twatch {
	return &Twatch{Fid: uint32(fid)}
}

func (w *Twatch) Tfid() Tfid {
	return Tfid(w.Fid)
}

func MkTclunk(fid Tfid) *Tclunk {
	return &Tclunk{Fid: uint32(fid)}
}

func (c *Tclunk) Tfid() Tfid {
	return Tfid(c.Fid)
}

func MkTremove(fid Tfid) *Tremove {
	return &Tremove{Fid: uint32(fid)}
}

func (r *Tremove) Tfid() Tfid {
	return Tfid(r.Fid)
}

func MkTstat(fid Tfid) *Tstat {
	return &Tstat{Fid: uint32(fid)}
}

func (s *Tstat) Tfid() Tfid {
	return Tfid(s.Fid)
}

func MkStatNull() *Stat {
	st := &Stat{}
	st.Qid = MakeQid(0, 0, 0)
	return st
}

func MkStat(qid *Tqid, perm Tperm, mtime uint32, name, owner string) *Stat {
	st := &Stat{
		Type:  0, // XXX
		Qid:   qid,
		Mode:  uint32(perm),
		Mtime: mtime,
		Atime: 0,
		Name:  name,
		Uid:   owner,
		Gid:   owner,
		Muid:  "",
	}
	return st

}

func (st *Stat) Tlength() Tlength {
	return Tlength(st.Length)
}

func (st *Stat) Tmode() Tperm {
	return Tperm(st.Mode)
}

func MkTwstat(fid Tfid, st *Stat) *Twstat {
	return &Twstat{Fid: uint32(fid), Stat: st}
}

func (w *Twstat) Tfid() Tfid {
	return Tfid(w.Fid)
}

func MkTrenameat(oldfid Tfid, oldname string, newfid Tfid, newname string) *Trenameat {
	return &Trenameat{OldFid: uint32(oldfid), OldName: oldname, NewFid: uint32(newfid), NewName: newname}
}

func (r *Trenameat) Tnewfid() Tfid {
	return Tfid(r.NewFid)
}

func (r *Trenameat) Toldfid() Tfid {
	return Tfid(r.OldFid)
}

func MkTgetfile(fid Tfid, mode Tmode, offset Toffset, cnt Tsize, path path.Path, resolve bool) *Tgetfile {
	return &Tgetfile{Fid: uint32(fid), Mode: uint32(mode), Offset: uint64(offset), Count: uint32(cnt), Wnames: path, Resolve: resolve}
}

func (g *Tgetfile) Tfid() Tfid {
	return Tfid(g.Fid)
}

func (g *Tgetfile) Tmode() Tmode {
	return Tmode(g.Mode)
}

func (g *Tgetfile) Toffset() Toffset {
	return Toffset(g.Offset)
}

func (g *Tgetfile) Tcount() Tsize {
	return Tsize(g.Count)
}

func MkTsetfile(fid Tfid, mode Tmode, offset Toffset, path path.Path, resolve bool, d []byte) *Tsetfile {
	return &Tsetfile{Fid: uint32(fid), Mode: uint32(mode), Offset: uint64(offset), Wnames: path, Resolve: resolve, Data: d}
}

func (s *Tsetfile) Tfid() Tfid {
	return Tfid(s.Fid)
}

func (s *Tsetfile) Tmode() Tmode {
	return Tmode(s.Mode)
}

func (s *Tsetfile) Toffset() Toffset {
	return Toffset(s.Offset)
}

func MkTputfile(fid Tfid, mode Tmode, perm Tperm, offset Toffset, path path.Path, d []byte) *Tputfile {
	return &Tputfile{Fid: uint32(fid), Mode: uint32(mode), Perm: uint32(perm), Offset: uint64(offset), Wnames: path, Data: d}
}

func (p *Tputfile) Tfid() Tfid {
	return Tfid(p.Fid)
}

func (p *Tputfile) Tmode() Tmode {
	return Tmode(p.Mode)
}

func (p *Tputfile) Tperm() Tperm {
	return Tperm(p.Perm)
}

func (p *Tputfile) Toffset() Toffset {
	return Toffset(p.Offset)
}

func MkTremovefile(fid Tfid, path path.Path, r bool) *Tremovefile {
	return &Tremovefile{Fid: uint32(fid), Wnames: path, Resolve: r}
}

func (r *Tremovefile) Tfid() Tfid {
	return Tfid(r.Fid)
}

func MkTheartbeat(sess []uint64) *Theartbeat {
	return &Theartbeat{Sids: sess}
}

func MkTdetach(pid, lid uint32) *Tdetach {
	return &Tdetach{PropId: pid, LeadId: lid}
}

func (Tversion) Type() fcall.Tfcall { return fcall.TTversion }
func (Rversion) Type() fcall.Tfcall { return fcall.TRversion }
func (Tauth) Type() fcall.Tfcall    { return fcall.TTauth }
func (Rauth) Type() fcall.Tfcall    { return fcall.TRauth }
func (Tattach) Type() fcall.Tfcall  { return fcall.TTattach }
func (Rattach) Type() fcall.Tfcall  { return fcall.TRattach }
func (Rerror) Type() fcall.Tfcall   { return fcall.TRerror }
func (Twalk) Type() fcall.Tfcall    { return fcall.TTwalk }
func (Rwalk) Type() fcall.Tfcall    { return fcall.TRwalk }
func (Topen) Type() fcall.Tfcall    { return fcall.TTopen }
func (Twatch) Type() fcall.Tfcall   { return fcall.TTwatch }
func (Ropen) Type() fcall.Tfcall    { return fcall.TRopen }
func (Tcreate) Type() fcall.Tfcall  { return fcall.TTcreate }
func (Rcreate) Type() fcall.Tfcall  { return fcall.TRcreate }
func (Rread) Type() fcall.Tfcall    { return fcall.TRread }
func (Rwrite) Type() fcall.Tfcall   { return fcall.TRwrite }
func (Tclunk) Type() fcall.Tfcall   { return fcall.TTclunk }
func (Rclunk) Type() fcall.Tfcall   { return fcall.TRclunk }
func (Tremove) Type() fcall.Tfcall  { return fcall.TTremove }
func (Rremove) Type() fcall.Tfcall  { return fcall.TRremove }
func (Tstat) Type() fcall.Tfcall    { return fcall.TTstat }
func (Twstat) Type() fcall.Tfcall   { return fcall.TTwstat }
func (Rwstat) Type() fcall.Tfcall   { return fcall.TRwstat }

//
// sigmaP
//

func (Rstat) Type() fcall.Tfcall       { return fcall.TRstat }
func (TreadV) Type() fcall.Tfcall      { return fcall.TTreadV }
func (TwriteV) Type() fcall.Tfcall     { return fcall.TTwriteV }
func (Trenameat) Type() fcall.Tfcall   { return fcall.TTrenameat }
func (Rrenameat) Type() fcall.Tfcall   { return fcall.TRrenameat }
func (Tremovefile) Type() fcall.Tfcall { return fcall.TTremovefile }
func (Tgetfile) Type() fcall.Tfcall    { return fcall.TTgetfile }
func (Tsetfile) Type() fcall.Tfcall    { return fcall.TTsetfile }
func (Tputfile) Type() fcall.Tfcall    { return fcall.TTputfile }
func (Tdetach) Type() fcall.Tfcall     { return fcall.TTdetach }
func (Rdetach) Type() fcall.Tfcall     { return fcall.TRdetach }
func (Theartbeat) Type() fcall.Tfcall  { return fcall.TTheartbeat }
func (Rheartbeat) Type() fcall.Tfcall  { return fcall.TRheartbeat }
func (Twriteread) Type() fcall.Tfcall  { return fcall.TTwriteread }