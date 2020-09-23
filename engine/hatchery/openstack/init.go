package openstack

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tenantnetworks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InitHatchery fetch uri from nova
// then list available models
// then list available images
func (h *HatcheryOpenstack) InitHatchery(ctx context.Context) error {
	workersAlive = map[string]int64{}

	authOpts := gophercloud.AuthOptions{
		Username:         h.Config.User,
		Password:         h.Config.Password,
		AllowReauth:      true,
		IdentityEndpoint: h.Config.Address,
		TenantName:       h.Config.Tenant,
		DomainName:       h.Config.Domain,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to openstack.AuthenticatedClient: %v", err))
	}

	openstackClient, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{Region: h.Config.Region})
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to openstack.NewComputeV2: %v", err))
	}
	h.openstackClient = openstackClient

	if err := h.initFlavors(); err != nil {
		log.Warning(ctx, "Error getting flavors: %v", err)
	}

	if err := h.initNetworks(); err != nil {
		log.Warning(ctx, "Error getting networks: %v", err)
	}

	if err := h.initIPStatus(ctx); err != nil {
		log.Warning(ctx, "Error on initIPStatus(): %v", err)
	}

	if err := h.RefreshServiceLogger(ctx); err != nil {
		log.Error(ctx, "Hatchery> openstack> Cannot get cdn configuration : %v", err)
	}
	h.GoRoutines.Run(context.Background(), "hatchery openstack routines", func(ctx context.Context) {
		h.main(ctx)
	})

	return nil
}

func (h *HatcheryOpenstack) initFlavors() error {
	all, err := flavors.ListDetail(h.openstackClient, nil).AllPages()
	if err != nil {
		return sdk.WithStack(fmt.Errorf("initFlavors> error on flavors.ListDetail: %v", err))
	}

	lflavors, err := flavors.ExtractFlavors(all)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("initFlavors> error on flavors.ExtractFlavors: %v", err))
	}

	h.flavors = h.filterAllowedFlavors(lflavors)

	return nil
}

func (h HatcheryOpenstack) filterAllowedFlavors(allFlavors []flavors.Flavor) []flavors.Flavor {
	// If allowed flavors are given in configuration we should check that given flavor is part of the list.
	if len(h.Config.AllowedFlavors) == 0 {
		return allFlavors
	}

	filteredFlavors := make([]flavors.Flavor, 0, len(allFlavors))
	for i := range allFlavors {
		var allowed bool
		for j := range h.Config.AllowedFlavors {
			if h.Config.AllowedFlavors[j] == allFlavors[i].Name {
				allowed = true
				break
			}
		}
		if !allowed {
			log.Debug("initFlavors> flavor '%s' is not allowed", allFlavors[i].Name)
			continue
		}
		filteredFlavors = append(filteredFlavors, allFlavors[i])
	}
	return filteredFlavors
}

func (h *HatcheryOpenstack) initNetworks() error {
	all, err := tenantnetworks.List(h.openstackClient).AllPages()
	if err != nil {
		return sdk.WithStack(fmt.Errorf("initNetworks> Unable to get Network: %v", err))
	}
	nets, err := tenantnetworks.ExtractNetworks(all)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("initNetworks> Unable to get Network: %v", err))
	}
	for _, n := range nets {
		if n.Name == h.Config.NetworkString {
			h.networkID = n.ID
			break
		}
	}
	return nil
}

// initIPStatus initializes ipsInfos to
// add workername on ip belong to openstack-ip-range
// this func is called once, when hatchery is starting
func (h *HatcheryOpenstack) initIPStatus(ctx context.Context) error {
	srvs := h.getServers(ctx)
	log.Info(ctx, "initIPStatus> %d srvs", len(srvs))
	for ip := range ipsInfos.ips {
		log.Info(ctx, "initIPStatus> checking %s", ip)
		for _, s := range srvs {
			if len(s.Addresses) == 0 {
				log.Info(ctx, "initIPStatus> server %s - 0 addr", s.Name)
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
						var version int
						for x, y := range z.(map[string]interface{}) {
							if x == "addr" {
								addr = y.(string)
							}
							if x == "version" {
								version = int(y.(float64))
							}
						}
						//we only support IPV4
						if addr != "" && version == 4 {
							log.Debug("initIPStatus> server %s - address %s (checking %s)", s.Name, addr, ip)
							if addr != "" && addr == ip {
								log.Info(ctx, "initIPStatus> worker %s - use IP: %s", s.Name, addr)
								ipsInfos.ips[ip] = ipInfos{workerName: s.Name}
							}
						}
					}
				}
			}
		}
	}
	return nil
}
