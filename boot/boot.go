package boot

import (
	"os"

	"sigmaos/auth"
	db "sigmaos/debug"
	"sigmaos/kernel"
	"sigmaos/kernelsrv"
	"sigmaos/proc"
)

type Boot struct {
	k *kernel.Kernel
}

// The boot processes enters here
func BootUp(param *kernel.Param, pe *proc.ProcEnv, bootstrapAS auth.AuthSrv) error {
	db.DPrintf(db.KERNEL, "Boot param %v ProcEnv %v env %v", param, pe, os.Environ())
	k, err := kernel.NewKernel(param, pe, bootstrapAS)
	if err != nil {
		return err
	}
	db.DPrintf(db.BOOT, "container %s booted %v\n", os.Args[1], k.Ip())
	if err := kernelsrv.RunKernelSrv(k); err != nil {
		return err
	}
	return nil
}
