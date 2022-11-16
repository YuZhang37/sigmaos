package protdevclnt

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"sigmaos/fslib"
	np "sigmaos/ninep"
	"sigmaos/protdevsrv"
)

type ProtDevClnt struct {
	*fslib.FsLib
	sid string
	fn  string
}

func MkProtDevClnt(fsl *fslib.FsLib, fn string) (*ProtDevClnt, error) {
	pdc := &ProtDevClnt{}
	pdc.FsLib = fsl
	pdc.fn = fn
	b, err := pdc.GetFile(pdc.fn + "/clone")
	if err != nil {
		return nil, fmt.Errorf("Clone err %v\n", err)
	}
	pdc.sid = "/" + string(b)
	return pdc, nil
}

func (pdc *ProtDevClnt) rpc(method string, a []byte) (*protdevsrv.Reply, error) {
	req := &protdevsrv.Request{method, a}

	ab := new(bytes.Buffer)
	ae := gob.NewEncoder(ab)
	if err := ae.Encode(req); err != nil {
		return nil, err
	}
	_, err := pdc.SetFile(pdc.fn+pdc.sid+"/data", ab.Bytes(), np.OWRITE, 0)
	if err != nil {
		return nil, fmt.Errorf("rpc err %v\n", err)
	}
	// XXX maybe the caller should use Reader
	// or maybe have two interfaces: rpc and request+response?
	b, err := pdc.GetFile(pdc.fn + pdc.sid + "/data")
	if err != nil {
		return nil, fmt.Errorf("Query response err %v", err)
	}
	rep := &protdevsrv.Reply{}
	rb := bytes.NewBuffer(b)
	re := gob.NewDecoder(rb)
	if err := re.Decode(rep); err != nil {
		return nil, err
	}
	return rep, nil
}

func (pdc *ProtDevClnt) RPC(method string, arg any, res any) error {
	ab := new(bytes.Buffer)
	ae := gob.NewEncoder(ab)
	if err := ae.Encode(arg); err != nil {
		return err
	}
	rep, err := pdc.rpc(method, ab.Bytes())
	if err != nil {
		return err
	}
	if rep.Error != "" {
		return fmt.Errorf("%s", rep.Error)
	}
	rb := bytes.NewBuffer(rep.Res)
	rd := gob.NewDecoder(rb)
	if err := rd.Decode(res); err != nil {
		return err
	}
	return nil
}
