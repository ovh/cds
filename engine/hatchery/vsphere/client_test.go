package vsphere

import (
	"context"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"
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

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "foo").DoAndReturn(
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
	h.Config.DNS = "192.168.0.253"

	c.EXPECT().LoadVirtualMachineDevices(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (object.VirtualDeviceList, error) {
			card := types.VirtualEthernetCard{}
			return object.VirtualDeviceList{
				&card,
			}, nil
		},
	)
	c.EXPECT().LoadNetwork(gomock.Any(), "vbox-net").Return(&object.Network{}, nil)
	c.EXPECT().SetupEthernetCard(gomock.Any(), gomock.Any(), "ethernet-card", gomock.Any()).Return(nil)
	c.EXPECT().LoadResourcePool(gomock.Any()).Return(&object.ResourcePool{}, nil)
	c.EXPECT().LoadDatastore(gomock.Any(), "datastore").Return(&object.Datastore{}, nil)

	ctx := context.Background()
	annot := annotation{}
	ip := &ipResult{ip: "192.168.0.3", gateway: "192.168.0.254", subnetMask: "255.255.255.0"}
	cloneSpec, err := h.prepareCloneSpec(ctx, &object.VirtualMachine{}, &annot, ip)
	require.NoError(t, err)
	require.NotNil(t, cloneSpec)

	// The clone spec uses the caller-chosen IP, and the IP is recorded in the annotation.
	assert.Equal(t, "192.168.0.3", (cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress))
	assert.Equal(t, "192.168.0.254", cloneSpec.Customization.NicSettingMap[0].Adapter.Gateway[0])
	assert.Equal(t, "192.168.0.253", cloneSpec.Customization.GlobalIPSettings.DnsServerList[0])
	assert.Equal(t, "192.168.0.3", annot.IPAddress)
}

func TestHatcheryVSphere_launchClientOpV2(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Config: HatcheryConfiguration{
			GuestCredentials: []GuestCredential{
				{
					ModelVMWare: "vmware-model",
					Username:    "user",
					Password:    "password",
				},
			},
		},
	}

	var vm = object.VirtualMachine{
		Common: object.Common{
			InventoryPath: "inventory-path",
		},
	}

	var procman = guest.ProcessManager{}

	c.EXPECT().ProcessManager(gomock.Any(), &vm).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (*guest.ProcessManager, error) {
			return &procman, nil
		},
	)

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (*types.StartProgramInGuestResponse, error) {
			t.Logf("req: %+v", req.Spec.GetGuestProgramSpec())
			assert.Equal(t, "user", req.Auth.(*types.NamePasswordAuthentication).Username)
			assert.Equal(t, "password", req.Auth.(*types.NamePasswordAuthentication).Password)
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "-n ;this is a script")
			assert.EqualValues(t, []string{"env=1"}, req.Spec.GetGuestProgramSpec().EnvVariables)
			return &types.StartProgramInGuestResponse{}, nil
		},
	)

	ctx := context.Background()

	h.launchClientOp(ctx, &vm,
		sdk.WorkerStarterWorkerModel{ModelV2: &sdk.V2WorkerModel{},
			VSphereSpec: sdk.V2WorkerModelVSphereSpec{
				Image:    "vmware-model",
				Username: "user",
				Password: "password",
			},
		}, "this is a script", []string{"env=1"})
}
