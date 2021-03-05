package vsphere

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestHatcheryVSphere_getAllServers(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: false,
					},
				},
			}, {
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: true,
					},
				},
			},
		}, nil
	}).AnyTimes()

	ctx := context.Background()
	vms := h.getRawVMs(ctx)
	require.Len(t, vms, 2)
}

func TestHatcheryVSphere_getVirtualMachines(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: false,
					},
				},
			}, {
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: true,
					},
				},
			},
		}, nil
	}).AnyTimes()

	ctx := context.Background()
	vms := h.getVirtualMachines(ctx)
	require.Len(t, vms, 1)
}

func TestHatcheryVSphere_getTemplates(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: false,
					},
				},
			}, {
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: true,
					},
				},
			},
		}, nil
	}).AnyTimes()

	ctx := context.Background()
	vms := h.getRawTemplates(ctx)
	require.Len(t, vms, 1)
}

func TestHatcheryVSphere_getVirtualMachineTemplates(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: false,
					},
				},
			}, {
				ManagedEntity: mo.ManagedEntity{
					Name: "foo",
				},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: true,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"model": true}`,
				},
			},
		}, nil
	}).AnyTimes()

	ctx := context.Background()
	tmpls := h.getVirtualMachineTemplates(ctx)
	require.Len(t, tmpls, 1)
}

func TestHatcheryVSphere_getVirtualMachineTemplateByName(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: false,
					},
				},
			}, {
				ManagedEntity: mo.ManagedEntity{
					Name: "foo",
				},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: true,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"model": true, "name": "foo"}`,
				},
			},
		}, nil
	}).AnyTimes()

	ctx := context.Background()
	tmpl, err := h.getVirtualMachineTemplateByName(ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, tmpl)
}

func TestHatcheryVSphere_deleteServer(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "register-foo").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &object.VirtualMachine{}, nil
		},
	)

	c.EXPECT().ShutdownVirtualMachine(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	c.EXPECT().DestroyVirtualMachine(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	ctx := context.Background()
	err := h.deleteServer(ctx, mo.VirtualMachine{
		ManagedEntity: mo.ManagedEntity{
			Name: "register-foo",
		},
		Summary: types.VirtualMachineSummary{
			Config: types.VirtualMachineConfigSummary{
				Template: true,
			},
		},
		Config: &types.VirtualMachineConfigInfo{
			Annotation: `{"model": true, "name": "foo"}`,
		},
	})
	require.NoError(t, err)
}

func TestHatcheryVSphere_prepareCloneSpec(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}
	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.availableIPAddresses = []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"}
	h.reservedIPAddresses = []string{"192.168.0.1", "192.168.0.2"}
	h.Config.Gateway = "192.168.0.254"
	h.Config.DNS = "192.168.0.253"

	c.EXPECT().LoadVirtualMachineDevices(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (object.VirtualDeviceList, error) {
			card := types.VirtualEthernetCard{}
			return object.VirtualDeviceList{
				&card,
			}, nil
		},
	)

	c.EXPECT().LoadNetwork(gomock.Any(), "vbox-net").DoAndReturn(
		func(ctx context.Context, s string) (object.NetworkReference, error) {
			return &object.Network{}, nil
		},
	)

	c.EXPECT().SetupEthernetCard(gomock.Any(), gomock.Any(), "ethernet-card", gomock.Any()).DoAndReturn(
		func(ctx context.Context, card *types.VirtualEthernetCard, ethernetCardName string, network object.NetworkReference) error {
			return nil
		},
	)

	c.EXPECT().LoadResourcePool(gomock.Any()).DoAndReturn(
		func(ctx context.Context) (*object.ResourcePool, error) {
			return &object.ResourcePool{}, nil
		},
	)

	c.EXPECT().LoadDatastore(gomock.Any(), "datastore").DoAndReturn(
		func(ctx context.Context, name string) (*object.Datastore, error) {
			return &object.Datastore{}, nil
		},
	)

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{
				{
					Summary: types.VirtualMachineSummary{
						Config: types.VirtualMachineConfigSummary{
							Template: false,
						},
					},
					Guest: &types.GuestInfo{
						Net: []types.GuestNicInfo{
							{
								IpAddress: []string{"192.168.0.2"},
							},
						},
					},
				},
			}, nil
		},
	).AnyTimes()

	ctx := context.Background()
	cloneSpec, err := h.prepareCloneSpec(ctx, &object.VirtualMachine{}, annotation{}, "foo")
	require.NoError(t, err)
	require.NotNil(t, cloneSpec)

	// Assert the IP address
	assert.Equal(t, "192.168.0.3", (cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress))
	assert.EqualValues(t, []string{"192.168.0.1", "192.168.0.3"}, h.reservedIPAddresses) // "192.168.0.2" should be removed because it's returned by the ListVirtualMachine
	// Assert the Gateway
	assert.Equal(t, "192.168.0.254", cloneSpec.Customization.NicSettingMap[0].Adapter.Gateway[0])
	// Assert the DNS
	assert.Equal(t, "192.168.0.253", cloneSpec.Customization.GlobalIPSettings.DnsServerList[0])

}

func TestHatcheryVSphere_launchClientOp(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	var vm = object.VirtualMachine{
		Common: object.Common{
			InventoryPath: "inventory-path",
		},
	}

	var model = sdk.ModelVirtualMachine{
		User:     "user",
		Password: "password",
	}

	var procman = guest.ProcessManager{}

	c.EXPECT().ProcessManager(gomock.Any(), &vm).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (*guest.ProcessManager, error) {
			return &procman, nil
		},
	)

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (int64, error) {
			t.Logf("req: %+v", req.Spec.GetGuestProgramSpec())
			assert.Equal(t, "user", req.Auth.(*types.NamePasswordAuthentication).Username)
			assert.Equal(t, "password", req.Auth.(*types.NamePasswordAuthentication).Password)
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;this is a script", req.Spec.GetGuestProgramSpec().Arguments)
			assert.EqualValues(t, []string{"env=1"}, req.Spec.GetGuestProgramSpec().EnvVariables)
			return 1, nil
		},
	)

	ctx := context.Background()
	h.launchClientOp(ctx, &vm, model, "this is a script", []string{"env=1"})
}
