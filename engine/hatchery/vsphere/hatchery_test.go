package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	sdkhatchery "github.com/ovh/cds/sdk/hatchery"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"
)

func TestHatcheryVSphere_CanSpawn_RejectsV1(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	var ctx = context.Background()
	var validModel = sdk.Model{
		Name: "model",
		Type: sdk.VSphere,
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Cmd: "cmd",
		}}

	// Worker model v1 is no longer supported on vSphere.
	can := h.CanSpawn(ctx, sdk.WorkerStarterWorkerModel{ModelV1: &validModel}, "1", []sdk.Requirement{})
	assert.False(t, can, "v1 worker models should always be rejected")
}

func TestHatcheryVSphere_CanSpawnv2(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	var ctx = context.Background()
	var validModel = sdk.V2WorkerModel{
		Name: "model",
		Type: sdk.WorkerModelTypeVSphere,
	}
	starter := func() sdk.WorkerStarterWorkerModel {
		return sdk.WorkerStarterWorkerModel{
			ModelV2: &validModel,
			Cmd:     "cmd",
			VSphereSpec: sdk.V2WorkerModelVSphereSpec{
				Image: "the-model",
			},
		}
	}

	// A provisioned VM ready to be claimed for the model: powered off,
	// provisioning annotation matching the VMware model path.
	readyProvision := mo.VirtualMachine{
		ManagedEntity: mo.ManagedEntity{
			Name: "provision-v2-worker",
		},
		Summary: types.VirtualMachineSummary{
			Config: types.VirtualMachineConfigSummary{
				Template: false,
			},
			Runtime: types.VirtualMachineRuntimeInfo{
				PowerState: types.VirtualMachinePowerStatePoweredOff,
			},
		},
		Config: &types.VirtualMachineConfigInfo{
			Annotation: `{"provisioning": true, "vmware_model_path": "the-model"}`,
		},
	}

	can := h.CanSpawn(ctx, starter(), "1", []sdk.Requirement{{Type: sdk.ServiceRequirement}})
	assert.False(t, can, "with a service requirement, it should return False")

	can = h.CanSpawn(ctx, starter(), "1", []sdk.Requirement{{Type: sdk.MemoryRequirement}})
	assert.False(t, can, "with a memory requirement, it should return False")

	can = h.CanSpawn(ctx, starter(), "1", []sdk.Requirement{{Type: sdk.HostnameRequirement}})
	assert.False(t, can, "with a hostname requirement, it should return False")

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
					Annotation: `{"job_id": "1"}`,
				},
			},
			readyProvision,
		}, nil
	})

	can = h.CanSpawn(ctx, starter(), "1", []sdk.Requirement{})
	assert.False(t, can, "it should return False, because there is a worker for the same job")

	// duplicate-job check + provisioned-worker check both list the VMs
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
					Annotation: `{"job_id": "2"}`,
				},
			},
			readyProvision,
		}, nil
	}).Times(2)

	can = h.CanSpawn(ctx, starter(), "1", []sdk.Requirement{})
	assert.True(t, can, "it should return True, a provisioned worker is available for the model")

	// a powered-on provision is not ready to be claimed
	startingProvision := readyProvision
	startingProvision.Summary.Runtime.PowerState = types.VirtualMachinePowerStatePoweredOn
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{startingProvision}, nil
	}).Times(2)
	can = h.CanSpawn(ctx, starter(), "1", []sdk.Requirement{})
	assert.False(t, can, "it should return False, the provisioned worker is not powered off yet")

	// without any provisioned VM for the model, it can't spawn
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{}, nil
	}).AnyTimes()
	can = h.CanSpawn(ctx, starter(), "0", []sdk.Requirement{})
	assert.False(t, can, "it should return False, no provisioned worker is available for the model")

	h.cachePendingJobID.list = append(h.cachePendingJobID.list, "666")
	can = h.CanSpawn(ctx, starter(), "666", []sdk.Requirement{})
	assert.False(t, can, "it should return False because the jobID is still in the local cache")
}

func TestHatcheryVSphere_NeedRegistration(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	var ctx = context.Background()
	var validModel = sdk.Model{
		Name: "model",
		Type: sdk.VSphere,
	}

	// NeedRegistration is now always false: vSphere no longer supports v1 model registration.
	assert.False(t, h.NeedRegistration(ctx, &validModel), "NeedRegistration must always return false")
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
	).AnyTimes()

	h.killDisabledWorkers(context.Background())

	assert.Equal(t, 1, len(h.cacheToDelete.list))
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
	h.Config.WorkerTTL = 5
	h.Config.WorkerRegistrationTTL = 5
	h.Config.FinishedWorkerGracePeriod = 300 // 5 min

	now := time.Now()
	old := now.Add(-10 * time.Minute)

	annot := func(a annotation) string {
		b, err := json.Marshal(a)
		require.NoError(t, err)
		return string(b)
	}

	// worker0, worker1 and worker6 are registered on the API side.
	cdsclient.EXPECT().WorkerList(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]sdk.Worker, error) {
			return []sdk.Worker{{Name: "worker0"}, {Name: "worker1"}, {Name: "worker6"}}, nil
		},
	)

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			poweredOn := types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOn}
			poweredOff := types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOff}
			return []mo.VirtualMachine{
				{ // running worker, recently started, on API → KEEP
					ManagedEntity: mo.ManagedEntity{Name: "worker0"},
					Summary:       types.VirtualMachineSummary{Runtime: poweredOn},
					Config:        &types.VirtualMachineConfigInfo{Annotation: annot(annotation{WorkerStartTime: now})},
				},
				{ // running worker, started long ago, on API → DELETE (WorkerTTL)
					ManagedEntity: mo.ManagedEntity{Name: "worker1"},
					Summary:       types.VirtualMachineSummary{Runtime: poweredOn},
					Config:        &types.VirtualMachineConfigInfo{Annotation: annot(annotation{WorkerStartTime: old})},
				},
				{ // pooled provision → SKIP (provision- prefix)
					ManagedEntity: mo.ManagedEntity{Name: "provision-worker2"},
					Summary:       types.VirtualMachineSummary{Runtime: poweredOff},
					Config:        &types.VirtualMachineConfigInfo{Annotation: annot(annotation{WorkerName: "provision-worker2", Provisioning: true})},
				},
				{ // finished/failed worker, powered off, old start → DELETE
					ManagedEntity: mo.ManagedEntity{Name: "worker3"},
					Summary:       types.VirtualMachineSummary{Runtime: poweredOff},
					Config:        &types.VirtualMachineConfigInfo{Annotation: annot(annotation{WorkerStartTime: old})},
				},
				{ // legacy worker (no WorkerStartTime), powered off, old CreateDate → DELETE
					ManagedEntity: mo.ManagedEntity{Name: "worker4"},
					Summary:       types.VirtualMachineSummary{Runtime: poweredOff},
					Config:        &types.VirtualMachineConfigInfo{Annotation: annot(annotation{}), CreateDate: &old},
				},
				{ // just-claimed worker, powered off but recent start, not on API → KEEP (in-flight)
					ManagedEntity: mo.ManagedEntity{Name: "worker5"},
					Summary:       types.VirtualMachineSummary{Runtime: poweredOff},
					Config:        &types.VirtualMachineConfigInfo{Annotation: annot(annotation{WorkerStartTime: now})},
				},
				{ // finished worker, powered off but still on API, recent start → DELETE NOW
					ManagedEntity: mo.ManagedEntity{Name: "worker6"},
					Summary:       types.VirtualMachineSummary{Runtime: poweredOff},
					Config:        &types.VirtualMachineConfigInfo{Annotation: annot(annotation{WorkerStartTime: now})},
				},
			}, nil
		},
	).Times(1)

	// deleteServer reloads the VM by name; only the deleted ones are loaded.
	var vm1 = object.VirtualMachine{Common: object.Common{InventoryPath: "worker1"}}
	var vm3 = object.VirtualMachine{Common: object.Common{InventoryPath: "worker3"}}
	var vm4 = object.VirtualMachine{Common: object.Common{InventoryPath: "worker4"}}
	var vm6 = object.VirtualMachine{Common: object.Common{InventoryPath: "worker6"}}

	c.EXPECT().LoadVirtualMachine(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vmname string) (*object.VirtualMachine, error) {
			switch vmname {
			case "worker1":
				return &vm1, nil
			case "worker3":
				return &vm3, nil
			case "worker4":
				return &vm4, nil
			case "worker6":
				return &vm6, nil
			}
			return nil, fmt.Errorf("not expected: %s", vmname)
		},
	).Times(4)

	// worker1 is powered on → shutdown then destroy; worker3/worker4/worker6 are
	// powered off → destroy only.
	c.EXPECT().ShutdownVirtualMachine(gomock.Any(), &vm1).Return(nil)
	c.EXPECT().DestroyVirtualMachine(gomock.Any(), &vm1).Return(nil)
	c.EXPECT().DestroyVirtualMachine(gomock.Any(), &vm3).Return(nil)
	c.EXPECT().DestroyVirtualMachine(gomock.Any(), &vm4).Return(nil)
	c.EXPECT().DestroyVirtualMachine(gomock.Any(), &vm6).Return(nil)

	h.killAwolServers(context.Background())
}

func TestHatcheryVSphere_requestProvisioning(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctx := context.Background()

	// nil channel (provisioning disabled): must be a safe no-op.
	h := &HatcheryVSphere{}
	h.requestProvisioning(ctx)

	// buffered size-1: repeated requests coalesce into a single pending refill.
	h.provisionSignal = make(chan struct{}, 1)
	h.requestProvisioning(ctx)
	h.requestProvisioning(ctx)
	h.requestProvisioning(ctx)
	assert.Len(t, h.provisionSignal, 1, "requests must coalesce to a single pending refill")

	<-h.provisionSignal
	assert.Len(t, h.provisionSignal, 0)
}

func TestHatcheryVSphere_Status(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{
				GoRoutines: sdk.NewGoRoutines(context.Background()),
			},
		},
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).Times(2).DoAndReturn(
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

func TestHatcheryVSphere_provisioning_v2_do_nothing(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cdsclient := mock_cdsclient.NewMockInterface(ctrl)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{
				GoRoutines: sdk.NewGoRoutines(context.Background()),
				Client:     cdsclient,
			},
		},
		Config: HatcheryConfiguration{
			WorkerProvisioning: []WorkerProvisioningConfig{
				{
					ModelVMWare: "the-model",
					Number:      1,
				},
			},
		},
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{
				{
					ManagedEntity: mo.ManagedEntity{
						Name: "provision-v2-worker",
					},
					Summary: types.VirtualMachineSummary{
						Config: types.VirtualMachineConfigSummary{
							Template: false,
						},
					},
					Config: &types.VirtualMachineConfigInfo{
						Annotation: `{"model": false, "vmware_model_path": "the-model", "provisioning": true}`,
					},
					Runtime: types.VirtualMachineRuntimeInfo{
						PowerState: types.VirtualMachinePowerStatePoweredOff,
					},
				},
			}, nil
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.provisioningV2(ctx)
}

// provisioningV2 must count both ready (existing) and in-flight (starting,
// tracked in the pending cache) provisions when computing the deficit, so a
// retrigger while clones are still in flight does not double-create.
func TestHatcheryVSphere_provisioningV2_noDoubleCreate(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{GoRoutines: sdk.NewGoRoutines(context.Background())},
		},
		Config: HatcheryConfiguration{
			WorkerProvisioning: []WorkerProvisioningConfig{{ModelVMWare: "the-model", Number: 3}},
		},
	}

	// One ready provision (powered off) already exists in vSphere.
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{{
				ManagedEntity: mo.ManagedEntity{Name: "provision-v2-ready"},
				Summary:       types.VirtualMachineSummary{Config: types.VirtualMachineConfigSummary{Template: false}},
				Config:        &types.VirtualMachineConfigInfo{Annotation: `{"vmware_model_path": "the-model", "provisioning": true}`},
				Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOff},
			}}, nil
		},
	).AnyTimes()

	// One in-flight clone already tracked (not yet visible in the inventory).
	h.cacheProvisioning.pending = map[string]string{"provision-v2-starting": "the-model"}

	// Each launched clone first loads the model template; block it so the clone
	// stays in flight, and count how many clones were launched.
	var cloneCount int64
	release := make(chan struct{})
	c.EXPECT().LoadVirtualMachine(gomock.Any(), "the-model").DoAndReturn(
		func(ctx context.Context, name string) (*object.VirtualMachine, error) {
			atomic.AddInt64(&cloneCount, 1)
			<-release
			return nil, fmt.Errorf("stop the clone here")
		},
	).AnyTimes()

	// deficit = Number(3) − existing(1) − starting(1) = 1
	h.provisioningV2(context.Background())
	assert.Eventually(t, func() bool { return atomic.LoadInt64(&cloneCount) == 1 }, 2*time.Second, 10*time.Millisecond)

	// Retrigger while the clone is still in flight: it is now tracked as pending,
	// so deficit = 3 − 1 − 2 = 0 → no new clone.
	h.provisioningV2(context.Background())
	time.Sleep(200 * time.Millisecond) // give any erroneous extra clone time to start
	assert.Equal(t, int64(1), atomic.LoadInt64(&cloneCount), "a retrigger must not re-create in-flight provisions")

	// Let the blocked clone fail and unwind, and wait for it to drop from pending
	// so its goroutine is done before the test (and gomock controller) tears down.
	close(release)
	assert.Eventually(t, func() bool {
		h.cacheProvisioning.mu.Lock()
		defer h.cacheProvisioning.mu.Unlock()
		return len(h.cacheProvisioning.pending) == 1 // only the manually-seeded "starting" entry remains
	}, 2*time.Second, 10*time.Millisecond)
}

func TestHatcheryVSphere_GetDetaultModelV2Name(t *testing.T) {
	h := HatcheryVSphere{
		Config: HatcheryConfiguration{
			DefaultWorkerModelsV2: []DefaultWorkerModelsV2{
				{
					WorkerModelV2: "the-model-v2",
					Binaries:      []string{"docker"},
				},
			},
		},
	}

	requirements := []sdk.Requirement{
		{
			Name:  "binary",
			Value: "docker",
			Type:  sdk.BinaryRequirement,
		},
	}
	got := h.GetDetaultModelV2Name(context.TODO(), requirements)
	require.Equal(t, "the-model-v2", got)

	got = h.GetDetaultModelV2Name(context.TODO(), []sdk.Requirement{})
	require.Equal(t, "the-model-v2", got)

	got = h.GetDetaultModelV2Name(context.TODO(), []sdk.Requirement{{Name: "foo", Value: "bar", Type: sdk.BinaryRequirement}, {Name: "foo", Value: "docker", Type: sdk.BinaryRequirement}})
	require.Equal(t, "the-model-v2", got)

	got = h.GetDetaultModelV2Name(context.TODO(), []sdk.Requirement{{Name: "foo", Value: "bar", Type: sdk.BinaryRequirement}})
	require.Equal(t, "", got)
}

func TestHatcheryVSphere_provisioning_v2_deprovision_excess(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cdsclient := mock_cdsclient.NewMockInterface(ctrl)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{
				GoRoutines: sdk.NewGoRoutines(context.Background()),
				Client:     cdsclient,
			},
		},
		Config: HatcheryConfiguration{
			WorkerProvisioning: []WorkerProvisioningConfig{
				{
					ModelVMWare: "the-model",
					Number:      1, // decreased from 3 to 1
				},
			},
		},
	}

	// 3 provisioned VMs exist, but config says only 1
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{
				{
					ManagedEntity: mo.ManagedEntity{Name: "provision-v2-worker-1"},
					Summary:       types.VirtualMachineSummary{Config: types.VirtualMachineConfigSummary{Template: false}},
					Config:        &types.VirtualMachineConfigInfo{Annotation: `{"model": false, "vmware_model_path": "the-model", "provisioning": true}`},
					Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOff},
				},
				{
					ManagedEntity: mo.ManagedEntity{Name: "provision-v2-worker-2"},
					Summary:       types.VirtualMachineSummary{Config: types.VirtualMachineConfigSummary{Template: false}},
					Config:        &types.VirtualMachineConfigInfo{Annotation: `{"model": false, "vmware_model_path": "the-model", "provisioning": true}`},
					Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOff},
				},
				{
					ManagedEntity: mo.ManagedEntity{Name: "provision-v2-worker-3"},
					Summary:       types.VirtualMachineSummary{Config: types.VirtualMachineConfigSummary{Template: false}},
					Config:        &types.VirtualMachineConfigInfo{Annotation: `{"model": false, "vmware_model_path": "the-model", "provisioning": true}`},
					Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOff},
				},
			}, nil
		},
	).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.provisioningV2(ctx)

	// 2 excess VMs should be marked for deletion (3 exist - 1 configured = 2 excess)
	h.cacheToDelete.mu.Lock()
	defer h.cacheToDelete.mu.Unlock()
	assert.Len(t, h.cacheToDelete.list, 2)
	assert.Contains(t, h.cacheToDelete.list, "provision-v2-worker-1")
	assert.Contains(t, h.cacheToDelete.list, "provision-v2-worker-2")
}

func TestHatcheryVSphere_provisioning_v2_deprovision_removed_model(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cdsclient := mock_cdsclient.NewMockInterface(ctrl)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Common: hatchery.Common{
			Common: service.Common{
				GoRoutines: sdk.NewGoRoutines(context.Background()),
				Client:     cdsclient,
			},
		},
		Config: HatcheryConfiguration{
			// No models configured - all should be removed
			WorkerProvisioning: []WorkerProvisioningConfig{},
		},
	}

	// 2 provisioned VMs exist for a model that's no longer in config
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{
				{
					ManagedEntity: mo.ManagedEntity{Name: "provision-v2-worker-1"},
					Summary:       types.VirtualMachineSummary{Config: types.VirtualMachineConfigSummary{Template: false}},
					Config:        &types.VirtualMachineConfigInfo{Annotation: `{"model": false, "vmware_model_path": "removed-model", "provisioning": true}`},
					Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOff},
				},
				{
					ManagedEntity: mo.ManagedEntity{Name: "provision-v2-worker-2"},
					Summary:       types.VirtualMachineSummary{Config: types.VirtualMachineConfigSummary{Template: false}},
					Config:        &types.VirtualMachineConfigInfo{Annotation: `{"model": false, "vmware_model_path": "removed-model", "provisioning": true}`},
					Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOff},
				},
			}, nil
		},
	).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.provisioningV2(ctx)

	// Both VMs should be marked for deletion since model is removed from config
	h.cacheToDelete.mu.Lock()
	defer h.cacheToDelete.mu.Unlock()
	assert.Len(t, h.cacheToDelete.list, 2)
}
