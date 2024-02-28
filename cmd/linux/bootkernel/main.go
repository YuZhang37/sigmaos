package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt"

	"sigmaos/auth"
	"sigmaos/boot"
	db "sigmaos/debug"
	"sigmaos/kernel"
	"sigmaos/keys"
	"sigmaos/netsigma"
	"sigmaos/proc"
	sp "sigmaos/sigmap"
)

func main() {
	if len(os.Args) != 12 {
		db.DFatalf("usage: %v kernelid srvs nameds dbip mongoip overlays reserveMcpu buildTag gvisor pubkey privkey\nprovided:%v", os.Args[0], os.Args)
	}
	db.DPrintf(db.BOOT, "Boot %v", os.Args[1:])
	srvs := strings.Split(os.Args[3], ";")
	overlays, err := strconv.ParseBool(os.Args[6])
	if err != nil {
		db.DFatalf("Error parse overlays: %v", err)
	}
	gvisor, err := strconv.ParseBool(os.Args[9])
	if err != nil {
		db.DFatalf("Error parse gvisor: %v", err)
	}
	masterPubKey, err := auth.NewPublicKey[*jwt.SigningMethodECDSA](jwt.SigningMethodES256, []byte(os.Args[10]))
	if err != nil {
		db.DFatalf("Error NewPublicKey", err)
	}
	masterPrivKey, err := auth.NewPrivateKey[*jwt.SigningMethodECDSA](jwt.SigningMethodES256, []byte(os.Args[11]))
	if err != nil {
		db.DFatalf("Error NewPrivateKey", err)
	}
	param := kernel.Param{
		MasterPubKey:  masterPubKey,
		MasterPrivKey: masterPrivKey,
		KernelID:      os.Args[1],
		Services:      srvs,
		Dbip:          os.Args[4],
		Mongoip:       os.Args[5],
		Overlays:      overlays,
		BuildTag:      os.Args[8],
		GVisor:        gvisor,
	}
	if len(os.Args) >= 8 {
		param.ReserveMcpu = os.Args[7]
	}
	db.DPrintf(db.KERNEL, "param %v\n", param)
	h := sp.SIGMAHOME
	p := os.Getenv("PATH")
	os.Setenv("PATH", p+":"+h+"/bin/kernel:"+h+"/bin/linux:"+h+"/bin/user")
	localIP, err1 := netsigma.LocalIP()
	if err1 != nil {
		db.DFatalf("Error local IP: %v", err1)
	}
	s3secrets, err := auth.GetAWSSecrets(sp.AWS_PROFILE)
	if err != nil {
		db.DFatalf("Failed to load AWS secrets %v", err)
	}
	secrets := map[string]*proc.ProcSecretProto{"s3": s3secrets}
	pe := proc.NewBootProcEnv(sp.NewPrincipal(sp.TprincipalID(param.KernelID), sp.NoToken()), secrets, sp.Tip(os.Args[2]), localIP, localIP, param.BuildTag, param.Overlays)
	proc.SetSigmaDebugPid(pe.GetPID().String())
	// Create an auth server with a constant GetKeyFn, to bootstrap with the
	// initial master key. This auth server should *not* be used long-term. It
	// needs to be replaced with one which queries the namespace for keys once
	// knamed has booted.
	kmgr := keys.NewKeyMgr(keys.WithConstGetKeyFn(auth.PublicKey(masterPubKey)))
	kmgr.AddPrivateKey(auth.SIGMA_DEPLOYMENT_MASTER_SIGNER, masterPrivKey)
	as, err1 := auth.NewAuthSrv[*jwt.SigningMethodECDSA](jwt.SigningMethodES256, auth.SIGMA_DEPLOYMENT_MASTER_SIGNER, sp.NOT_SET, kmgr)
	if err1 != nil {
		db.DFatalf("Error NewAuthSrv: %v", err1)
	}
	if err1 := as.MintAndSetToken(pe); err1 != nil {
		db.DFatalf("Error MintToken: %v", err1)
	}
	if err := boot.BootUp(&param, pe, as); err != nil {
		db.DFatalf("%v: boot %v err %v\n", os.Args[0], os.Args[1:], err)
	}
	os.Exit(0)
}
