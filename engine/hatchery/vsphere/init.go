package vsphere

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/ovh/cds/sdk"
)

// InitHatchery create new client for vsphere
func (h *HatcheryVSphere) InitHatchery(ctx context.Context) error {
	h.user = h.Config.VSphereUser
	h.password = h.Config.VSpherePassword
	h.endpoint = h.Config.VSphereEndpoint

	// Connect and login to ESX or vCenter
	c, err := h.newClient(ctx)
	if err != nil {
		return fmt.Errorf("Unable to vsphere.newClient: %v", err)
	}
	h.vclient = c

	finder := find.NewFinder(h.vclient.Client, false)
	h.finder = finder

	if h.datacenter, err = finder.DatacenterOrDefault(ctx, h.datacenterString); err != nil {
		return fmt.Errorf("Unable to find datacenter %s: %v", h.datacenterString, err)
	}
	finder.SetDatacenter(h.datacenter)

	if h.network, err = finder.NetworkOrDefault(ctx, h.networkString); err != nil {
		return fmt.Errorf("Unable to find network %s: %v", h.networkString, err)
	}

	if err := h.RefreshServiceLogger(ctx); err != nil {
		return fmt.Errorf("hatchery> vsphere> Cannot get cdn configuration : %v", err)
	}
	h.GoRoutines.Run(context.Background(), "hatchery vsphere main", func(ctx context.Context) {
		h.main(ctx)
	})

	return nil
}

// newClient creates a govmomi.Client for use in the examples
func (h *HatcheryVSphere) newClient(ctx context.Context) (*govmomi.Client, error) {
	// Parse URL from string
	u, err := soap.ParseURL("https://" + h.user + ":" + h.password + "@" + h.endpoint)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot parse url")
	}

	// Connect and log in to ESX or vCenter
	return govmomi.NewClient(ctx, u, false)
}
