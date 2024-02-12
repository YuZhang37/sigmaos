// Package bootkernelclnt starts a SigmaOS kernel
package bootkernelclnt

import (
	"os"
	"os/exec"
	"path"

	"sigmaos/auth"
	db "sigmaos/debug"
	"sigmaos/kernelclnt"
	"sigmaos/proc"
	"sigmaos/rand"
	"sigmaos/sigmaclnt"
	sp "sigmaos/sigmap"
)

// Shell script that starts the sigmaos container, which invokes Start
// of [bootkernel]
const (
	START     = "../start-kernel.sh"
	K_OUT_DIR = "/tmp/sigmaos-kernel-start-logs"
)

func Start(kernelId string, pcfg *proc.ProcEnv, srvs string, overlays, gvisor bool, masterPubKey auth.PublicKey, masterPrivKey auth.PrivateKey) (string, error) {
	args := []string{
		"--pull", pcfg.BuildTag,
		"--boot", srvs,
		"--named", pcfg.EtcdIP,
		"--pubkey", masterPubKey.String(),
		"--privkey", masterPrivKey.String(),
		"--host",
	}
	db.DPrintf(db.ALWAYS, "pub %v priv %v", masterPubKey.String(), masterPrivKey.String())
	if overlays {
		args = append(args, "--overlays")
	}
	if gvisor {
		args = append(args, "--gvisor")
	}
	args = append(args, kernelId)
	// Ensure the kernel output directory has been created
	os.Mkdir(K_OUT_DIR, 0777)
	ofilePath := path.Join(K_OUT_DIR, kernelId+".stdout")
	ofile, err := os.Create(ofilePath)
	if err != nil {
		db.DPrintf(db.ERROR, "Create out file %v", ofilePath)
		return "", err
	}
	defer ofile.Close()
	efilePath := path.Join(K_OUT_DIR, kernelId+".stderr")
	efile, err := os.Create(efilePath)
	if err != nil {
		db.DPrintf(db.ERROR, "Create out file %v", ofilePath)
		return "", err
	}
	defer efile.Close()
	// Create the command struct and set stdout/stderr
	cmd := exec.Command(START, args...)
	cmd.Stdout = ofile
	cmd.Stderr = efile
	if err := cmd.Run(); err != nil {
		db.DPrintf(db.BOOT, "Boot: run err %v", err)
		return "", err
	}
	if err := ofile.Sync(); err != nil {
		db.DPrintf(db.ERROR, "Sync out file %v: %v", ofilePath, err)
		return "", err
	}
	out, err := os.ReadFile(ofilePath)
	if err != nil {
		db.DPrintf(db.ERROR, "Read out file %v: %v", ofilePath, err)
		return "", err
	}
	ip := string(out)
	db.DPrintf(db.BOOT, "Start: %v srvs %v IP %v overlays %v gvisor %v", kernelId, srvs, ip, overlays, gvisor)
	return ip, nil
}

func GenKernelId() string {
	return "sigma-" + rand.String(4)
}

type Kernel struct {
	*sigmaclnt.SigmaClnt
	kernelId string
	kclnt    *kernelclnt.KernelClnt
}

func NewKernelClntStart(pcfg *proc.ProcEnv, conf string, overlays, gvisor bool, masterPubKey auth.PublicKey, masterPrivKey auth.PrivateKey) (*Kernel, error) {
	kernelId := GenKernelId()
	_, err := Start(kernelId, pcfg, conf, overlays, gvisor, masterPubKey, masterPrivKey)
	if err != nil {
		return nil, err
	}
	return NewKernelClnt(kernelId, pcfg)
}

func NewKernelClnt(kernelId string, pcfg *proc.ProcEnv) (*Kernel, error) {
	db.DPrintf(db.SYSTEM, "NewKernelClnt %s\n", kernelId)
	sc, err := sigmaclnt.NewSigmaClntRootInit(pcfg)
	if err != nil {
		db.DPrintf(db.ALWAYS, "NewKernelClnt sigmaclnt err %v", err)
		return nil, err
	}
	pn := sp.BOOT + kernelId
	if kernelId == "" {
		var pn1 string
		var err error
		if pcfg.EtcdIP != pcfg.GetOuterContainerIP().String() {
			// If running in a distributed setting, bootkernel clnt can be ~any
			pn1, _, err = sc.ResolveMount(sp.BOOT + "~any")
		} else {
			pn1, _, err = sc.ResolveMount(sp.BOOT + "~local")
		}
		if err != nil {
			db.DPrintf(db.ALWAYS, "Error resolve local")
			return nil, err
		}
		pn = pn1
		kernelId = path.Base(pn)
	}

	db.DPrintf(db.SYSTEM, "NewKernelClnt %s %s\n", pn, kernelId)
	kclnt, err := kernelclnt.NewKernelClnt(sc.FsLib, pn)
	if err != nil {
		db.DPrintf(db.ALWAYS, "Error NewKernelClnt %v", err)
		return nil, err
	}

	return &Kernel{sc, kernelId, kclnt}, nil
}

func (k *Kernel) NewSigmaClnt(pcfg *proc.ProcEnv) (*sigmaclnt.SigmaClnt, error) {
	return sigmaclnt.NewSigmaClntRootInit(pcfg)
}

func (k *Kernel) Shutdown() error {
	db.DPrintf(db.SYSTEM, "Shutdown kernel %s", k.kernelId)
	err := k.kclnt.Shutdown()
	db.DPrintf(db.SYSTEM, "Shutdown kernel %s err %v", k.kernelId, err)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kernel) Boot(s string) error {
	_, err := k.kclnt.Boot(s, []string{})
	return err
}

func (k *Kernel) Kill(s string) error {
	return k.kclnt.Kill(s)
}

func (k *Kernel) KernelId() string {
	return k.kernelId
}
