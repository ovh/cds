package openstack

import (
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tenantnetworks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// Init fetch uri from nova
// then list available models
// then list available images
func (h *HatcheryCloud) Init() error {
	// Register without declaring model
	h.hatch = &sdk.Hatchery{
		Name: hatchery.GenerateName("openstack", viper.GetString("name")),
		UID:  viper.GetString("uk"),
	}

	workersAlive = map[string]int64{}

	authOpts := gophercloud.AuthOptions{
		Username:         h.user,
		Password:         h.password,
		AllowReauth:      true,
		IdentityEndpoint: h.address,
		TenantName:       h.tenant,
	}

	provider, errac := openstack.AuthenticatedClient(authOpts)
	if errac != nil {
		log.Error("Unable to openstack.AuthenticatedClient: %s", errac)
		os.Exit(11)
	}

	client, errn := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{Region: h.region})
	if errn != nil {
		log.Error("Unable to openstack.NewComputeV2: %s", errn)
		os.Exit(12)
	}
	h.client = client

	if err := h.initFlavors(); err != nil {
		log.Warning("Error getting flavors: %s", err)
	}

	if err := h.initNetworks(); err != nil {
		log.Warning("Error getting networks: %s", err)
	}

	if err := h.initIPStatus(); err != nil {
		log.Warning("Error on initIPStatus(): %s", err)
	}

	if errRegistrer := hatchery.Register(h.hatch, viper.GetString("token")); errRegistrer != nil {
		log.Warning("Cannot register hatchery: %s", errRegistrer)
	}

	go h.main()

	return nil
}

func (h *HatcheryCloud) initFlavors() error {
	all, err := flavors.ListDetail(h.client, nil).AllPages()
	if err != nil {
		return fmt.Errorf("initFlavors> error on flavors.ListDetail: %s", err)
	}
	lflavors, err := flavors.ExtractFlavors(all)
	if err != nil {
		return fmt.Errorf("initFlavors> error on flavors.ExtractFlavors: %s", err)
	}
	h.flavors = lflavors
	return nil
}

func (h *HatcheryCloud) initNetworks() error {
	all, err := tenantnetworks.List(h.client).AllPages()
	if err != nil {
		return fmt.Errorf("initNetworks> Unable to get Network: %s", err)
	}
	nets, err := tenantnetworks.ExtractNetworks(all)
	if err != nil {
		return fmt.Errorf("initNetworks> Unable to get Network: %s", err)
	}
	for _, n := range nets {
		if n.Name == h.networkString {
			h.networkID = n.ID
			break
		}
	}
	return nil
}

// initIPStatus initializes ipsInfos to
// add workername on ip belong to openstack-ip-range
// this func is called once, when hatchery is starting
func (h *HatcheryCloud) initIPStatus() error {
	srvs := h.getServers()
	log.Info("initIPStatus> %d srvs", len(srvs))
	for ip := range ipsInfos.ips {
		log.Info("initIPStatus> checking %s", ip)
		for _, s := range srvs {
			if len(s.Addresses) == 0 {
				log.Info("initIPStatus> server %s - 0 addr", s.Name)
				continue
			}
			for k, v := range s.Addresses {
				if k != h.networkString {
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
								log.Info("initIPStatus> worker %s - use IP: %s", s.Name, addr)
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
