package vsphere

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"
)

func TestHatcheryVSphere_launchScriptWorkerv2(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Config: HatcheryConfiguration{
			GuestCredentials: []GuestCredential{
				{
					ModelVMWare: "the-model",
					Username:    "user",
					Password:    "password",
				},
			},
		},
	}
	var ctx = context.Background()
	var vm = object.VirtualMachine{
		Common: object.Common{},
	}

	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), &vm, gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, _ *string, _ string) error {
			return nil
		},
	)

	var procman = guest.ProcessManager{}

	c.EXPECT().ProcessManager(gomock.Any(), &vm).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (*guest.ProcessManager, error) {
			return &procman, nil
		},
	).AnyTimes()

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (*types.StartProgramInGuestResponse, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "-n ;env")
			return &types.StartProgramInGuestResponse{}, nil
		},
	)

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (*types.StartProgramInGuestResponse, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "-n ;\n")
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "./worker")
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "shutdown -h now")
			var foundConfig bool
			for _, env := range req.Spec.GetGuestProgramSpec().EnvVariables {
				if strings.HasPrefix(env, "CDS_CONFIG=") {
					foundConfig = true
				}
			}
			assert.True(t, foundConfig, "CDS_CONFIG env variable should be set")
			return &types.StartProgramInGuestResponse{}, nil
		},
	)

	spawnArgs := hatchery.SpawnArguments{
		WorkerName:  "worker1",
		WorkerToken: "xxxxxxxx",
		Model: sdk.WorkerStarterWorkerModel{
			ModelV2: &sdk.V2WorkerModel{},
			VSphereSpec: sdk.V2WorkerModelVSphereSpec{
				Image: "the-model",
			},
			Cmd:     "./worker",
			PostCmd: "shutdown -h now",
		},
		RegisterOnly: true,
	}

	err := h.launchScriptWorker(ctx, spawnArgs, &vm, "worker1")
	require.NoError(t, err)
}

func TestHatcheryVSphere_SpawnWorkerv2(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Config: HatcheryConfiguration{
			GuestCredentials: []GuestCredential{
				{
					ModelVMWare: "the-model",
					Username:    "user",
					Password:    "password",
				},
			},
		},
	}

	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.availableIPAddresses = []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"}
	h.availableNetworks = []availableNetwork{{
		config:      NetworkConfig{IPRange: "192.168.0.0/24", Gateway: "192.168.0.254", SubnetMask: "255.255.255.0"},
		ipAddresses: []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"},
	}}
	h.Config.Gateway = "192.168.0.254"
	h.Config.DNS = "192.168.0.253"

	var ctx = context.Background()

	var validModel = sdk.Model{
		Name: "cds-model-name",
		Type: sdk.VSphere,
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Cmd:     "./worker",
			Image:   "the-model",
			PostCmd: "shutdown -h now",
		},
		Group: &sdk.Group{
			Name: sdk.SharedInfraGroupName,
		},
	}

	var vmTemplate = object.VirtualMachine{
		Common: object.Common{
			InventoryPath: "the-model",
		},
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{
					Name: validModel.Name,
				},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: true,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"vmware_model_path": "the-model", "model": true}`,
				},
			},
		}, nil
	}).AnyTimes()

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "the-model").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vmTemplate, nil
		},
	)

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

	var folder object.Folder

	c.EXPECT().LoadFolder(gomock.Any()).DoAndReturn(
		func(ctx context.Context) (*object.Folder, error) {
			return &folder, nil
		},
	)

	var workerRef types.ManagedObjectReference

	c.EXPECT().CloneVirtualMachine(gomock.Any(), &vmTemplate, &folder, "worker-name", gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, folder *object.Folder, name string, config *types.VirtualMachineCloneSpec) (*types.ManagedObjectReference, error) {
			return &workerRef, nil
		},
	)

	var workerVM object.VirtualMachine

	c.EXPECT().NewVirtualMachine(gomock.Any(), gomock.Any(), &workerRef, gomock.Any()).DoAndReturn(
		func(ctx context.Context, cloneSpec *types.VirtualMachineCloneSpec, ref *types.ManagedObjectReference, vmname string) (*object.VirtualMachine, error) {
			assert.False(t, cloneSpec.Template)
			assert.True(t, cloneSpec.PowerOn)
			var givenAnnotation annotation
			json.Unmarshal([]byte(cloneSpec.Config.Annotation), &givenAnnotation)
			assert.Equal(t, "the-model", givenAnnotation.VMwareModelPath)
			assert.False(t, givenAnnotation.Model)
			assert.Equal(t, "192.168.0.1", (cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress))
			return &workerVM, nil
		},
	)

	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), &workerVM, gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, _ *string, _ string) error {
			return nil
		},
	)

	var procman = guest.ProcessManager{}

	c.EXPECT().ProcessManager(gomock.Any(), &workerVM).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (*guest.ProcessManager, error) {
			return &procman, nil
		},
	).AnyTimes()

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (*types.StartProgramInGuestResponse, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "-n ;env")
			return &types.StartProgramInGuestResponse{}, nil
		},
	)

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (*types.StartProgramInGuestResponse, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "-n ;\n")
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "shutdown -h now")
			var foundConfig bool
			for _, env := range req.Spec.GetGuestProgramSpec().EnvVariables {
				if strings.HasPrefix(env, "CDS_CONFIG=") {
					foundConfig = true
				}
			}
			assert.True(t, foundConfig, "CDS_CONFIG env variable should be set")
			return &types.StartProgramInGuestResponse{}, nil
		},
	)

	err := h.SpawnWorker(ctx, hatchery.SpawnArguments{
		WorkerName:  "worker-name",
		WorkerToken: "worker.token.xxx",
		Model: sdk.WorkerStarterWorkerModel{
			ModelV2: &sdk.V2WorkerModel{
				Name: "cds-model-name",
				Spec: json.RawMessage(`{"image":"the-model"}`),
			},
			VSphereSpec: sdk.V2WorkerModelVSphereSpec{
				Image:    "the-model",
				Username: "user",
				Password: "password",
			},
			PostCmd: "sudo shutdown -h now",
		},
		JobName: "job_name",
		JobID:   "666",
		Requirements: []sdk.Requirement{
			{
				Type:  sdk.ModelRequirement,
				Value: validModel.Name,
			},
		},
		RegisterOnly: false,
		HatcheryName: "hatchery_name",
	})
	require.NoError(t, err)
}

func TestHatcheryVSphere_SpawnWorkerFromProvisioning(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Config: HatcheryConfiguration{
			GuestCredentials: []GuestCredential{
				{
					ModelVMWare: "the-model",
					Username:    "user",
					Password:    "password",
				},
			},
		},
	}
	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.availableIPAddresses = []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"}
	h.availableNetworks = []availableNetwork{{
		config:      NetworkConfig{IPRange: "192.168.0.0/24", Gateway: "192.168.0.254", SubnetMask: "255.255.255.0"},
		ipAddresses: []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"},
	}}
	h.Config.Gateway = "192.168.0.254"
	h.Config.DNS = "192.168.0.253"

	var ctx = context.Background()

	var vmTemplate = object.VirtualMachine{
		Common: object.Common{
			InventoryPath: "the-model",
		},
	}

	var vmProvisionned = object.VirtualMachine{
		Common: object.Common{
			InventoryPath: "worker-name",
		},
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{
					Name: "the-model",
				},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: true,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"vmware_model_path": "the-model", "model": true}`,
				},
			}, {
				ManagedEntity: mo.ManagedEntity{
					Name: "provision-v2-worker",
				},
				Runtime: types.VirtualMachineRuntimeInfo{
					PowerState: types.VirtualMachinePowerStatePoweredOff,
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"provisioning": true, "vmware_model_path": "the-model"}`,
				},
			},
		}, nil
	}).Times(1)

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "the-model").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vmTemplate, nil
		},
	)

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "provision-v2-worker").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vmProvisionned, nil
		},
	)

	c.EXPECT().LoadVirtualMachineEvents(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, eventTypes ...string) ([]types.BaseEvent, error) {
			t.Logf("LoadVirtualMachineEvents for %s", vm.Name())
			return []types.BaseEvent{
				&types.VmPoweredOffEvent{
					VmEvent: types.VmEvent{
						Event: types.Event{
							CreatedTime: time.Now().Add(-10 * time.Minute),
						},
					},
				},
			}, nil
		},
	).Times(1)

	c.EXPECT().RenameVirtualMachine(gomock.Any(), &vmProvisionned, "worker-name").DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, s string) error {
			return nil
		},
	)

	c.EXPECT().StartVirtualMachine(gomock.Any(), &vmProvisionned).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{
					Name: "the-model",
				},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: true,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"vmware_model_path": "the-model", "model": true}`,
				},
			}, {
				ManagedEntity: mo.ManagedEntity{
					Name: "worker-name",
				},
				Runtime: types.VirtualMachineRuntimeInfo{
					PowerState: types.VirtualMachinePowerStatePoweredOn,
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"provisioning": true, "vmware_model_path": "the-model"}`,
				},
			},
		}, nil
	})

	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), &vmProvisionned, gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, _ *string, _ string) error {
			return nil
		},
	).Times(2)

	var procman = guest.ProcessManager{}

	c.EXPECT().ProcessManager(gomock.Any(), &vmProvisionned).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (*guest.ProcessManager, error) {
			return &procman, nil
		},
	).AnyTimes()

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (*types.StartProgramInGuestResponse, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;env", req.Spec.GetGuestProgramSpec().Arguments)
			return &types.StartProgramInGuestResponse{
				Returnval: 0,
			}, nil
		},
	)

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (*types.StartProgramInGuestResponse, error) {
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "-n ;\n")
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "./worker")
			assert.Contains(t, req.Spec.GetGuestProgramSpec().Arguments, "shutdown -h now")
			var foundConfig bool
			for _, env := range req.Spec.GetGuestProgramSpec().EnvVariables {
				if strings.HasPrefix(env, "CDS_CONFIG=") {
					foundConfig = true
				}
			}
			assert.True(t, foundConfig, "CDS_CONFIG env variable should be set")
			return &types.StartProgramInGuestResponse{}, nil
		},
	)

	err := h.SpawnWorker(ctx, hatchery.SpawnArguments{
		WorkerName:  "worker-name",
		WorkerToken: "worker.token.xxx",
		Model: sdk.WorkerStarterWorkerModel{
			ModelV2: &sdk.V2WorkerModel{
				Name: "cds-model-name",
				Spec: json.RawMessage(`{"image":"the-model"}`),
			},
			VSphereSpec: sdk.V2WorkerModelVSphereSpec{
				Image:    "the-model",
				Username: "user",
				Password: "password",
			},
			Cmd:     "./worker",
			PostCmd: "shutdown -h now",
		},
		JobName:     "job_name",
		JobID:       "666",
		NodeRunID:   999,
		NodeRunName: "nore_run_name",
		Requirements: []sdk.Requirement{
			{
				Type:  sdk.ModelRequirement,
				Value: "the-model",
			},
		},
		RegisterOnly: false,
		HatcheryName: "hatchery_name",
	})
	require.NoError(t, err)

}

func TestHatcheryVSphere_ProvisionWorker(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Config: HatcheryConfiguration{
			GuestCredentials: []GuestCredential{
				{
					ModelVMWare: "the-model",
					Username:    "user",
					Password:    "password",
				},
			},
		},
	}
	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.availableIPAddresses = []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"}
	h.availableNetworks = []availableNetwork{{
		config:      NetworkConfig{IPRange: "192.168.0.0/24", Gateway: "192.168.0.254", SubnetMask: "255.255.255.0"},
		ipAddresses: []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"},
	}}
	h.Config.Gateway = "192.168.0.254"
	h.Config.DNS = "192.168.0.253"

	var ctx = context.Background()

	var vmTemplate = object.VirtualMachine{
		Common: object.Common{
			InventoryPath: "the-model",
		},
	}

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "the-model").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vmTemplate, nil
		},
	)

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

	var folder object.Folder

	c.EXPECT().LoadFolder(gomock.Any()).DoAndReturn(
		func(ctx context.Context) (*object.Folder, error) {
			return &folder, nil
		},
	)

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{
					Name: "the-model",
				},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: true,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"vmware_model_path": "the-model", "model": true}`,
				},
			},
		}, nil
	})

	var workerRef types.ManagedObjectReference

	c.EXPECT().CloneVirtualMachine(gomock.Any(), &vmTemplate, &folder, "provisionned-worker", gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, folder *object.Folder, name string, config *types.VirtualMachineCloneSpec) (*types.ManagedObjectReference, error) {
			return &workerRef, nil
		},
	)

	var workerVM object.VirtualMachine

	c.EXPECT().NewVirtualMachine(gomock.Any(), gomock.Any(), &workerRef, gomock.Any()).DoAndReturn(
		func(ctx context.Context, cloneSpec *types.VirtualMachineCloneSpec, ref *types.ManagedObjectReference, vmname string) (*object.VirtualMachine, error) {
			assert.False(t, cloneSpec.Template)
			assert.True(t, cloneSpec.PowerOn)
			var givenAnnotation annotation
			json.Unmarshal([]byte(cloneSpec.Config.Annotation), &givenAnnotation)
			assert.Equal(t, "the-model", givenAnnotation.VMwareModelPath)
			assert.False(t, givenAnnotation.Model)
			assert.Equal(t, "192.168.0.1", (cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress))
			return &workerVM, nil
		},
	)

	err := h.ProvisionWorkerV2(ctx, "the-model", "provisionned-worker")
	require.NoError(t, err)
}
