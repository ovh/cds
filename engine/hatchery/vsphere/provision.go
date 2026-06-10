package vsphere

import (
	"context"
	"strings"
	"sync"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/namesgenerator"
)

// provisionTask represents a unit of provisioning work submitted to the pool.
type provisionTask struct {
	// For "clone" tasks: clone a new VM then finish provisioning
	cloneV2 string // non-empty for V2 clone tasks (vmware model path)

	// For "finish" tasks: the VM already exists, just wait IP + shutdown
	finishVM string // non-nil for reconciled VMs (name of existing powered-on VM)

	wg *sync.WaitGroup // shared per-batch WaitGroup, caller calls wg.Wait()
}

// provisioningPool manages a fixed set of long-lived worker goroutines that
// execute provisioning tasks. Created at startup, shut down on context cancel.
type provisioningPool struct {
	tasks chan provisionTask
}

// startProvisioningPool launches the worker pool. Workers are long-lived and
// pull tasks from the channel until the context is cancelled.
func (h *HatcheryVSphere) startProvisioningPool(ctx context.Context) {
	poolSize := h.Config.WorkerProvisioningPoolSize
	if poolSize == 0 {
		poolSize = 1
	}

	h.provisioningPool = &provisioningPool{
		tasks: make(chan provisionTask, poolSize*2),
	}

	for i := 0; i < poolSize; i++ {
		h.GoRoutines.Run(ctx, "hatchery-vsphere-provisioning-worker",
			func(ctx context.Context) {
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
			},
		)
	}
}

// executeProvisionTask runs a single provisioning task.
func (h *HatcheryVSphere) executeProvisionTask(ctx context.Context, task provisionTask) {
	defer task.wg.Done()

	if task.finishVM != "" {
		// Reconciled VM: just finish provisioning (wait IP + shutdown)
		h.finishProvisioning(ctx, task.finishVM)
		return
	}

	// Clone task: generate name, clone the VM, then finish provisioning
	var workerName string
	var err error

	if task.cloneV2 != "" {
		workerName = namesgenerator.GenerateWorkerName("provision-v2")

		h.cacheProvisioning.mu.Lock()
		h.cacheProvisioning.pending = append(h.cacheProvisioning.pending, workerName)
		h.cacheProvisioning.mu.Unlock()

		err = h.ProvisionWorkerV2(ctx, task.cloneV2, workerName)
		if err != nil {
			ctx = log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to provision vmware worker v2 %q model %q: %v", workerName, task.cloneV2, err)
		}
	}

	if err != nil {
		h.markToDelete(ctx, workerName)
		h.cacheProvisioning.mu.Lock()
		h.cacheProvisioning.pending = sdk.DeleteFromArray(h.cacheProvisioning.pending, workerName)
		h.cacheProvisioning.mu.Unlock()
		return
	}

	// Clone succeeded — now wait for IP and shut down
	h.finishProvisioning(ctx, workerName)
}

// reconcileProvisionedVMs detects provisioned VMs that are powered on but not
// actively being provisioned (not in cacheProvisioning.pending). For each
// orphaned VM it submits a "finish" task to the pool via the provided batch.
func (h *HatcheryVSphere) reconcileProvisionedVMs(ctx context.Context, machines []mo.VirtualMachine, prefix string, batch *sync.WaitGroup) {
	h.cacheProvisioning.mu.Lock()
	pending := make([]string, len(h.cacheProvisioning.pending))
	copy(pending, h.cacheProvisioning.pending)
	h.cacheProvisioning.mu.Unlock()

	for _, machine := range machines {
		if !strings.HasPrefix(machine.Name, prefix) {
			continue
		}

		if machine.Summary.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOn {
			continue
		}

		annot := getVirtualMachineCDSAnnotation(ctx, machine)
		if annot == nil {
			continue
		}
		if annot.HatcheryName != h.Name() {
			continue
		}
		if !annot.Provisioning {
			continue
		}

		if sdk.IsInArray(machine.Name, pending) {
			continue
		}

		vmName := machine.Name
		log.Info(ctx, "reconcileProvisionedVMs: resuming provisioning for VM %q", vmName)

		h.cacheProvisioning.mu.Lock()
		h.cacheProvisioning.pending = append(h.cacheProvisioning.pending, vmName)
		h.cacheProvisioning.mu.Unlock()

		batch.Add(1)
		h.provisioningPool.tasks <- provisionTask{
			finishVM: vmName,
			wg:       batch,
		}
	}
}

// finishProvisioning completes the provisioning of an already-cloned VM by
// waiting for it to get an IP and then shutting it down. On failure the VM is
// marked for deletion. In all cases the VM name is removed from
// cacheProvisioning.pending when done.
func (h *HatcheryVSphere) finishProvisioning(ctx context.Context, vmName string) {
	defer func() {
		h.cacheProvisioning.mu.Lock()
		h.cacheProvisioning.pending = sdk.DeleteFromArray(h.cacheProvisioning.pending, vmName)
		h.cacheProvisioning.mu.Unlock()
	}()

	vm, err := h.vSphereClient.LoadVirtualMachine(ctx, vmName)
	if err != nil {
		log.Error(ctx, "finishProvisioning: unable to load VM %q: %v", vmName, err)
		h.markToDelete(ctx, vmName)
		return
	}

	if err := h.vSphereClient.WaitForVirtualMachineIP(ctx, vm, nil, vmName); err != nil {
		log.Warn(ctx, "finishProvisioning: VM %q failed to get IP, marking for deletion: %v", vmName, err)
		h.markToDelete(ctx, vmName)
		return
	}

	if err := h.vSphereClient.ShutdownVirtualMachine(ctx, vm); err != nil {
		log.Warn(ctx, "finishProvisioning: unable to shutdown VM %q: %v", vmName, err)
		h.markToDelete(ctx, vmName)
		return
	}

	log.Info(ctx, "finishProvisioning: VM %q provisioning completed", vmName)
}
