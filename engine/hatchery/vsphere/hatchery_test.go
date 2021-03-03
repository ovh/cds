package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	sdkhatchery "github.com/ovh/cds/sdk/hatchery"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestHatcheryVSphere_CanSpawn(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	var ctx = context.Background()
	var invalidModel = sdk.Model{}
	var validModel = sdk.Model{
		Name: "model",
		Type: sdk.VSphere,
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Cmd: "cmd",
		}}

	assert.False(t, h.CanSpawn(ctx, &invalidModel, 1, []sdk.Requirement{{Type: sdk.ModelRequirement}}), "without a model VSphere, it should return False")
	assert.False(t, h.CanSpawn(ctx, &validModel, 1, []sdk.Requirement{{Type: sdk.ServiceRequirement}}), "without a service requirement, it should return False")
	assert.False(t, h.CanSpawn(ctx, &validModel, 1, []sdk.Requirement{{Type: sdk.MemoryRequirement}}), "without a memory requirement, it should return False")
	assert.False(t, h.CanSpawn(ctx, &validModel, 1, []sdk.Requirement{{Type: sdk.HostnameRequirement}}), "without a hostname requirement, it should return False")

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{
					Name: "worker1",
				},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: false,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"job_id": 1}`,
				},
			},
		}, nil
	})
	assert.False(t, h.CanSpawn(ctx, &validModel, 1, []sdk.Requirement{}), "it should return False, because there is a worker for the same job")

	h.cacheVirtualMachines.list = []mo.VirtualMachine{} // flush the cache for the next call
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{
					Name: "worker1",
				},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template: false,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"job_id": 2}`,
				},
			},
		}, nil
	})
	assert.True(t, h.CanSpawn(ctx, &validModel, 1, []sdk.Requirement{}), "it should return True")

	h.cacheVirtualMachines.list = []mo.VirtualMachine{} // flush the cache for the next call
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{}, nil
	})
	assert.True(t, h.CanSpawn(ctx, &validModel, 0, []sdk.Requirement{}), "it should return True")

	h.cacheVirtualMachines.list = []mo.VirtualMachine{} // flush the cache for the next call
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{
					Name: validModel.Name + "-tmp",
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: fmt.Sprintf(`{"worker_model_path": "%s"}`, validModel.Name),
				},
			},
		}, nil
	})
	assert.False(t, h.CanSpawn(ctx, &validModel, 0, []sdk.Requirement{}), "with a 'tmp' vm, it should return False")

	h.cacheVirtualMachines.list = []mo.VirtualMachine{} // flush the cache for the next call
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{
					Name: "register-" + validModel.Name + "-blabla",
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: fmt.Sprintf(`{"worker_model_path": "%s"}`, validModel.Name),
				},
			},
		}, nil
	})
	assert.False(t, h.CanSpawn(ctx, &validModel, 0, []sdk.Requirement{}), "with a 'register' vm, it should return False")

	h.cacheVirtualMachines.list = []mo.VirtualMachine{} // flush the cache for the next call
	h.cachePendingJobID.list = append(h.cachePendingJobID.list, 666)
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{}, nil
	})
	assert.False(t, h.CanSpawn(ctx, &validModel, 666, []sdk.Requirement{}), "it should return False because the jobID is still in the local cache")

}

func TestHatcheryVSphere_NeedRegistration(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	var ctx = context.Background()
	var now = time.Now()
	var validModel = sdk.Model{
		Name: "model",
		Type: sdk.VSphere,
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Cmd: "cmd",
		},
		UserLastModified: now,
	}

	// Without any VM returned by vSphere, it should return True
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{}, nil
	})
	assert.True(t, h.NeedRegistration(ctx, &validModel), "without any VM returned by vSphere, it should return True")

	// vSphere returns a VM Template maching to te model, it should return False
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{
					Name: "model",
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
	assert.False(t, h.NeedRegistration(ctx, &validModel), "vSphere returns a VM Template maching to te model, it should return False")

}

func TestHatcheryVSphere_killDisabledWorkers(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	sdkhatchery.InitMetrics(context.Background())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cdsclient := mock_cdsclient.NewMockInterface(ctrl)

	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{
				Client: cdsclient,
			},
		},
	}

	cdsclient.EXPECT().WorkerList(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]sdk.Worker, error) {
			var un int64 = 1
			return []sdk.Worker{
				{
					Name:    "worker1",
					ModelID: &un,
					Status:  sdk.StatusDisabled,
				},
			}, nil
		},
	)

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{
				{
					ManagedEntity: mo.ManagedEntity{
						Name: "worker1",
					},
					Summary: types.VirtualMachineSummary{
						Config: types.VirtualMachineConfigSummary{
							Template: false,
						},
					},
					Config: &types.VirtualMachineConfigInfo{
						Annotation: `{"model": false}`,
					},
				},
			}, nil
		},
	)

	var vm = object.VirtualMachine{
		Common: object.Common{},
	}

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "worker1").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vm, nil
		},
	)

	c.EXPECT().ShutdownVirtualMachine(gomock.Any(), &vm).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	c.EXPECT().DestroyVirtualMachine(gomock.Any(), &vm).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	h.killDisabledWorkers(context.Background())
}

func TestHatcheryVSphere_killAwolServers(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	sdkhatchery.InitMetrics(context.Background())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cdsclient := mock_cdsclient.NewMockInterface(ctrl)

	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{
				Client: cdsclient,
			},
		},
	}

	cdsclient.EXPECT().WorkerList(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]sdk.Worker, error) {
			var un int64 = 1
			return []sdk.Worker{
				{
					Name:    "worker1",
					ModelID: &un,
					Status:  sdk.StatusDisabled,
				},
			}, nil
		},
	)

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{
				{
					ManagedEntity: mo.ManagedEntity{
						Name: "worker0",
					},
					Summary: types.VirtualMachineSummary{
						Config: types.VirtualMachineConfigSummary{
							Template: false,
						},
						Runtime: types.VirtualMachineRuntimeInfo{
							PowerState: types.VirtualMachinePowerStatePoweredOn,
						},
					},
					Config: &types.VirtualMachineConfigInfo{
						Annotation: `{"model": false, "to_delete": false, "worker_model_path": "someting"}`,
					},
				},
				{
					ManagedEntity: mo.ManagedEntity{
						Name: "worker1",
					},
					Summary: types.VirtualMachineSummary{
						Config: types.VirtualMachineConfigSummary{
							Template: false,
						},
					},
					Config: &types.VirtualMachineConfigInfo{
						Annotation: `{"model": false, "to_delete": true}`,
					},
				},
			}, nil
		},
	).Times(2)

	var vm0 = object.VirtualMachine{
		Common: object.Common{
			InventoryPath: "worker0",
		},
	}

	var vm1 = object.VirtualMachine{
		Common: object.Common{},
	}

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "worker0").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vm0, nil
		},
	)

	c.EXPECT().ReconfigureVirtualMachine(gomock.Any(), &vm0, gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, config types.VirtualMachineConfigSpec) error {
			t.Logf("config: %+v", config)
			assert.Equal(t, `{"worker_model_path":"someting","to_delete":true,"created":"0001-01-01T00:00:00Z"}`, config.Annotation)
			return nil
		},
	)

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "worker1").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			return &vm1, nil
		},
	)

	c.EXPECT().ShutdownVirtualMachine(gomock.Any(), &vm1).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	c.EXPECT().DestroyVirtualMachine(gomock.Any(), &vm1).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error {
			return nil
		},
	)

	h.killAwolServers(context.Background())
}

func TestHatcheryVSphere_Status(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{
				GoRoutines: sdk.NewGoRoutines(),
			},
		},
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{
				{
					ManagedEntity: mo.ManagedEntity{
						Name: "worker0",
					},
					Summary: types.VirtualMachineSummary{
						Config: types.VirtualMachineConfigSummary{
							Template: false,
						},
						Runtime: types.VirtualMachineRuntimeInfo{
							PowerState: types.VirtualMachinePowerStatePoweredOn,
						},
					},
					Config: &types.VirtualMachineConfigInfo{
						Annotation: `{"model": false, "to_delete": false, "worker_model_path": "someting"}`,
					},
				},
				{
					ManagedEntity: mo.ManagedEntity{
						Name: "worker1",
					},
					Summary: types.VirtualMachineSummary{
						Config: types.VirtualMachineConfigSummary{
							Template: false,
						},
					},
					Config: &types.VirtualMachineConfigInfo{
						Annotation: `{"model": false, "to_delete": true}`,
					},
				},
			}, nil
		},
	)

	s := h.Status(context.Background())
	t.Logf("status: %+v", s)
	assert.NotNil(t, s)
}

func TestHatcheryVSphere_provisionning_do_nothing(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cdsclient := mock_cdsclient.NewMockInterface(ctrl)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{
				GoRoutines: sdk.NewGoRoutines(),
				Client:     cdsclient,
			},
		},
		Config: HatcheryConfiguration{
			WorkerProvisionning: map[string]int{
				sdk.SharedInfraGroupName + "/" + validModel.Name: 1,
			},
		},
	}

	var now = time.Now()

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
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
						Name: "provisionned_worker",
					},
					Summary: types.VirtualMachineSummary{
						Config: types.VirtualMachineConfigSummary{
							Template: false,
						},
					},
					Config: &types.VirtualMachineConfigInfo{
						Annotation: fmt.Sprintf(`{"worker_model_last_modified": "%d", "model": false, "worker_model_path": "%s", "provisionning": true}`, now.Unix(), sdk.SharedInfraGroupName+"/"+validModel.Name),
					},
					Runtime: types.VirtualMachineRuntimeInfo{
						PowerState: types.VirtualMachinePowerStatePoweredOff,
					},
				},
			}, nil
		},
	)

	cdsclient.EXPECT().WorkerModelGet(sdk.SharedInfraGroupName, validModel.Name).DoAndReturn(
		func(groupName, name string) (sdk.Model, error) {
			return validModel, nil
		},
	)

	h.provisionning(context.Background())
}

func TestHatcheryVSphere_provisionning_start_one(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cdsclient := mock_cdsclient.NewMockInterface(ctrl)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{
				GoRoutines: sdk.NewGoRoutines(),
				Client:     cdsclient,
			},
		},
		Config: HatcheryConfiguration{
			WorkerProvisionning: map[string]int{
				sdk.SharedInfraGroupName + "/" + validModel.Name: 1,
			},
		},
	}

	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.availableIPAddresses = []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"}
	h.Config.Gateway = "192.168.0.254"
	h.Config.DNS = "192.168.0.253"

	var now = time.Now()

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
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
		},
	)

	cdsclient.EXPECT().WorkerModelGet(sdk.SharedInfraGroupName, validModel.Name).DoAndReturn(
		func(groupName, name string) (sdk.Model, error) {
			return validModel, nil
		},
	)

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

	var workerRef types.ManagedObjectReference

	c.EXPECT().CloneVirtualMachine(gomock.Any(), &vmTemplate, &folder, gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, folder *object.Folder, name string, config *types.VirtualMachineCloneSpec) (*types.ManagedObjectReference, error) {
			assert.True(t, strings.HasPrefix(name, "shared-infra-model"))
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

	h.provisionning(context.Background())
}
