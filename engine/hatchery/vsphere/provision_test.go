package vsphere

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/sdk"
)

func TestHatcheryVSphere_reconcileProvisionedVMs_completesProvisioning(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{vSphereClient: c}
	h.GoRoutines = sdk.NewGoRoutines(context.Background())

	ctx := context.Background()

	machines := []mo.VirtualMachine{
		{ // powered on + provisioning → reconciled (reused)
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-orphan"},
			Summary:       types.VirtualMachineSummary{Runtime: types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOn}},
			Config:        &types.VirtualMachineConfigInfo{Annotation: `{"provisioning": true, "vmware_model_path": "model-a"}`},
		},
		{ // already powered off (ready) → ignored
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-ready"},
			Summary:       types.VirtualMachineSummary{Runtime: types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOff}},
			Config:        &types.VirtualMachineConfigInfo{Annotation: `{"provisioning": true, "vmware_model_path": "model-a"}`},
		},
		{ // already tracked as pending → ignored
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-pending"},
			Summary:       types.VirtualMachineSummary{Runtime: types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOn}},
			Config:        &types.VirtualMachineConfigInfo{Annotation: `{"provisioning": true, "vmware_model_path": "model-a"}`},
		},
	}

	h.cacheProvisioning.pending = map[string]string{"provision-v2-pending": "model-a"}

	// Only the orphan is resumed: LoadVirtualMachine → WaitForVirtualMachineIP → ShutdownVirtualMachine.
	vmObj := &object.VirtualMachine{}
	done := make(chan struct{})
	c.EXPECT().LoadVirtualMachine(gomock.Any(), "provision-v2-orphan").Return(vmObj, nil)
	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), vmObj, nil, "provision-v2-orphan").Return(nil)
	c.EXPECT().ShutdownVirtualMachine(gomock.Any(), vmObj).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) error { close(done); return nil },
	)

	h.reconcileProvisionedVMs(ctx, machines, "provision-v2")

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reconcile did not finish provisioning the orphan in time")
	}

	// orphan removed from pending after completion; nothing marked for deletion.
	assert.Eventually(t, func() bool {
		h.cacheProvisioning.mu.Lock()
		_, stillPending := h.cacheProvisioning.pending["provision-v2-orphan"]
		h.cacheProvisioning.mu.Unlock()
		return !stillPending
	}, 2*time.Second, 10*time.Millisecond)

	h.cacheToDelete.mu.Lock()
	defer h.cacheToDelete.mu.Unlock()
	assert.Empty(t, h.cacheToDelete.list)
}

func TestHatcheryVSphere_reconcileProvisionedVMs_deletesOnIPTimeout(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{vSphereClient: c}
	h.GoRoutines = sdk.NewGoRoutines(context.Background())

	ctx := context.Background()

	machines := []mo.VirtualMachine{
		{ // no IP → WaitForVirtualMachineIP fails → marked for deletion
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-stuck"},
			Summary:       types.VirtualMachineSummary{Runtime: types.VirtualMachineRuntimeInfo{PowerState: types.VirtualMachinePowerStatePoweredOn}},
			Config:        &types.VirtualMachineConfigInfo{Annotation: `{"provisioning": true, "vmware_model_path": "model-a"}`},
		},
	}

	vmObj := &object.VirtualMachine{}
	c.EXPECT().LoadVirtualMachine(gomock.Any(), "provision-v2-stuck").Return(vmObj, nil)
	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), vmObj, nil, "provision-v2-stuck").Return(fmt.Errorf("context deadline exceeded"))
	// markToDelete reloads the VM list.
	c.EXPECT().ListVirtualMachines(gomock.Any()).Return(machines, nil).AnyTimes()

	h.reconcileProvisionedVMs(ctx, machines, "provision-v2")

	assert.Eventually(t, func() bool {
		h.cacheToDelete.mu.Lock()
		defer h.cacheToDelete.mu.Unlock()
		return sdk.IsInArray("provision-v2-stuck", h.cacheToDelete.list)
	}, 5*time.Second, 10*time.Millisecond)
}
