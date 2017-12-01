package openstack

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/ovh/cds/sdk/log"
)

// for each ip in the range, look for the first free ones
func (h *HatcheryOpenstack) findAvailableIP(workerName string) (string, error) {
	srvs := h.getServers()

	ipsInfos.mu.Lock()
	defer ipsInfos.mu.Unlock()
	for ip, infos := range ipsInfos.ips {
		// ip used less than 10s
		// 15s max to display worker name on nova list after call create
		if time.Since(infos.dateLastBooked) < 15*time.Second {
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

			for k, v := range s.Addresses {
				if k != h.Config.NetworkString {
					continue
				}
				switch v.(type) {
				case []interface{}:
					for _, z := range v.([]interface{}) {
						var addr string
						for x, y := range z.(map[string]interface{}) {
							if x == "addr" {
								addr = y.(string)
							}
						}
						if addr == ip {
							free = false
						}
					}
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
