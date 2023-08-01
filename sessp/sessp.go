package sessp

import (
	"fmt"
	"log"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
)

type Tfcall uint8
type Ttag uint16
type Tsize uint32

type Tsession uint64
type Tseqno uint64
type Tclient uint64

// NoTag is the tag for Tversion and Rversion requests.
const NoTag Ttag = ^Ttag(0)

// NoSeqno signifies the fcall came from a wire-compatible peer
const NoSeqno Tseqno = ^Tseqno(0)

// Atomically increment pointer and return result
func (n *Tseqno) Next() Tseqno {
	next := atomic.AddUint64((*uint64)(n), 1)
	return Tseqno(next)
}

// NoSession signifies the fcall came from a wire-compatible peer
const NoSession Tsession = ^Tsession(0)

func (s Tsession) String() string {
	return strconv.FormatUint(uint64(s), 16)
}

type Tmsg interface {
	Type() Tfcall
}

type FcallMsg struct {
	Fc   *Fcall
	Msg  Tmsg
	Data []byte
}

func (fcm *FcallMsg) Session() Tsession {
	return Tsession(fcm.Fc.Session)
}

func (fcm *FcallMsg) Client() Tclient {
	return Tclient(fcm.Fc.Client)
}

func (fcm *FcallMsg) Type() Tfcall {
	return Tfcall(fcm.Fc.Type)
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

func MakeFcallMsgNull() *FcallMsg {
	fc := &Fcall{Received: &Tinterval{}}
	return &FcallMsg{fc, nil, nil}
}

func MakeFcallMsg(msg Tmsg, data []byte, cli Tclient, sess Tsession, seqno *Tseqno, rcv Tinterval) *FcallMsg {
	fcall := &Fcall{
		Type:     uint32(msg.Type()),
		Tag:      0,
		Client:   uint64(cli),
		Session:  uint64(sess),
		Received: &rcv,
	}
	if seqno != nil {
		fcall.Seqno = uint64(seqno.Next())
	}
	return &FcallMsg{fcall, msg, data}
}

func MakeFcallMsgReply(req *FcallMsg, reply Tmsg) *FcallMsg {
	fm := MakeFcallMsg(reply, nil, Tclient(req.Fc.Client), Tsession(req.Fc.Session), nil, Tinterval{})
	fm.Fc.Seqno = req.Fc.Seqno
	fm.Fc.Received = req.Fc.Received
	fm.Fc.Tag = req.Fc.Tag
	return fm
}

func (fm *FcallMsg) String() string {
	return fmt.Sprintf("%v t %v s %v seq %v recv %v msg %v", fm.Msg.Type(), fm.Fc.Tag, fm.Fc.Session, fm.Fc.Seqno, fm.Fc.Received, fm.Msg)
}

func (fm *FcallMsg) GetType() Tfcall {
	return Tfcall(fm.Fc.Type)
}

func (fm *FcallMsg) GetMsg() Tmsg {
	return fm.Msg
}

func MkInterval(start, end uint64) *Tinterval {
	return &Tinterval{
		Start: start,
		End:   end,
	}
}

func (iv0 Tinterval) Eq(iv1 *Tinterval) bool {
	return iv0.Start == iv1.Start && iv0.End == iv1.End
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

type IIntervals interface {
	String() string
	Delete(*Tinterval)
	Insert(*Tinterval)
	Length() int
	Contains(uint64) bool
	Present(*Tinterval) bool
	Find(*Tinterval) *Tinterval
	Pop() Tinterval
	Deepcopy(IIntervals)
}

const (

	//
	// 9P
	//

	TTversion Tfcall = iota + 100
	TRversion
	TTauth
	TRauth
	TTattach9P
	TRattach
	TTerror
	TRerror9P
	TTflush
	TRflush
	TTwalk
	TRwalk
	TTopen9P
	TRopen
	TTcreate9P
	TRcreate
	TTread
	TRread9P
	TTwrite
	TRwrite
	TTclunk
	TRclunk
	TTremove
	TRremove
	TTstat
	TRstat9P
	TTwstat9P
	TRwstat

	//
	// SigmaP
	//
	TTattach
	TRerror
	TTopen
	TTcreate
	TTreadV
	TRread
	TTwriteV
	TTwatch
	TRstat
	TTwstat
	TTrenameat
	TRrenameat
	TTremovefile
	TTgetfile
	TTputfile
	TTdetach
	TRdetach
	TTheartbeat
	TRheartbeat
	TTwriteread
)

func (fct Tfcall) String() string {
	switch fct {
	case TTversion:
		return "Tversion"
	case TRversion:
		return "Rversion"
	case TTauth:
		return "Tauth"
	case TRauth:
		return "Rauth"
	case TTattach9P:
		return "Tattach"
	case TRattach:
		return "Rattach"
	case TRerror9P:
		return "Rerror9P"
	case TTflush:
		return "Tflush"
	case TRflush:
		return "Rflush"
	case TTwalk:
		return "Twalk"
	case TRwalk:
		return "Rwalk"
	case TTopen9P:
		return "Topen9P"
	case TRopen:
		return "Ropen"
	case TTcreate9P:
		return "Tcreate"
	case TRcreate:
		return "Rcreate"
	case TTread:
		return "Tread9P"
	case TRread:
		return "Rread9P"
	case TTwrite:
		return "Twrite9P"
	case TRwrite:
		return "Rwrite9P"
	case TTclunk:
		return "Tclunk"
	case TRclunk:
		return "Rclunk"
	case TTremove:
		return "Tremove"
	case TRremove:
		return "Rremove"
	case TTstat:
		return "Tstat"
	case TRstat9P:
		return "Rstat9p"
	case TTwstat9P:
		return "Twstat9p"
	case TRwstat:
		return "Rwstat"

	case TTattach:
		return "Tattach"
	case TRerror:
		return "Rerror"
	case TTopen:
		return "Ropen"
	case TTcreate:
		return "Tcreate"
	case TTreadV:
		return "TreadV"
	case TTwriteV:
		return "TwriteV"
	case TRstat:
		return "Rstat"
	case TTwstat:
		return "Tstat"
	case TTwatch:
		return "Twatch"
	case TTrenameat:
		return "Trenameat"
	case TRrenameat:
		return "Rrenameat"
	case TTremovefile:
		return "Tremovefile"
	case TTgetfile:
		return "Tgetfile"
	case TTputfile:
		return "Tputfile"
	case TTdetach:
		return "Tdetach"
	case TRdetach:
		return "Rdetach"
	case TTheartbeat:
		return "Theartbeat"
	case TRheartbeat:
		return "Rheartbeat"
	case TTwriteread:
		return "Twriteread"
	default:
		return "Tunknown"
	}
}
