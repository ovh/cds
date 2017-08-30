package vsphere

import (
	"context"
	"os"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
	"github.com/spf13/viper"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
)

// Init create newt client for vsphere
func (h *HatcheryVSphere) Init() error {
	// Register without declaring model
	h.hatch = &sdk.Hatchery{
		Name: hatchery.GenerateName("vsphere", viper.GetString("name")),
		UID:  viper.GetString("uk"),
	}

	workersAlive = map[string]int64{}
	ctx := context.TODO()

	// Connect and login to ESX or vCenter
	c, errNc := h.newClient(ctx)
	if errNc != nil {
		log.Error("Unable to vsphere.newClient: %s", errNc)
		os.Exit(11)
	}
	h.client = c

	finder := find.NewFinder(h.client.Client, false)
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

	if err := h.initImages(ctx); err != nil {
		log.Error("Unable to vsphere.initImages: %s", errNc)
		os.Exit(14)
	}

	if errRegistrer := hatchery.Register(h.hatch, viper.GetString("token")); errRegistrer != nil {
		log.Warning("Cannot register hatchery: %s", errRegistrer)
	}

	// go h.main()

	return nil
}

func (h *HatcheryVSphere) initImages(ctx context.Context) error {
	var vms []mo.VirtualMachine
	m := view.NewManager(h.client.Client)

	v, err := m.CreateContainerView(ctx, h.client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return err
	}
	defer v.Destroy(ctx)

	// Retrieve summary property for all machines
	// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.VirtualMachine.html
	if err := v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary", "config"}, &vms); err != nil {
		return err
	}

	for _, vm := range vms {
		h.images = append(h.images, vm.Summary.Config.Name)
	}

	return nil
}

// newClient creates a govmomi.Client for use in the examples
func (h *HatcheryVSphere) newClient(ctx context.Context) (*govmomi.Client, error) {
	// Parse URL from string
	u, err := soap.ParseURL("https://" + h.user + ":" + h.password + "@" + h.endpoint)
	if err != nil {
		return nil, err
	}

	// Connect and log in to ESX or vCenter
	return govmomi.NewClient(ctx, u, false)
}
