package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestHatcheryVSphere_createVirtualMachineTemplate(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.availableIPAddresses = []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"}
	h.Config.Gateway = "192.168.0.254"
	h.Config.DNS = "192.168.0.253"

	var ctx = context.Background()
	var validModel = sdk.Model{
		Name: "model",
		Type: sdk.VSphere,
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Cmd:     "cmd_register_this_model",
			Image:   "model",
			PostCmd: "shutdown -h now",
		},
		Group: &sdk.Group{
			Name: sdk.SharedInfraGroupName,
		},
	}

	var vm = object.VirtualMachine{
		Common: object.Common{},
	}

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "model").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vm, nil
		},
	)

	var ethernetCard types.VirtualEthernetCard

	c.EXPECT().LoadVirtualMachineDevices(gomock.Any(), &vm).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (object.VirtualDeviceList, error) {
			return object.VirtualDeviceList{
				&ethernetCard,
			}, nil
		},
	)

	var network object.Network

	c.EXPECT().LoadNetwork(gomock.Any(), "vbox-net").DoAndReturn(
		func(ctx context.Context, s string) (object.NetworkReference, error) {
			return &network, nil
		},
	)

	c.EXPECT().SetupEthernetCard(gomock.Any(), &ethernetCard, "ethernet-card", &network).DoAndReturn(
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
								IpAddress: []string{"192.168.0.1"},
							},
						},
					},
				},
			}, nil
		},
	)

	var folder object.Folder

	c.EXPECT().LoadFolder(gomock.Any()).DoAndReturn(
		func(ctx context.Context) (*object.Folder, error) {
			return &folder, nil
		},
	)

	var clonedRef types.ManagedObjectReference

	c.EXPECT().CloneVirtualMachine(gomock.Any(), &vm, &folder, "model-tmp", gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, folder *object.Folder, name string, config *types.VirtualMachineCloneSpec) (*types.ManagedObjectReference, error) {
			return &clonedRef, nil
		},
	)

	var clonedVM object.VirtualMachine

	c.EXPECT().NewVirtualMachine(gomock.Any(), gomock.Any(), &clonedRef).DoAndReturn(
		func(ctx context.Context, cloneSpec *types.VirtualMachineCloneSpec, ref *types.ManagedObjectReference) (*object.VirtualMachine, error) {
			assert.False(t, cloneSpec.Template)
			assert.True(t, cloneSpec.PowerOn)
			var givenAnnotation annotation
			json.Unmarshal([]byte(cloneSpec.Config.Annotation), &givenAnnotation)
			assert.Equal(t, "model", givenAnnotation.WorkerModelPath)
			assert.True(t, givenAnnotation.Model)
			assert.Equal(t, "192.168.0.2", (cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress))
			return &clonedVM, nil
		},
	)

	var procman = guest.ProcessManager{}

	c.EXPECT().ProcessManager(gomock.Any(), &clonedVM).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (*guest.ProcessManager, error) {
			return &procman, nil
		},
	).AnyTimes()

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (int64, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;env", req.Spec.GetGuestProgramSpec().Arguments)
			return 1, nil
		},
	)

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (int64, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;shutdown -h now", req.Spec.GetGuestProgramSpec().Arguments)
			return 1, nil
		},
	)

	c.EXPECT().WaitForVirtualMachineShutdown(gomock.Any(), &clonedVM).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	c.EXPECT().RenameVirtualMachine(gomock.Any(), &clonedVM, "model").DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, s string) error {
			return nil
		},
	)

	c.EXPECT().MarkVirtualMachineAsTemplate(gomock.Any(), &clonedVM).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	vmTemplate, err := h.createVirtualMachineTemplate(ctx, validModel, "worker1")
	require.NoError(t, err)
	require.NotNil(t, vmTemplate)
}

func TestHatcheryVSphere_launchScriptWorker(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}
	var ctx = context.Background()
	var vm = object.VirtualMachine{
		Common: object.Common{},
	}
	var validModel = sdk.Model{
		Name: "model",
		Type: sdk.VSphere,
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Cmd:     "./worker",
			Image:   "model",
			PostCmd: "shutdown -h now",
		},
		Group: &sdk.Group{
			Name: sdk.SharedInfraGroupName,
		},
	}

	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), &vm).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
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
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (int64, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;env", req.Spec.GetGuestProgramSpec().Arguments)
			return 1, nil
		},
	)

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (int64, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;\n./worker register\nshutdown -h now", req.Spec.GetGuestProgramSpec().Arguments)
			return 1, nil
		},
	)

	err := h.launchScriptWorker(ctx, "worker1", -1, "xxxxxxxx", validModel, true, &vm)
	require.NoError(t, err)

}

func TestHatcheryVSphere_SpawnWorker(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}
	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.availableIPAddresses = []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"}
	h.Config.Gateway = "192.168.0.254"
	h.Config.DNS = "192.168.0.253"

	var ctx = context.Background()

	var validModel = sdk.Model{
		Name: "model",
		Type: sdk.VSphere,
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Cmd:     "./worker",
			Image:   "model",
			PostCmd: "shutdown -h now",
		},
		Group: &sdk.Group{
			Name: sdk.SharedInfraGroupName,
		},
	}

	var now = time.Now()

	var vmTemplate = object.VirtualMachine{
		Common: object.Common{},
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
					Annotation: fmt.Sprintf(`{"worker_model_last_modified": "%d", "model": true}`, now.Unix()),
				},
			},
		}, nil
	})

	c.EXPECT().LoadVirtualMachine(gomock.Any(), validModel.Name).DoAndReturn(
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

	c.EXPECT().NewVirtualMachine(gomock.Any(), gomock.Any(), &workerRef).DoAndReturn(
		func(ctx context.Context, cloneSpec *types.VirtualMachineCloneSpec, ref *types.ManagedObjectReference) (*object.VirtualMachine, error) {
			assert.False(t, cloneSpec.Template)
			assert.True(t, cloneSpec.PowerOn)
			var givenAnnotation annotation
			json.Unmarshal([]byte(cloneSpec.Config.Annotation), &givenAnnotation)
			assert.Equal(t, "shared.infra/model", givenAnnotation.WorkerModelPath)
			assert.False(t, givenAnnotation.Model)
			assert.Equal(t, "192.168.0.1", (cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress))
			return &workerVM, nil
		},
	)

	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), &workerVM).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
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
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (int64, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;env", req.Spec.GetGuestProgramSpec().Arguments)
			return 1, nil
		},
	)

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (int64, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;\n./worker\nshutdown -h now", req.Spec.GetGuestProgramSpec().Arguments)
			return 1, nil
		},
	)

	err := h.SpawnWorker(ctx, hatchery.SpawnArguments{
		WorkerName:  "worker-name",
		WorkerToken: "worker.token.xxx",
		Model:       &validModel,
		JobName:     "job_name",
		JobID:       666,
		NodeRunID:   999,
		NodeRunName: "nore_run_name",
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
	}
	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.availableIPAddresses = []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"}
	h.Config.Gateway = "192.168.0.254"
	h.Config.DNS = "192.168.0.253"

	var ctx = context.Background()

	var now = time.Now()

	var vmTemplate = object.VirtualMachine{
		Common: object.Common{},
	}

	var vmProvisionned = object.VirtualMachine{
		Common: object.Common{},
	}

	var validModel = sdk.Model{
		Name: "model",
		Type: sdk.VSphere,
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Cmd:     "./worker",
			Image:   "model",
			PostCmd: "shutdown -h now",
		},
		Group: &sdk.Group{
			Name: sdk.SharedInfraGroupName,
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
					Annotation: fmt.Sprintf(`{"worker_model_last_modified": "%d", "model": true}`, now.Unix()),
				},
			}, {
				ManagedEntity: mo.ManagedEntity{
					Name: "provisionned-worker",
				},
				Runtime: types.VirtualMachineRuntimeInfo{
					PowerState: types.VirtualMachinePowerStatePoweredOff,
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: fmt.Sprintf(`{"worker_model_last_modified": "%d", "provisioning": true, "worker_model_path": "%s"}`, now.Unix(), validModel.Path()),
				},
			},
		}, nil
	})

	c.EXPECT().LoadVirtualMachine(gomock.Any(), validModel.Name).DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vmTemplate, nil
		},
	)

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "provisionned-worker").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vmProvisionned, nil
		},
	)

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

	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), &vmProvisionned).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	var procman = guest.ProcessManager{}

	c.EXPECT().ProcessManager(gomock.Any(), &vmProvisionned).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (*guest.ProcessManager, error) {
			return &procman, nil
		},
	).AnyTimes()

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (int64, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;env", req.Spec.GetGuestProgramSpec().Arguments)
			return 1, nil
		},
	)

	c.EXPECT().StartProgramInGuest(gomock.Any(), &procman, gomock.Any()).DoAndReturn(
		func(ctx context.Context, procman *guest.ProcessManager, req *types.StartProgramInGuest) (int64, error) {
			assert.Equal(t, "/bin/echo", req.Spec.GetGuestProgramSpec().ProgramPath)
			assert.Equal(t, "-n ;\n./worker\nshutdown -h now", req.Spec.GetGuestProgramSpec().Arguments)
			return 1, nil
		},
	)

	err := h.SpawnWorker(ctx, hatchery.SpawnArguments{
		WorkerName:  "worker-name",
		WorkerToken: "worker.token.xxx",
		Model:       &validModel,
		JobName:     "job_name",
		JobID:       666,
		NodeRunID:   999,
		NodeRunName: "nore_run_name",
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

func TestHatcheryVSphere_ProvisionWorker(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}
	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.availableIPAddresses = []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"}
	h.Config.Gateway = "192.168.0.254"
	h.Config.DNS = "192.168.0.253"

	var ctx = context.Background()

	var validModel = sdk.Model{
		Name: "model",
		Type: sdk.VSphere,
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Cmd:     "./worker",
			Image:   "model",
			PostCmd: "shutdown -h now",
		},
		Group: &sdk.Group{
			Name: sdk.SharedInfraGroupName,
		},
	}

	var vmTemplate = object.VirtualMachine{
		Common: object.Common{},
	}

	c.EXPECT().LoadVirtualMachine(gomock.Any(), validModel.Name).DoAndReturn(
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

	var now = time.Now()

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
					Annotation: fmt.Sprintf(`{"worker_model_last_modified": "%d", "model": true}`, now.Unix()),
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

	c.EXPECT().NewVirtualMachine(gomock.Any(), gomock.Any(), &workerRef).DoAndReturn(
		func(ctx context.Context, cloneSpec *types.VirtualMachineCloneSpec, ref *types.ManagedObjectReference) (*object.VirtualMachine, error) {
			assert.False(t, cloneSpec.Template)
			assert.True(t, cloneSpec.PowerOn)
			var givenAnnotation annotation
			json.Unmarshal([]byte(cloneSpec.Config.Annotation), &givenAnnotation)
			assert.Equal(t, "shared.infra/model", givenAnnotation.WorkerModelPath)
			assert.False(t, givenAnnotation.Model)
			assert.Equal(t, "192.168.0.1", (cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress))
			return &workerVM, nil
		},
	)

	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), &workerVM).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	c.EXPECT().ShutdownVirtualMachine(gomock.Any(), &workerVM).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	err := h.ProvisionWorker(ctx, validModel, "provisionned-worker")
	require.NoError(t, err)
}
