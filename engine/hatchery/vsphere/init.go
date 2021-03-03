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
	// Connect and login to ESX or vCenter
	c, err := h.newGovmomiClient(ctx)
	if err != nil {
		return fmt.Errorf("Unable to vsphere.newClient: %v", err)
	}

	log.Info(ctx, "connecting datacenter %s...", h.Config.VSphereDatacenterString)
	h.vSphereClient = NewVSphereClient(c, h.Config.VSphereDatacenterString)

	if h.Config.IPRange != "" {
		h.availableIPAddresses, err = sdk.IPinRanges(ctx, h.Config.IPRange)
		if err != nil {
			return err
		}
	}

	if err := h.RefreshServiceLogger(ctx); err != nil {
		return fmt.Errorf("hatchery> vsphere> Cannot get cdn configuration : %v", err)
	}

	h.GoRoutines.Run(ctx, "hatchery-vsphere-main", func(ctx context.Context) {
		if err := h.RefreshServiceLogger(ctx); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable get cdn configuration : %v", err)
		}

		cdnConfTick := time.NewTicker(60 * time.Second).C
		killAwolServersTick := time.NewTicker(20 * time.Second).C
		killDisabledWorkersTick := time.NewTicker(60 * time.Second).C

		for {
			select {
			case <-killAwolServersTick:
				h.killAwolServers(ctx)
			case <-killDisabledWorkersTick:
				h.killDisabledWorkers(ctx)
			case <-cdnConfTick:
				if err := h.RefreshServiceLogger(ctx); err != nil {
					ctx = sdk.ContextWithStacktrace(ctx, err)
					log.Error(ctx, "unable to get cdn configuration : %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	})

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

	log.Info(ctx, "initializing connection to %v...", u)

	// Connect and log in to ESX or vCenter
	return govmomi.NewClient(ctx, u, false)
}
