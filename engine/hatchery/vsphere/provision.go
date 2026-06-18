package vsphere

import (
	"context"
	"strings"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk/namesgenerator"
)

// requestProvisioning asks the provisioning loop to run a refill pass now instead
// of waiting for the next tick. It is non-blocking: if a refill is already pending
// (or provisioning is not enabled), the request is dropped — the pending run will
// pick up the latest state anyway.
func (h *HatcheryVSphere) requestProvisioning(ctx context.Context) {
	if h.provisionSignal == nil {
		return
	}
	select {
	case h.provisionSignal <- struct{}{}:
		log.Debug(ctx, "provisioning refill requested")
	default:
		// a refill is already pending
	}
}

// addPending records an in-flight provision (vm name -> vmware model) so a
// concurrent/subsequent provisioning pass does not re-create a clone whose VM is
// not yet visible in the vSphere inventory. The map is an accelerator only and is
// safe to lose on restart (state is rebuilt from vSphere — see provisioningV2).
func (h *HatcheryVSphere) addPending(name, model string) {
	h.cacheProvisioning.mu.Lock()
	if h.cacheProvisioning.pending == nil {
		h.cacheProvisioning.pending = map[string]string{}
	}
	h.cacheProvisioning.pending[name] = model
	h.cacheProvisioning.mu.Unlock()
}

func (h *HatcheryVSphere) removePending(name string) {
	h.cacheProvisioning.mu.Lock()
	delete(h.cacheProvisioning.pending, name)
	h.cacheProvisioning.mu.Unlock()
}

// acquireProvisionSlot / releaseProvisionSlot bound the number of concurrent
// provisioning operations when WorkerProvisioningPoolSize > 0. When it is 0 the
// semaphore is nil and provisioning runs fully in parallel (maximize throughput).
func (h *HatcheryVSphere) acquireProvisionSlot() {
	if h.provisionSem != nil {
		h.provisionSem <- struct{}{}
	}
}

func (h *HatcheryVSphere) releaseProvisionSlot() {
	if h.provisionSem != nil {
		<-h.provisionSem
	}
}

// startProvisionClone clones a new provision for the model and finishes it
// (wait IP + shutdown), tracked as pending for its whole lifetime. Fire-and-forget:
// it returns immediately so the provisioning loop stays responsive.
func (h *HatcheryVSphere) startProvisionClone(ctx context.Context, model string) {
	name := namesgenerator.GenerateWorkerName("provision-v2")
	h.addPending(name, model)

	// Exec (not Run): these are short-lived, fire-and-forget routines, so they
	// must not be registered in the long-lived goroutine monitoring status.
	h.GoRoutines.Exec(ctx, "hatchery-vsphere-provision-clone", func(ctx context.Context) {
		defer h.removePending(name)
		h.acquireProvisionSlot()
		defer h.releaseProvisionSlot()

		if err := h.ProvisionWorkerV2(ctx, model, name); err != nil {
			ctx = log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to provision vmware worker v2 %q model %q: %v", name, model, err)
			h.markToDelete(ctx, name)
			return
		}
		h.finishProvisioning(ctx, name)
	})
}

// startProvisionFinish resumes finishing an already-cloned provision VM (used by
// reconciliation, e.g. after a restart). Tracked as pending; fire-and-forget.
func (h *HatcheryVSphere) startProvisionFinish(ctx context.Context, name, model string) {
	h.addPending(name, model)

	// Exec (not Run): short-lived, fire-and-forget; not monitored like long-lived routines.
	h.GoRoutines.Exec(ctx, "hatchery-vsphere-provision-finish", func(ctx context.Context) {
		defer h.removePending(name)
		h.acquireProvisionSlot()
		defer h.releaseProvisionSlot()

		h.finishProvisioning(ctx, name)
	})
}

// reconcileProvisionedVMs detects provisioned VMs that are powered on but not
// actively being provisioned (not in cacheProvisioning.pending) — typically
// in-flight provisions orphaned by a hatchery restart — and resumes (reuses)
// them by finishing their provisioning.
func (h *HatcheryVSphere) reconcileProvisionedVMs(ctx context.Context, machines []mo.VirtualMachine, prefix string) {
	h.cacheProvisioning.mu.Lock()
	pending := make(map[string]struct{}, len(h.cacheProvisioning.pending))
	for name := range h.cacheProvisioning.pending {
		pending[name] = struct{}{}
	}
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

		if _, ok := pending[machine.Name]; ok {
			continue
		}

		log.Info(ctx, "reconcileProvisionedVMs: resuming provisioning for VM %q", machine.Name)
		h.startProvisionFinish(ctx, machine.Name, annot.VMwareModelPath)
	}
}

// finishProvisioning completes the provisioning of an already-cloned VM by
// waiting for it to get an IP and then shutting it down. On failure the VM is
// marked for deletion. Pending bookkeeping is owned by the caller
// (startProvisionClone / startProvisionFinish).
func (h *HatcheryVSphere) finishProvisioning(ctx context.Context, vmName string) {
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
