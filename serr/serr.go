package serr

import (
	"errors"
	"fmt"
	"strings"

	"sigmaos/path"
)

type Terror uint32

const (
	TErrBadattach Terror = iota + 1
	TErrBadoffset
	TErrBadcount
	TErrBotch
	TErrCreatenondir
	TErrDupfid
	TErrDuptag
	TErrIsdir
	TErrNocreate
	TErrNomem
	TErrNoremove
	TErrNostat
	TErrNotfound
	TErrNowrite
	TErrNowstat
	TErrPerm
	TErrUnknownfid
	TErrBaddir
	TErrWalknodir

	//
	// sigma protocol errors
	//

	TErrUnreachable
	TErrNotSupported
	TErrInval
	TErrUnknownMsg
	TErrNotDir
	TErrNotFile
	TErrNotSymlink
	TErrNotEmpty
	TErrVersion
	TErrStale
	TErrExists
	TErrClosed // for closed sessions and pipes.
	TErrBadFcall

	//
	// sigma OS errors
	//

	TErrRetry
	TErrBadFd

	//
	// To propagate non-sigma errors.
	// Must be *last* for String2Err()
	//
	TErrError
)

func (err Terror) String() string {
	switch err {
	case TErrBadattach:
		return "unknown specifier in attach"
	case TErrBadoffset:
		return "bad offset"
	case TErrBadcount:
		return "bad count"
	case TErrBotch:
		return "9P protocol botch"
	case TErrCreatenondir:
		return "create in non-directory"
	case TErrDupfid:
		return "duplicate fid"
	case TErrDuptag:
		return "duplicate tag"
	case TErrIsdir:
		return "is a directory"
	case TErrNocreate:
		return "create prohibited"
	case TErrNomem:
		return "out of memory"
	case TErrNoremove:
		return "remove prohibited"
	case TErrNostat:
		return "stat prohibited"
	case TErrNotfound:
		return "file not found"
	case TErrNowrite:
		return "write prohibited"
	case TErrNowstat:
		return "wstat prohibited"
	case TErrPerm:
		return "permission denied"
	case TErrUnknownfid:
		return "unknown fid"
	case TErrBaddir:
		return "bad directory in wstat"
	case TErrWalknodir:
		return "walk in non-directory"

	// sigma
	case TErrUnreachable:
		return "Unreachable"
	case TErrNotSupported:
		return "not supported"
	case TErrInval:
		return "invalid argument"
	case TErrUnknownMsg:
		return "unknown message"
	case TErrNotDir:
		return "not a directory"
	case TErrNotFile:
		return "not a file"
	case TErrNotSymlink:
		return "not a symlink"
	case TErrNotEmpty:
		return "not empty"
	case TErrVersion:
		return "version mismatch"
	case TErrStale:
		return "stale fence"
	case TErrExists:
		return "file exists"
	case TErrClosed:
		return "closed"
	case TErrBadFcall:
		return "bad fcall"

	// sigma OS errors
	case TErrRetry:
		return "retry"
	case TErrBadFd:
		return "Bad fd"

	// for passing non-sigma errors through
	case TErrError:
		return "Non-sigma error"

	default:
		return "unknown error"
	}
}

type Err struct {
	ErrCode Terror
	Obj     string
	Err     error
}

func MkErr(err Terror, obj interface{}) *Err {
	return &Err{err, fmt.Sprintf("%v", obj), nil}
}

func MkErrError(error error) *Err {
	return &Err{TErrError, "", error}
}

func (err *Err) Code() Terror {
	return err.ErrCode
}

func (err *Err) Unwrap() error { return err.Err }

func (err *Err) Error() string {
	return fmt.Sprintf("%v %v %v", err.ErrCode, err.Obj, err.Err)
}

func (err *Err) String() string {
	return err.Error()
}

// SigmaOS server couldn't find the requested file
func IsErrNotfound(error error) bool {
	return error != nil && strings.Contains(error.Error(), TErrNotfound.String())
}

// SigmaOS server couldn't reach a server
func IsErrUnreachable(error error) bool {
	return error != nil && strings.HasPrefix(error.Error(), TErrUnreachable.String())
}

// A file is unavailable: either a server on the file's path is
// unreachable or the file is not found
func IsErrUnavailable(error error) bool {
	return IsErrUnreachable(error) || IsErrNotfound(error)
}

func ErrPath(error error) string {
	if IsErrNotfound(error) {
		return strings.TrimPrefix(error.Error(), TErrNotfound.String()+" ")
	} else if IsErrUnreachable(error) {
		return strings.TrimPrefix(error.Error(), TErrUnreachable.String()+" ")
	} else {
		return ""
	}
}

func IsErrExists(error error) bool {
	return strings.Contains(error.Error(), TErrExists.String())
}

func IsErrStale(error error) bool {
	return strings.HasPrefix(error.Error(), TErrStale.String())
}

func IsErrVersion(error error) bool {
	return strings.HasPrefix(error.Error(), TErrVersion.String())
}

func IsErrRetry(error error) bool {
	return strings.HasPrefix(error.Error(), TErrRetry.String())
}

func IsErrSessClosed(error error) bool {
	return strings.HasPrefix(error.Error(), TErrClosed.String()) && strings.Contains(error.Error(), "sess")
}

func IsErrPipeClosed(error error) bool {
	return strings.HasPrefix(error.Error(), TErrClosed.String()) && strings.Contains(error.Error(), "pipe")
}

func IsErrNotDir(error error) bool {
	return strings.HasPrefix(error.Error(), TErrNotDir.String())
}

// Maybe the error is because of a symlink or ~
func IsMaybeSpecialElem(error error) bool {
	return IsErrNotDir(error) || IsErrUnionElem(error)
}

func IsErrUnionElem(error error) bool {
	return IsErrNotfound(error) && path.IsUnionElem(ErrPath(error))
}

func IsErrCode(error error, code Terror) bool {
	var err *Err
	if errors.As(error, &err) {
		return err.Code() == code
	}
	return false
}