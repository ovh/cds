package vsphere

import (
	"context"
	"fmt"
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
	var validModel = sdk.Model{Type: sdk.VSphere, ModelVirtualMachine: sdk.ModelVirtualMachine{
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
			},
		}, nil
	})
	assert.False(t, h.CanSpawn(ctx, &validModel, 0, []sdk.Requirement{}), "with a 'register' vm, it should return False")

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
	h := HatcheryVSphere{
		vSphereClient: c,
	}

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
						Annotation: `{"model": false, "to_delete": true}`,
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

	h.killAwolServers(context.Background())
}
