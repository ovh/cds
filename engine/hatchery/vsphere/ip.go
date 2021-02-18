package vsphere

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// for each ip in the range, look for the first free ones
func (h *HatcheryVSphere) findAvailableIP(ctx context.Context, workerName string) (string, error) {
	srvs := h.getVirtualMachines(ctx)

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
	serverLoop:
		for _, s := range srvs {
			if s.Guest == nil {
				continue
			}
			for _, n := range s.Guest.Net {
				for _, vmIP := range n.IpAddress {
					if vmIP == ip {
						free = false
						break serverLoop
					}
				}

			}
		}
		if free {
			freeIP = append(freeIP, ip)
		}
	}

	if len(freeIP) == 0 {
		return "", fmt.Errorf("no IP left")
	}

	ipToBook := freeIP[rand.Intn(len(freeIP))]
	infos := ipInfos{
		workerName:     workerName,
		dateLastBooked: time.Now(),
	}
	ipsInfos.ips[ipToBook] = infos

	return ipToBook, nil
}
