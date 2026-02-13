package vsphere

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.opencensus.io/stats/view"
	"go.uber.org/mock/gomock"
)

func TestCollectVSphereMetrics(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}
	h.Config.Name = "test-hatchery"
	h.Common.Common.ServiceName = "test-hatchery"

	ctx := context.Background()

	// Init metrics measures and views
	h.initVSphereMetricsMeasures()
	// Register views directly (no telemetry exporter needed in tests)
	require.NoError(t, view.Register(h.allViews()...))
	t.Cleanup(func() {
		view.Unregister(h.allViews()...)
	})

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
			// Worker owned by this hatchery: 4 vCPUs, 8192 MB
			{
				ManagedEntity: mo.ManagedEntity{Name: "worker-abc"},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template:     false,
						NumCpu:       4,
						MemorySizeMB: 8192,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: string(workerAnnot),
				},
			},
			// Provisioned VM owned by this hatchery: 2 vCPUs, 4096 MB
			{
				ManagedEntity: mo.ManagedEntity{Name: "provision-v2-xxx"},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template:     false,
						NumCpu:       2,
						MemorySizeMB: 4096,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: string(provisionAnnot),
				},
			},
			// Template VM owned by this hatchery: 2 vCPUs, 4096 MB
			{
				ManagedEntity: mo.ManagedEntity{Name: "model-debian12"},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template:     false,
						NumCpu:       2,
						MemorySizeMB: 4096,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: string(templateAnnot),
				},
			},
			// VM owned by another hatchery: 8 vCPUs, 16384 MB
			{
				ManagedEntity: mo.ManagedEntity{Name: "worker-xyz"},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template:     false,
						NumCpu:       8,
						MemorySizeMB: 16384,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: string(otherAnnot),
				},
			},
			// Non-CDS VM (no annotation): 16 vCPUs, 32768 MB
			{
				ManagedEntity: mo.ManagedEntity{Name: "infrastructure-vm"},
				Summary: types.VirtualMachineSummary{
					Config: types.VirtualMachineConfigSummary{
						Template:     false,
						NumCpu:       16,
						MemorySizeMB: 32768,
					},
				},
			},
		}, nil
	}).AnyTimes()

	c.EXPECT().LoadResourcePool(gomock.Any()).DoAndReturn(func(ctx context.Context) (*object.ResourcePool, error) {
		return nil, assert.AnError
	}).AnyTimes()

	// Run collection (Resource Pool will fail gracefully)
	h.collectVSphereMetrics(ctx)

	// Verify Level 3: Global pool counts ALL VMs
	// 4 + 2 + 2 + 8 + 16 = 32 vCPUs
	// 8192 + 4096 + 4096 + 16384 + 32768 = 65536 MB
	// 5 VMs total
	assertLastMetricValue(t, "cds/hatchery/vsphere/pool_total_vcpus", 32)
	assertLastMetricValue(t, "cds/hatchery/vsphere/pool_total_memory_mb", 65536)
	assertLastMetricValue(t, "cds/hatchery/vsphere/pool_total_vm_count", 5)

	// Verify Level 2: Hatchery aggregate (only this hatchery, excludes templates)
	// worker-abc(4) + provision-v2-xxx(2) = 6 vCPUs
	// worker-abc(8192) + provision-v2-xxx(4096) = 12288 MB
	assertLastMetricValue(t, "cds/hatchery/vsphere/allocated_vcpus", 6)
	assertLastMetricValue(t, "cds/hatchery/vsphere/allocated_memory_mb", 12288)
	assertLastMetricValue(t, "cds/hatchery/vsphere/vm_count", 2)
	assertLastMetricValue(t, "cds/hatchery/vsphere/provisioned_vm_count", 1)

	// Verify Level 2: Template metrics
	assertLastMetricValue(t, "cds/hatchery/vsphere/template_vcpus", 2)
	assertLastMetricValue(t, "cds/hatchery/vsphere/template_memory_mb", 4096)
	assertLastMetricValue(t, "cds/hatchery/vsphere/template_count", 1)
}

func assertLastMetricValue(t *testing.T, metricName string, expected int64) {
	t.Helper()
	rows, err := view.RetrieveData(metricName)
	if err != nil {
		t.Errorf("failed to retrieve metric %q: %v", metricName, err)
		return
	}
	if len(rows) == 0 {
		t.Errorf("no data for metric %q", metricName)
		return
	}
	lastValue, ok := rows[len(rows)-1].Data.(*view.LastValueData)
	if !ok {
		t.Errorf("metric %q is not a LastValue gauge", metricName)
		return
	}
	assert.Equal(t, float64(expected), lastValue.Value, "metric %s", metricName)
}
