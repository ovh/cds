package vsphere

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/ovh/cds/sdk"
)

// InitHatchery create new client for vsphere
func (h *HatcheryVSphere) InitHatchery(ctx context.Context) error {
	if err := h.Common.Init(ctx, h); err != nil {
		return err
	}

	// Connect and login to ESX or vCenter
	c, err := h.newGovmomiClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to vsphere.newClient: %v", err)
	}

	log.Info(ctx, "connecting datacenter %s...", h.Config.VSphereDatacenterString)
	h.vSphereClient = NewVSphereClient(c, h.Config.VSphereDatacenterString)

	if err := h.initVSphereMetrics(ctx); err != nil {
		return fmt.Errorf("unable to init vsphere metrics: %v", err)
	}

	if h.Config.IPRange != "" {
		h.availableIPAddresses, err = sdk.IPinRanges(ctx, h.Config.IPRange)
		if err != nil {
			return err
		}
	}

	killAwolServersTick := time.NewTicker(2 * time.Minute)
	killDisabledWorkersTick := time.NewTicker(2 * time.Minute)

	if len(h.Config.WorkerProvisioning) > 0 {
		log.Debug(ctx, "provisioning is enabled")

		provisioningInterval := 2 * time.Minute
		if h.Config.WorkerProvisioningInterval > 0 {
			provisioningInterval = time.Duration(h.Config.WorkerProvisioningInterval) * time.Second
		}

		provisioningTick := time.NewTicker(provisioningInterval)
		h.GoRoutines.Run(ctx, "hatchery-vsphere-provisioning",
			func(ctx context.Context) {
				defer provisioningTick.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-provisioningTick.C:
						h.provisioningV2(ctx)
						h.provisioningV1(ctx)
					}
				}
			},
		)
	}

	h.GoRoutines.Run(ctx, "hatchery-vsphere-kill-awol-servers",
		func(ctx context.Context) {
			defer killAwolServersTick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-killAwolServersTick.C:
					h.killAwolServers(ctx)
				}
			}
		},
	)

	h.GoRoutines.Run(ctx, "hatchery-vsphere-kill-disable-workers",
		func(ctx context.Context) {
			defer killDisabledWorkersTick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-killDisabledWorkersTick.C:
					h.killDisabledWorkers(ctx)
				}
			}
		},
	)

	h.GoRoutines.Run(ctx, "hatchery-vsphere-metrics",
		func(ctx context.Context) {
			h.startVSphereMetricsRoutine(ctx, 30)
		},
	)

	// Log flavor configuration
	if len(h.Config.Flavors) > 0 {
		log.Info(ctx, "VM flavors configured: %d flavor(s)", len(h.Config.Flavors))
		for _, flavor := range h.Config.Flavors {
			log.Info(ctx, "  - %s: %d vCPUs, %d MB RAM", flavor.Name, flavor.CPUs, flavor.MemoryMB)
		}
		if h.Config.DefaultFlavor != "" {
			log.Info(ctx, "Default flavor: %s", h.Config.DefaultFlavor)
		}
		if h.Config.CountSmallerFlavorToKeep > 0 {
			log.Info(ctx, "Starvation prevention: reserve capacity for %d smaller workers", h.Config.CountSmallerFlavorToKeep)
		}
	} else {
		log.Info(ctx, "No VM flavors configured (template resources will be used)")
	}

	log.Info(ctx, "vSphere hatchery initialized")

	return nil
}

// newClient creates a govmomi.Client for use in the examples
func (h *HatcheryVSphere) newGovmomiClient(ctx context.Context) (*govmomi.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Parse URL from string
	u, err := soap.ParseURL("https://" + h.Config.VSphereUser + ":" + h.Config.VSpherePassword + "@" + h.Config.VSphereEndpoint)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot parse url")
	}

	log.Info(ctx, "initializing connection to https://%v", h.Config.VSphereEndpoint)

	// Connect and log in to ESX or vCenter
	return govmomi.NewClient(ctx, u, false)
}
