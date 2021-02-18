package vsphere

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/ovh/cds/sdk"
)

// InitHatchery create new client for vsphere
func (h *HatcheryVSphere) InitHatchery(ctx context.Context) error {
	// Connect and login to ESX or vCenter
	c, err := h.newClient(ctx)
	if err != nil {
		return fmt.Errorf("Unable to vsphere.newClient: %v", err)
	}
	h.vclient = c

	log.Info(ctx, "connecting datacenter %s...", h.Config.VSphereDatacenterString)
	finder := find.NewFinder(h.vclient.Client, false)
	h.finder = finder

	if h.datacenter, err = finder.DatacenterOrDefault(ctx, h.Config.VSphereDatacenterString); err != nil {
		return fmt.Errorf("unable to find datacenter %s: %v", h.Config.VSphereDatacenterString, err)
	}
	finder.SetDatacenter(h.datacenter)
	log.Info(ctx, "connected to datacenter %v", h.datacenter)

	if h.network, err = finder.NetworkOrDefault(ctx, h.Config.VSphereNetworkString); err != nil {
		return fmt.Errorf("unable to find network %s: %v", h.Config.VSphereNetworkString, err)
	}

	if err := h.initIPStatus(ctx); err != nil {
		log.Warn(ctx, "Error on initIPStatus(): %v", err)
	}

	if err := h.RefreshServiceLogger(ctx); err != nil {
		return fmt.Errorf("hatchery> vsphere> Cannot get cdn configuration : %v", err)
	}
	h.GoRoutines.Run(ctx, "hatchery vsphere main", func(ctx context.Context) {
		log.Debug(ctx, "entering main loop")
		h.main(ctx)
	})

	log.Info(ctx, "vSphere hatchery initialized")

	return nil
}

// newClient creates a govmomi.Client for use in the examples
func (h *HatcheryVSphere) newClient(ctx context.Context) (*govmomi.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Parse URL from string
	u, err := soap.ParseURL("https://" + h.Config.VSphereUser + ":" + h.Config.VSpherePassword + "@" + h.Config.VSphereEndpoint)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot parse url")
	}

	log.Info(ctx, "initializing connection to %v...", u)

	// Connect and log in to ESX or vCenter
	return govmomi.NewClient(ctx, u, false)
}

// initIPStatus initializes ipsInfos to
// add workername on ip belong to openstack-ip-range
// this func is called once, when hatchery is starting
func (h *HatcheryVSphere) initIPStatus(ctx context.Context) error {
	srvs := h.getVirtualMachines(ctx)
	log.Info(ctx, "initIPStatus> %d srvs", len(srvs))
ipLoop:
	for ip := range ipsInfos.ips {
		log.Info(ctx, "initIPStatus> checking %s", ip)
		for _, s := range srvs {
			if s.Guest == nil {
				log.Info(ctx, "initIPStatus> server %s - 0 addr", s.Name)
				continue
			}
			for _, n := range s.Guest.Net {
				for _, vmIP := range n.IpAddress {
					log.Debug(ctx, "initIPStatus> server %s - address %s (checking %s)", s.Name, vmIP, ip)
					if vmIP != "" && vmIP == ip {
						log.Info(ctx, "initIPStatus> worker %s - use IP: %s", s.Name, vmIP)
						ipsInfos.ips[ip] = ipInfos{workerName: s.Name}
						continue ipLoop
					}
				}

			}
			log.Info(ctx, "initIPStatus> server %s - 0 addr", s.Name)
			continue
		}
	}
	return nil
}
