package imgresized_test

import (
	"fmt"
	"image/jpeg"
	"os"
	"path"
	"testing"
	"time"

	"github.com/nfnt/resize"
	"github.com/stretchr/testify/assert"

	db "sigmaos/debug"
	"sigmaos/fsetcd"
	"sigmaos/groupmgr"
	"sigmaos/imgresized"
	"sigmaos/proc"
	rd "sigmaos/rand"
	sp "sigmaos/sigmap"
	"sigmaos/test"
)

const (
	IMG_RESIZE_MCPU proc.Tmcpu = 100
	IMG_RESIZE_MEM  proc.Tmem  = 0
)

func TestResizeImg(t *testing.T) {
	fn := "/tmp/thumb.jpeg"

	os.Remove(fn)

	in, err := os.Open("1.jpg")
	assert.Nil(t, err)
	img, err := jpeg.Decode(in)
	assert.Nil(t, err)

	start := time.Now()

	img1 := resize.Resize(160, 0, img, resize.Lanczos3)

	fmt.Printf("resize %v\n", time.Since(start))

	out, err := os.Create(fn)
	assert.Nil(t, err)
	jpeg.Encode(out, img1, nil)
}

func TestResizeProc(t *testing.T) {
	ts := test.NewTstateAll(t)
	in := path.Join(sp.S3, "~local/9ps3/img-save/6.jpg")
	//	in := path.Join(sp.S3, "~local/9ps3/img-save/6.jpg")
	out := path.Join(sp.S3, "~local/9ps3/img/6-thumb-xxx.jpg")
	ts.Remove(out)
	p := proc.NewProc("imgresize", []string{in, out})
	err := ts.Spawn(p)
	assert.Nil(t, err, "Spawn")
	err = ts.WaitStart(p.GetPid())
	assert.Nil(t, err, "WaitStart error")
	status, err := ts.WaitExit(p.GetPid())
	assert.Nil(t, err, "WaitExit error %v", err)
	assert.True(t, status.IsStatusOK(), "WaitExit status error: %v", status)
	ts.Shutdown()
}

type Tstate struct {
	job string
	*test.Tstate
}

func newTstate(t *testing.T) *Tstate {
	ts := &Tstate{}
	ts.Tstate = test.NewTstateAll(t)
	ts.job = rd.String(4)
	return ts
}

func (ts *Tstate) WaitDone(t int) error {
	for true {
		time.Sleep(1 * time.Second)
		if n, err := imgresized.NTaskDone(ts.SigmaClnt.FsLib, ts.job); err != nil {
			return err
		} else if n == t {
			break
		} else {
			fmt.Printf("%d..", n)
		}
	}
	fmt.Printf("\n")
	return nil
}

func TestImgdOne(t *testing.T) {
	ts := newTstate(t)

	err := imgresized.MkDirs(ts.SigmaClnt.FsLib, ts.job)
	assert.Nil(t, err)

	fn := path.Join(sp.S3, "~local/9ps3/img-save/1.jpg")
	ts.Remove(imgresized.ThumbName(fn))

	err = imgresized.SubmitTask(ts.SigmaClnt.FsLib, ts.job, fn)
	assert.Nil(t, err)

	imgd := imgresized.StartImgd(ts.SigmaClnt, ts.job, IMG_RESIZE_MCPU, IMG_RESIZE_MEM, false, 1)

	ts.WaitDone(1)

	err = imgresized.SubmitTask(ts.SigmaClnt.FsLib, ts.job, imgresized.STOP)
	assert.Nil(t, err)

	imgd.Wait()

	ts.Shutdown()
}

func TestImgdMany(t *testing.T) {
	ts := newTstate(t)

	err := imgresized.MkDirs(ts.SigmaClnt.FsLib, ts.job)
	assert.Nil(t, err)

	imgd := imgresized.StartImgd(ts.SigmaClnt, ts.job, IMG_RESIZE_MCPU, IMG_RESIZE_MEM, false, 1)

	sts, err := ts.GetDir(path.Join(sp.S3, "~local/9ps3/img-save"))
	assert.Nil(t, err)

	for _, st := range sts {
		fn := path.Join(sp.S3, "~local/9ps3/img-save/", st.Name)
		ts.Remove(imgresized.ThumbName(fn))
	}

	sts, err = ts.GetDir(path.Join(sp.S3, "~local/9ps3/img-save"))
	assert.Nil(t, err)

	for _, st := range sts {
		fmt.Printf("submit %v\n", st.Name)
		fn := path.Join(sp.S3, "~local/9ps3/img-save/", st.Name)
		err = imgresized.SubmitTask(ts.SigmaClnt.FsLib, ts.job, fn)
		assert.Nil(t, err)
	}

	ts.WaitDone(len(sts))

	err = imgresized.SubmitTask(ts.SigmaClnt.FsLib, ts.job, imgresized.STOP)
	assert.Nil(t, err)

	imgd.Wait()

	ts.Shutdown()
}

func TestImgdRestart(t *testing.T) {
	ts := newTstate(t)

	err := imgresized.MkDirs(ts.SigmaClnt.FsLib, ts.job)
	assert.Nil(t, err)

	fn := path.Join(sp.S3, "~local/9ps3/img-save/1.jpg")
	ts.Remove(imgresized.ThumbName(fn))

	err = imgresized.SubmitTask(ts.SigmaClnt.FsLib, ts.job, fn)
	assert.Nil(t, err)

	imgd := imgresized.StartImgd(ts.SigmaClnt, ts.job, IMG_RESIZE_MCPU, IMG_RESIZE_MEM, true, 1)

	go func() {
		err := ts.WaitDone(1)
		if err != nil {
			db.DPrintf(db.TEST, "WaitDone err %v\n", err)
		}
	}()

	time.Sleep(2 * time.Second)

	imgd.Stop()

	ts.Shutdown()

	time.Sleep(2 * fsetcd.LeaseTTL * time.Second)

	db.DPrintf(db.TEST, "Restart")

	ts.Tstate = test.NewTstateAll(t)

	gms, err := groupmgr.Recover(ts.SigmaClnt)
	assert.Nil(ts.T, err, "Recover")
	assert.Equal(ts.T, 1, len(gms))

	err = ts.WaitDone(1)
	assert.Nil(t, err)

	err = imgresized.SubmitTask(ts.SigmaClnt.FsLib, ts.job, imgresized.STOP)
	assert.Nil(t, err)

	gms[0].Wait()

	ts.Shutdown()
}
