package openstack

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	"github.com/ovh/cds/sdk/log"
)

// for each ip in the range, look for the first free ones
func (h *HatcheryCloud) findAvailableIP(workerName string) (string, error) {
	srvs := h.getServers()

	ipsInfos.mu.Lock()
	defer ipsInfos.mu.Unlock()
	for ip, infos := range ipsInfos.ips {
		// ip used less than 10s
		// 10s max to display worker name on nova list after call create
		if time.Since(infos.dateLastBooked) < 10*time.Second {
			continue
		}
		found := false
		for _, srv := range srvs {
			if infos.workerName == srv.Name {
				found = true
			}
		}
		if !found {
			infos.workerName = ""
			ipsInfos.ips[ip] = infos
		}
	}

	freeIP := []string{}
	for ip := range ipsInfos.ips {
		free := true
		if ipsInfos.ips[ip].workerName != "" {
			continue // ip already used by a worker
		}
		for _, s := range srvs {
			if len(s.Addresses) == 0 {
				continue
			}
			all, errap := servers.ListAddressesByNetwork(h.client, s.ID, h.networkString).AllPages()
			if errap != nil {
				// check not found -> this append when a srv is deleted
				if !strings.Contains(errap.Error(), "Resource not found") {
					return "", fmt.Errorf("findAvailableIP> error on pager.AllPages with server.ID:%s, err:%s ", s.ID, errap)
				}
				continue
			}
			addrs, erren := servers.ExtractNetworkAddresses(all)
			if erren != nil {
				if !strings.Contains(errap.Error(), "Resource not found") {
					log.Error("findAvailableIP> error on ExtractNetworkAddresses with server.ID:%s, err:%s", s.ID, erren)
				}
				continue
			}
			for _, a := range addrs {
				if a.Address == ip {
					free = false
				}
			}
			if !free {
				break
			}
		}
		if free {
			freeIP = append(freeIP, ip)
		}
	}

	if len(freeIP) == 0 {
		return "", fmt.Errorf("No IP left")
	}

	ipToBook := freeIP[rand.Intn(len(freeIP))]
	infos := ipInfos{
		workerName:     workerName,
		dateLastBooked: time.Now(),
	}
	ipsInfos.ips[ipToBook] = infos

	return ipToBook, nil
}

// IPinRanges returns a slice of all IP in all given IP ranges
// i.e 72.44.1.240/28,72.42.1.23/27
func IPinRanges(IPranges string) ([]string, error) {
	var ips []string

	ranges := strings.Split(IPranges, ",")
	for _, r := range ranges {
		i, err := IPinRange(r)
		if err != nil {
			return nil, err
		}
		ips = append(ips, i...)
	}
	return ips, nil
}

// IPinRange returns a slice of all IP in given IP range
// i.e 10.35.11.240/28
func IPinRange(IPrange string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(IPrange)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip2 := ip.Mask(ipnet.Mask); ipnet.Contains(ip2); inc(ip2) {
		log.Info("Adding %s to IP pool", ip2)
		ips = append(ips, ip2.String())
	}

	return ips, nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
