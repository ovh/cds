package vsphere

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// Init create new client for vsphere
func (h *HatcheryVSphere) Init() error {
	h.hatch = &sdk.Hatchery{
		Name:    h.Configuration().Name,
		Version: sdk.VERSION,
	}

	h.client = cdsclient.NewHatchery(
		h.Configuration().API.HTTP.URL,
		h.Configuration().API.Token,
		h.Configuration().Provision.RegisterFrequency,
		h.Configuration().API.HTTP.Insecure,
		h.hatch.Name,
	)
	if err := hatchery.Register(h); err != nil {
		return fmt.Errorf("Cannot register: %s", err)
	}
	ctx := context.Background()

	// Connect and login to ESX or vCenter
	c, errNc := h.newClient(ctx)
	if errNc != nil {
		return fmt.Errorf("Unable to vsphere.newClient: %s", errNc)
	}
	h.vclient = c

	finder := find.NewFinder(h.vclient.Client, false)
	h.finder = finder

	var errDc error
	if h.datacenter, errDc = finder.DatacenterOrDefault(ctx, h.datacenterString); errDc != nil {
		return fmt.Errorf("Unable to find datacenter %s : %s", h.datacenterString, errDc)
	}
	finder.SetDatacenter(h.datacenter)

	var errN error
	if h.network, errN = finder.NetworkOrDefault(ctx, h.networkString); errN != nil {
		return fmt.Errorf("Unable to find network %s : %s", h.networkString, errN)
	}

	go h.main()

	return nil
}

// newClient creates a govmomi.Client for use in the examples
func (h *HatcheryVSphere) newClient(ctx context.Context) (*govmomi.Client, error) {
	// Parse URL from string
	u, err := soap.ParseURL("https://" + h.user + ":" + h.password + "@" + h.endpoint)
	if err != nil {
		return nil, sdk.WrapError(err, "newClient> cannot parse url")
	}

	// Connect and log in to ESX or vCenter
	return govmomi.NewClient(ctx, u, false)
}
