package netsigma

import (
	"fmt"
	"net"
	"runtime/debug"
	"strings"

	db "sigmaos/debug"
	sp "sigmaos/sigmap"
)

// Rearrange addrs so that first addr is in the realm as clnt.
func Rearrange(clntnet string, addrs sp.Taddrs) sp.Taddrs {
	if len(addrs) == 1 {
		return addrs
	}
	raddrs := make(sp.Taddrs, len(addrs))
	for i := 0; i < len(addrs); i++ {
		raddrs[i] = addrs[i]
	}
	p := -1
	l := -1
	for i, a := range raddrs {
		if a.NetNS == clntnet {
			l = i
			break
		}
		if a.NetNS == sp.ROOTREALM.String() && p < 0 {
			p = i
		}
	}
	if l >= 0 {
		swap(raddrs, l)
	} else if p >= 0 {
		swap(raddrs, p)
	}
	return raddrs
}

func swap(addrs sp.Taddrs, i int) sp.Taddrs {
	v := addrs[0]
	addrs[0] = addrs[i]
	addrs[i] = v
	return addrs
}

func QualifyAddr(addrstr string) (sp.Tip, sp.Tport, error) {
	return QualifyAddrLocalIP("", addrstr)
}

func QualifyAddrLocalIP(lip sp.Tip, addrstr string) (sp.Tip, sp.Tport, error) {
	h, pstr, err := net.SplitHostPort(addrstr)
	if err != nil {
		db.DFatalf("Err split host port %v: %v", addrstr, err)
		return sp.NO_IP, sp.NO_PORT, err
	}
	p, err := sp.ParsePort(pstr)
	if err != nil {
		db.DFatalf("Err split host port %v: %v", addrstr, err)
		return sp.NO_IP, sp.NO_PORT, err
	}
	var host sp.Tip = lip
	var port sp.Tport = p
	if h == "::" {
		if lip == "" {
			ip, err := LocalIP()
			if err != nil {
				db.DFatalf("LocalIP \"%v\" %v", addrstr, err)
				return sp.NO_IP, sp.NO_PORT, err
			}
			host = ip
		}
	}
	return host, port, nil
}

// XXX deduplicate with localIP
func LocalInterface() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsLoopback() {
				continue
			}
			if ip.To4() == nil {
				continue
			}
			return i.Name, nil
		}
	}
	return "", fmt.Errorf("localInterface: not found")
}

func localIPs() ([]net.IP, error) {
	var ips []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		db.DFatalf("Err Get net interfaces %v: %v\n%s", ifaces, err, debug.Stack())
		return nil, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsLoopback() {
				continue
			}
			if ip.To4() == nil {
				continue
			}
			ips = append(ips, ip)
		}
	}
	return ips, nil
}

// XXX should find what outgoing ip is
func LocalIP() (sp.Tip, error) {
	ips, err := localIPs()
	if err != nil {
		return "", err
	}

	// if we have a local ip in 10.10.x.x (for Cloudlab), prioritize that first
	for _, i := range ips {
		if strings.HasPrefix(i.String(), "10.10.") {
			return sp.Tip(i.String()), nil
		}
		if !strings.HasPrefix(i.String(), "127.") {
			return sp.Tip(i.String()), nil
		}
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("LocalIP: no IP")
	}

	return sp.Tip(ips[len(ips)-1].String()), nil
}
