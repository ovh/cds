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
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "unable get cdn configuration : %v", err)
	}

	cdnConfTick := time.NewTicker(60 * time.Second)
	killAwolServersTick := time.NewTicker(2 * time.Minute)
	killDisabledWorkersTick := time.NewTicker(2 * time.Minute)
	provisioningTick := time.NewTicker(2 * time.Minute)

	h.GoRoutines.Run(ctx, "hatchery-vsphere-provisioning",
		func(ctx context.Context) {
			defer provisioningTick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-provisioningTick.C:
					h.provisioning(ctx)
				}
			}
		},
	)

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

	h.GoRoutines.Run(ctx, "hatchery-vsphere-refresh-service-logger",
		func(ctx context.Context) {
			defer cdnConfTick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-cdnConfTick.C:
					if err := h.RefreshServiceLogger(ctx); err != nil {
						ctx = sdk.ContextWithStacktrace(ctx, err)
						log.Error(ctx, "unable to get cdn configuration : %v", err)
					}
				}
			}
		},
	)

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
