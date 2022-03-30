package atomic

import (
	"encoding/json"
	"fmt"

	db "ulambda/debug"
	"ulambda/fslib"
	np "ulambda/ninep"
	"ulambda/rand"
)

func PutFileAtomic(fsl *fslib.FsLib, fname string, perm np.Tperm, data []byte) error {
	tmpName := fname + rand.String(16)
	_, err := fsl.PutFile(tmpName, perm, np.OWRITE, data)
	if err != nil {
		db.DFatalf("MakeFileAtomic %v: %v", tmpName, err)
		return err
	}
	err = fsl.Rename(tmpName, fname)
	if err != nil {
		db.DFatalf("MakeFileAtomic rename %v -> %v: err %v", tmpName, fname, err)
		return err
	}
	return nil
}

func PutFileJsonAtomic(fsl *fslib.FsLib, fname string, perm np.Tperm, i interface{}) error {
	data, err := json.Marshal(i)
	if err != nil {
		return fmt.Errorf("PutFileJsonAtomic marshal err %v", err)
	}
	return PutFileAtomic(fsl, fname, perm, data)
}
