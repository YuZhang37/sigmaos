package container

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/vishvananda/netlink"

	db "sigmaos/debug"
	"sigmaos/fslib"
	sp "sigmaos/sigmap"
)

const (
	SCNETBIN = "/usr/bin/scnet"
)

func RunKernelContainer(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	cmd.Args = append([]string{KERNEL, cmd.Path}, cmd.Args...)
	cmd.Path = path.Join(sp.PRIVILEGED_BIN, "linux/exec-container")

	db.DPrintf(db.CONTAINER, "contain cmd  %v\n", cmd)

	if err := cmd.Start(); err != nil {
		return err
	}

	db.DPrintf(db.CONTAINER, "mkscnet %v\n", cmd.Process.Pid)
	if err := mkScnet(cmd.Process.Pid); err != nil {
		return err
	}
	return nil
}

func mkScnet(pid int) error {
	cmd := exec.Command(SCNETBIN, "up", strconv.Itoa(pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scnet: out: %s, err: %v", out, err)
	}
	return nil
}

func DelScnet(pid int) error {
	cmd := exec.Command(SCNETBIN, "down", strconv.Itoa(pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scnet: out: %s, err: %v", out, err)
	}
	return nil
}

func setupKContainer() error {
	root := sp.UXROOT

	// XXX specialized for named for now
	db.DPrintf(db.CONTAINER, "execContainer: %v\n", os.Args)

	if err := syscall.Sethostname([]byte("sigmaos")); err != nil {
		return err
	}

	// Make a new /proc (XXX should be per realm)
	if err := syscall.Mount(path.Join(root, "proc"), "/proc", "proc", 0, ""); err != nil {
		db.DPrintf(db.CONTAINER, "failed to mount /proc: %v", err)
		return err
	}

	host, _, error := net.SplitHostPort(fslib.Named()[0])
	if error != nil {
		return fmt.Errorf("SplitHostPort: %v", error)
	}
	if err := setupScnet(host); err != nil {
		return err
	}
	return nil
}

func setupScnet(ip string) error {
	db.DPrintf(db.CONTAINER, "SetupScnet %v\n", ip)
	lnk, err := waitScnet()
	if err != nil {
		db.DPrintf(db.CONTAINER, "wait failed err %v\n", err)
		return err
	}
	db.DPrintf(db.CONTAINER, "wait link %v\n", lnk.Attrs().Name)
	if err := confScnet(lnk, ip); err != nil {
		db.DPrintf(db.CONTAINER, "setup failed err %v\n", err)
		return err
	}
	return nil
}

func waitScnet() (netlink.Link, error) {
	const NSEC = 5
	db.DPrintf(db.CONTAINER, "Wait for network interface\n")
	start := time.Now()
	for {
		if time.Since(start) > NSEC*time.Second {
			return nil, fmt.Errorf("failed to find veth interface in %d seconds", NSEC)
		}
		lst, err := netlink.LinkList()
		if err != nil {
			return nil, err
		}
		for _, l := range lst {
			// found "veth" interface
			if l.Type() == "veth" {
				return l, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func confScnet(lnk netlink.Link, ip string) error {
	db.DPrintf(db.CONTAINER, "Setup network interface\n")

	cidr := ip + "/24"

	// up loopback
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("lo interface: %v", err)
	}
	if err := netlink.LinkSetUp(lo); err != nil {
		return fmt.Errorf("up veth: %v", err)
	}

	// up and configure lnk
	if err := netlink.LinkSetUp(lnk); err != nil {
		return fmt.Errorf("up veth: %v", err)
	}
	addr, err := netlink.ParseAddr(cidr)
	if err != nil {
		return fmt.Errorf("ParseAddr: %v", err)
	}
	db.DPrintf(db.CONTAINER, "addr lnk %v %v\n", addr, lnk.Attrs().Name)
	if err := netlink.AddrAdd(lnk, addr); err != nil {
		return err
	}
	i, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("ParseCIDR %v error %v", cidr, err)
	}
	gw := i.To4()
	gw[3] = 1
	dr := netlink.Route{Gw: gw, Dst: nil}
	if err := netlink.RouteAdd(&dr); err != nil {
		return err
	}
	db.DPrintf(db.CONTAINER, "route gw %v %v\n", gw, dr)
	return nil
}