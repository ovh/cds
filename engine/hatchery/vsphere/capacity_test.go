package vsphere

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"
)

func TestCountAllocatedResources(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}
	h.Config.Name = "test-hatchery"
	h.Common.Common.ServiceName = "test-hatchery"

	ctx := context.Background()

	workerAnnot, _ := json.Marshal(annotation{
		HatcheryName:    "test-hatchery",
		WorkerName:      "worker-abc",
		WorkerModelPath: "myorg/mymodel",
	})
	provisionAnnot, _ := json.Marshal(annotation{
		HatcheryName: "test-hatchery",
		Provisioning: true,
	})
	templateAnnot, _ := json.Marshal(annotation{
		HatcheryName: "test-hatchery",
		Model:        true,
	})
	otherAnnot, _ := json.Marshal(annotation{
		HatcheryName: "other-hatchery",
		WorkerName:   "worker-xyz",
	})

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			// Worker owned by this hatchery: 4 vCPUs, 8192 MB — powered ON → counted
			{
				ManagedEntity: mo.ManagedEntity{Name: "worker-abc"},
				Summary: types.VirtualMachineSummary{
					Runtime: types.VirtualMachineRuntimeInfo{
						PowerState: types.VirtualMachinePowerStatePoweredOn,
					},
					Config: types.VirtualMachineConfigSummary{
						NumCpu:       4,
						MemorySizeMB: 8192,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: string(workerAnnot),
				},
			},
			// Provisioned VM owned by this hatchery: 2 vCPUs, 4096 MB — powered OFF → EXCLUDED
			// Powered-off VMs don't consume CPU/RAM in vSphere Resource Pools
			{
				ManagedEntity: mo.ManagedEntity{Name: "provision-v2-xxx"},
				Summary: types.VirtualMachineSummary{
					Runtime: types.VirtualMachineRuntimeInfo{
						PowerState: types.VirtualMachinePowerStatePoweredOff,
					},
					Config: types.VirtualMachineConfigSummary{
						NumCpu:       2,
						MemorySizeMB: 4096,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: string(provisionAnnot),
				},
			},
			// Template VM owned by this hatchery: 2 vCPUs, 4096 MB — EXCLUDED (model=true)
			{
				ManagedEntity: mo.ManagedEntity{Name: "model-debian12"},
				Summary: types.VirtualMachineSummary{
					Runtime: types.VirtualMachineRuntimeInfo{
						PowerState: types.VirtualMachinePowerStatePoweredOff,
					},
					Config: types.VirtualMachineConfigSummary{
						NumCpu:       2,
						MemorySizeMB: 4096,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: string(templateAnnot),
				},
			},
			// VM owned by another hatchery: 8 vCPUs, 16384 MB — EXCLUDED (other hatchery)
			{
				ManagedEntity: mo.ManagedEntity{Name: "worker-xyz"},
				Summary: types.VirtualMachineSummary{
					Runtime: types.VirtualMachineRuntimeInfo{
						PowerState: types.VirtualMachinePowerStatePoweredOn,
					},
					Config: types.VirtualMachineConfigSummary{
						NumCpu:       8,
						MemorySizeMB: 16384,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: string(otherAnnot),
				},
			},
		}, nil
	}).AnyTimes()

	cpus, mem := h.countAllocatedResources(ctx)

	// Only worker-abc (4 vCPUs, 8192 MB) — provision-v2-xxx is powered-off, excluded
	assert.Equal(t, 4, cpus, "expected 4 vCPUs")
	assert.Equal(t, 8192, mem, "expected 8192 MB")
}
