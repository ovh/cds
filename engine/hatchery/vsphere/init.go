package vsphere

import (
	"context"
	"fmt"
	"os"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// Init create new client for vsphere
func (h *HatcheryVSphere) Init(name, api, token string, requestSecondsTimeout int, insecureSkipVerifyTLS bool) error {
	h.hatch = &sdk.Hatchery{
		Name:    hatchery.GenerateName("vsphere", name),
		Version: sdk.VERSION,
	}

	h.client = cdsclient.NewHatchery(api, token, requestSecondsTimeout, insecureSkipVerifyTLS)
	if err := hatchery.Register(h); err != nil {
		return fmt.Errorf("Cannot register: %s", err)
	}
	ctx := context.Background()

	// Connect and login to ESX or vCenter
	c, errNc := h.newClient(ctx)
	if errNc != nil {
		log.Error("Unable to vsphere.newClient: %s", errNc)
		os.Exit(11)
	}
	h.vclient = c

	finder := find.NewFinder(h.vclient.Client, false)
	h.finder = finder

	var errDc error
	if h.datacenter, errDc = finder.DatacenterOrDefault(ctx, h.datacenterString); errDc != nil {
		log.Error("Unable to find datacenter %s : %s", h.datacenterString, errDc)
		os.Exit(12)
	}
	finder.SetDatacenter(h.datacenter)

	var errN error
	if h.network, errN = finder.NetworkOrDefault(ctx, h.networkString); errN != nil {
		log.Error("Unable to find network %s : %s", h.networkString, errN)
		os.Exit(13)
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
