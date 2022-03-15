package npcodec

import (
	"fmt"

	np "ulambda/ninep"
)

// Adopted from https://github.com/docker/go-p9p/message.go

func newMsg(typ np.Tfcall) (np.Tmsg, *np.Err) {
	switch typ {
	case np.TTversion:
		return np.Tversion{}, nil
	case np.TRversion:
		return np.Rversion{}, nil
	case np.TTauth:
		return np.Tauth{}, nil
	case np.TRauth:
		return np.Rauth{}, nil
	case np.TTattach:
		return np.Tattach{}, nil
	case np.TRattach:
		return np.Rattach{}, nil
	case np.TRerror:
		return np.Rerror{}, nil
	case np.TTflush:
		return np.Tflush{}, nil
	case np.TRflush:
		return np.Rflush{}, nil
	case np.TTwalk:
		return np.Twalk{}, nil
	case np.TRwalk:
		return np.Rwalk{}, nil
	case np.TTopen:
		return np.Topen{}, nil
	case np.TRopen:
		return np.Ropen{}, nil
	case np.TTcreate:
		return np.Tcreate{}, nil
	case np.TRcreate:
		return np.Rcreate{}, nil
	case np.TTread:
		return np.Tread{}, nil
	case np.TRread:
		return np.Rread{}, nil
	case np.TTwrite:
		return np.Twrite{}, nil
	case np.TRwrite:
		return np.Rwrite{}, nil
	case np.TTclunk:
		return np.Tclunk{}, nil
	case np.TRclunk:
		return np.Rclunk{}, nil // no response body
	case np.TTremove:
		return np.Tremove{}, nil
	case np.TRremove:
		return np.Rremove{}, nil
	case np.TTstat:
		return np.Tstat{}, nil
	case np.TRstat:
		return np.Rstat{}, nil
	case np.TTwstat:
		return np.Twstat{}, nil
	case np.TRwstat:
		return np.Rwstat{}, nil
	case np.TTwatch:
		return np.Twatch{}, nil
	case np.TTread1:
		return np.Tread1{}, nil
	case np.TTwrite1:
		return np.Twrite1{}, nil
	case np.TTrenameat:
		return np.Trenameat{}, nil
	case np.TRrenameat:
		return np.Rrenameat{}, nil
	case np.TTremovefile:
		return np.Tremovefile{}, nil
	case np.TTgetfile:
		return np.Tgetfile{}, nil
	case np.TRgetfile:
		return np.Rgetfile{}, nil
	case np.TTsetfile:
		return np.Tsetfile{}, nil
	case np.TTputfile:
		return np.Tputfile{}, nil
	case np.TTmkfence:
		return np.Tmkfence{}, nil
	case np.TRmkfence:
		return np.Rmkfence{}, nil
	case np.TTregfence:
		return np.Tregfence{}, nil
	case np.TTunfence:
		return np.Tunfence{}, nil
	case np.TTrmfence:
		return np.Trmfence{}, nil
	case np.TTdetach:
		return np.Tdetach{}, nil
	}
	return nil, np.MkErr(np.TErrBadFcall, fmt.Sprintf("unknown type: %v", (uint64)(typ)))
}
