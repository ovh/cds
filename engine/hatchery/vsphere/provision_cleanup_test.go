package vsphere

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"
)

// setupTestPool creates a provisioning pool with a single worker for testing.
func setupTestPool(ctx context.Context, h *HatcheryVSphere) {
	h.provisioningPool = &provisioningPool{
		tasks: make(chan provisionTask, 10),
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case task, ok := <-h.provisioningPool.tasks:
				if !ok {
					return
				}
				h.executeProvisionTask(ctx, task)
			}
		}
	}()
}

func TestHatcheryVSphere_reconcileProvisionedVMs_completesProvisioning(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupTestPool(ctx, &h)

	stuckCreatedAt := time.Now().Add(-15 * time.Minute)

	machines := []mo.VirtualMachine{
		{
			// Powered on, provisioning annotation — should be reconciled
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-orphan"},
			Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOn},
			Config: &types.VirtualMachineConfigInfo{
				Annotation: fmt.Sprintf(`{"provisioning": true, "vmware_model_path": "model-a", "created": "%s"}`, stuckCreatedAt.Format(time.RFC3339)),
			},
		},
		{
			// Already powered off → ignored
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-ready"},
			Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOff},
			Config: &types.VirtualMachineConfigInfo{
				Annotation: fmt.Sprintf(`{"provisioning": true, "vmware_model_path": "model-a", "created": "%s"}`, stuckCreatedAt.Format(time.RFC3339)),
			},
		},
		{
			// In pending cache → ignored
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-pending"},
			Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOn},
			Config: &types.VirtualMachineConfigInfo{
				Annotation: fmt.Sprintf(`{"provisioning": true, "vmware_model_path": "model-a", "created": "%s"}`, stuckCreatedAt.Format(time.RFC3339)),
			},
		},
	}

	h.cacheProvisioning.pending = []string{"provision-v2-pending"}

	// The worker will: LoadVirtualMachine → WaitForVirtualMachineIP → ShutdownVirtualMachine
	vmObj := &object.VirtualMachine{}
	c.EXPECT().LoadVirtualMachine(gomock.Any(), "provision-v2-orphan").Return(vmObj, nil)
	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), vmObj, nil, "provision-v2-orphan").Return(nil)
	c.EXPECT().ShutdownVirtualMachine(gomock.Any(), vmObj).Return(nil)

	var batch sync.WaitGroup
	h.reconcileProvisionedVMs(ctx, machines, "provision-v2", &batch)
	batch.Wait()

	// VM should have been removed from pending after completion
	h.cacheProvisioning.mu.Lock()
	assert.NotContains(t, h.cacheProvisioning.pending, "provision-v2-orphan")
	h.cacheProvisioning.mu.Unlock()

	// No VMs should be marked for deletion — provisioning completed successfully
	h.cacheToDelete.mu.Lock()
	defer h.cacheToDelete.mu.Unlock()
	assert.Empty(t, h.cacheToDelete.list)
}

func TestHatcheryVSphere_reconcileProvisionedVMs_deletesOnIPTimeout(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupTestPool(ctx, &h)

	stuckCreatedAt := time.Now().Add(-15 * time.Minute)

	machines := []mo.VirtualMachine{
		{
			// No IP, WaitForVirtualMachineIP will fail → marked for deletion
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-stuck"},
			Runtime:       types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOn},
			Config: &types.VirtualMachineConfigInfo{
				Annotation: fmt.Sprintf(`{"provisioning": true, "vmware_model_path": "model-a", "created": "%s"}`, stuckCreatedAt.Format(time.RFC3339)),
			},
		},
	}

	vmObj := &object.VirtualMachine{}
	c.EXPECT().LoadVirtualMachine(gomock.Any(), "provision-v2-stuck").Return(vmObj, nil)
	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), vmObj, nil, "provision-v2-stuck").Return(fmt.Errorf("context deadline exceeded"))

	// markToDelete calls ListVirtualMachines
	c.EXPECT().ListVirtualMachines(gomock.Any()).Return(machines, nil).AnyTimes()

	var batch sync.WaitGroup
	h.reconcileProvisionedVMs(ctx, machines, "provision-v2", &batch)
	batch.Wait()

	// The stuck VM should be marked for deletion
	h.cacheToDelete.mu.Lock()
	defer h.cacheToDelete.mu.Unlock()
	assert.Equal(t, []string{"provision-v2-stuck"}, h.cacheToDelete.list)
}
