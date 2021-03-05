package vsphere

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

type annotation struct {
	HatcheryName            string    `json:"hatchery_name,omitempty"`
	WorkerName              string    `json:"worker_name,omitempty"`
	RegisterOnly            bool      `json:"register_only,omitempty"`
	Provisioning            bool      `json:"provisioning,omitempty"`
	WorkerModelPath         string    `json:"worker_model_path,omitempty"`
	WorkerModelLastModified string    `json:"worker_model_last_modified,omitempty"`
	Model                   bool      `json:"model,omitempty"`
	Created                 time.Time `json:"created,omitempty"`
	JobID                   int64     `json:"job_id,omitempty"`
}

// SpawnWorker creates a new vm instance
func (h *HatcheryVSphere) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) (err error) {
	log.Info(ctx, "SpawnWorker %q", spawnArgs.WorkerName)
	defer func() {
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "HatcheryVSphere> SpawnWorker %q from model %q: ERROR: %v", spawnArgs.WorkerName, spawnArgs.ModelName(), err)

			h.cachePendingJobID.mu.Lock()
			h.cachePendingJobID.list = sdk.DeleteFromInt64Array(h.cachePendingJobID.list, spawnArgs.JobID)
			h.cachePendingJobID.mu.Unlock()
		} else {
			log.Info(ctx, "HatcheryVSphere> SpawnWorker %q from model %q: DONE", spawnArgs.WorkerName, spawnArgs.ModelName())
		}
	}()

	if spawnArgs.JobID == 0 && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

	if spawnArgs.JobID != 0 {
		h.cachePendingJobID.mu.Lock()
		h.cachePendingJobID.list = append(h.cachePendingJobID.list, spawnArgs.JobID)
		defer h.cachePendingJobID.mu.Unlock()

		go func() {
			time.Sleep(3 * time.Minute)
			h.cachePendingJobID.mu.Lock()
			h.cachePendingJobID.list = sdk.DeleteFromInt64Array(h.cachePendingJobID.list, spawnArgs.JobID)
			h.cachePendingJobID.mu.Unlock()
		}()
	}

	var vmTemplate *object.VirtualMachine

	if _, err := h.getVirtualMachineTemplateByName(ctx, spawnArgs.Model.Name); err != nil || spawnArgs.Model.NeedRegistration {
		// Generate worker model vm
		log.Info(ctx, "creating virtual machine model %q", spawnArgs.Model.Name)
		vmTemplate, err = h.createVirtualMachineTemplate(ctx, *spawnArgs.Model, spawnArgs.WorkerName)
		if err != nil {
			log.Error(ctx, "Unable to create VM Model: %v", err)
			return err
		}
	}

	if vmTemplate == nil {
		var err error
		log.Info(ctx, "loading virtual machine template %q", spawnArgs.Model.Name)
		if vmTemplate, err = h.vSphereClient.LoadVirtualMachine(ctx, spawnArgs.Model.Name); err != nil {
			return sdk.WrapError(err, "cannot find virtual machine template with this model")
		}
	}

	// Try to find a provisionned worker
	if !spawnArgs.RegisterOnly {
		provisionnedVMWorker, err := h.FindProvisionnedWorker(ctx, *spawnArgs.Model)
		if err != nil {
			return err
		}

		if provisionnedVMWorker != nil {
			log.Info(ctx, "starting worker %q with provisionned machine %q", spawnArgs.Model.Name, provisionnedVMWorker.Name())

			if err := h.vSphereClient.RenameVirtualMachine(ctx, provisionnedVMWorker, spawnArgs.WorkerName); err != nil {
				return sdk.WrapError(err, "unable to rename VM %q", provisionnedVMWorker.Name())
			}

			// Before restart it, keep it in the cache for a few minutes to avoid the "killAwolServer" to delete it
			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.restarting = append(h.cacheProvisioning.restarting, spawnArgs.WorkerName)
			h.cacheProvisioning.mu.Unlock()

			time.Sleep(2 * time.Second)

			go func() {
				time.Sleep(time.Duration(h.Config.WorkerTTL) * time.Minute)
				h.cacheProvisioning.mu.Lock()
				h.cacheProvisioning.restarting = sdk.DeleteFromArray(h.cacheProvisioning.restarting, spawnArgs.WorkerName)
				h.cacheProvisioning.mu.Unlock()
			}()

			if err := h.vSphereClient.StartVirtualMachine(ctx, provisionnedVMWorker); err != nil {
				_ = h.vSphereClient.ShutdownVirtualMachine(ctx, provisionnedVMWorker)
				h.markToDelete(ctx, provisionnedVMWorker)
				return sdk.WrapError(err, "unable to start VM %q", spawnArgs.WorkerName)
			}

			return h.launchScriptWorker(ctx, spawnArgs.WorkerName, spawnArgs.JobID, spawnArgs.WorkerToken, *spawnArgs.Model, false, provisionnedVMWorker)
		}

	}

	annot := annotation{
		HatcheryName:            h.Name(),
		WorkerName:              spawnArgs.WorkerName,
		RegisterOnly:            spawnArgs.RegisterOnly,
		WorkerModelLastModified: fmt.Sprintf("%d", spawnArgs.Model.UserLastModified.Unix()),
		WorkerModelPath:         spawnArgs.ModelName(),
		Created:                 time.Now(),
		JobID:                   spawnArgs.JobID,
	}

	cloneSpec, err := h.prepareCloneSpec(ctx, vmTemplate, annot, spawnArgs.WorkerName)
	if err != nil {
		return err
	}

	folder, err := h.vSphereClient.LoadFolder(ctx)
	if err != nil {
		return err
	}

	log.Info(ctx, "Create vm to execute worker %q, cloneSpec: %+v", spawnArgs.WorkerName, *cloneSpec)
	defer log.Info(ctx, "Terminate to create vm for worker %s", spawnArgs.WorkerName)

	cloneRef, err := h.vSphereClient.CloneVirtualMachine(ctx, vmTemplate, folder, spawnArgs.WorkerName, cloneSpec)
	if err != nil {
		return err
	}

	vmWorker, err := h.vSphereClient.NewVirtualMachine(ctx, cloneSpec, cloneRef)
	if err != nil {
		return err
	}

	return h.launchScriptWorker(ctx, spawnArgs.WorkerName, spawnArgs.JobID, spawnArgs.WorkerToken, *spawnArgs.Model, spawnArgs.RegisterOnly, vmWorker)
}

// createVirtualMachineTemplate create a model for a specific worker model
func (h *HatcheryVSphere) createVirtualMachineTemplate(ctx context.Context, model sdk.Model, workerName string) (vm *object.VirtualMachine, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	log.Info(ctx, "Create vm model %q from %q", model.Name, model.ModelVirtualMachine.Image)
	defer func() {
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "Create vm model %q from %q ERROR: %v", model.Name, model.ModelVirtualMachine.Image, err)
		} else {
			log.Info(ctx, "Create vm model %q from %q DONE", model.Name, model.ModelVirtualMachine.Image)
		}
	}()

	vm, err = h.vSphereClient.LoadVirtualMachine(ctx, model.ModelVirtualMachine.Image)
	if err != nil {
		return vm, sdk.WrapError(err, "unable to find virtual machine %q", model.ModelVirtualMachine.Image)
	}

	log.Debug(ctx, "found virtual machine image %q: %+v", model.ModelVirtualMachine.Image, vm)

	annot := annotation{
		HatcheryName:            h.Name(),
		WorkerModelLastModified: fmt.Sprintf("%d", model.UserLastModified.Unix()),
		WorkerModelPath:         model.Path(),
		Model:                   true,
		Created:                 time.Now(),
	}

	cloneSpec, err := h.prepareCloneSpec(ctx, vm, annot, workerName)
	if err != nil {
		return nil, sdk.WrapError(err, "createVMModel> cannot create VM configuration")
	}

	name := model.Name + "-tmp"
	log.Info(ctx, "creating worker %q by cloning vm to %q ", workerName, name)

	folder, err := h.vSphereClient.LoadFolder(ctx)
	if err != nil {
		return nil, err
	}

	cloneRef, err := h.vSphereClient.CloneVirtualMachine(ctx, vm, folder, name, cloneSpec)
	if err != nil {
		return nil, err
	}

	clonedVM, err := h.vSphereClient.NewVirtualMachine(ctx, cloneSpec, cloneRef)
	if err != nil {
		return nil, err
	}

	if err := h.checkVirtualMachineIsReady(ctx, model, clonedVM); err != nil {
		log.Error(ctx, "virtual machine %q is not ready: %v", clonedVM.Name(), err)
		log.Warn(ctx, "shutdown virtual machine %q", clonedVM.Name())
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, clonedVM); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", model.Name, err)
		}
		h.markToDelete(ctx, clonedVM)
		return nil, err
	}

	if _, err := h.launchClientOp(ctx, clonedVM, model.ModelVirtualMachine, model.ModelVirtualMachine.PostCmd, nil); err != nil {
		log.Error(ctx, "cannot start program on virtual machine %q: %v", clonedVM.Name(), err)
		log.Warn(ctx, "shutdown virtual machine %q", clonedVM.Name())
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, clonedVM); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", model.Name, err)
		}
		h.markToDelete(ctx, clonedVM)
		return nil, err
	}

	if err := h.vSphereClient.WaitForVirtualMachineShutdown(ctx, clonedVM); err != nil {
		return nil, err
	}

	modelFound, err := h.getVirtualMachineTemplateByName(ctx, model.Name)
	if err == nil {
		if err := h.deleteServer(ctx, modelFound); err != nil {
			log.Warn(ctx, "createVMModel> Cannot delete previous model %s : %s", model.Name, err)
		}
	}

	if err := h.vSphereClient.RenameVirtualMachine(ctx, clonedVM, model.Name); err != nil {
		return nil, err
	}
	log.Debug(ctx, "renaming virtual machine %q to %q: DONE", clonedVM.String(), model.Name)

	log.Info(ctx, "mark virtual machine %q as template %q", name, model.Name)
	if err := h.vSphereClient.MarkVirtualMachineAsTemplate(ctx, clonedVM); err != nil {
		return nil, err
	}

	return vm, nil
}

func (h *HatcheryVSphere) checkVirtualMachineIsReady(ctx context.Context, model sdk.Model, vm *object.VirtualMachine) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var latestError error
	for {
		if ctx.Err() != nil {
			return sdk.WithStack(fmt.Errorf("vm %q is not ready: %v - %v", vm.Name(), latestError, ctx.Err()))
		}
		// Try to run a script
		_, err := h.launchClientOp(ctx, vm, model.ModelVirtualMachine, "env", nil)
		if err != nil {
			log.Warn(ctx, "virtual machine %q is not ready: %v", vm.Name(), err)
			latestError = err
			time.Sleep(time.Second)
			continue // If it failing, wait and retry
		}
		break // else it means that it is ready
	}

	return nil
}

// launchScriptWorker launch a script on the worker
func (h *HatcheryVSphere) launchScriptWorker(ctx context.Context, name string, jobID int64, token string, model sdk.Model, registerOnly bool, vm *object.VirtualMachine) error {
	if err := h.vSphereClient.WaitForVirtualMachineIP(ctx, vm); err != nil {
		return err
	}

	env := []string{
		"CDS_FROM_WORKER_IMAGE=true",
	}

	env = append(env, h.getGraylogEnv(model)...)
	udata := model.ModelVirtualMachine.PreCmd + "\n" + model.ModelVirtualMachine.Cmd

	if registerOnly {
		udata += " register"
	}
	udata += "\n" + model.ModelVirtualMachine.PostCmd

	tmpl, errt := template.New("udata").Parse(udata)
	if errt != nil {
		return errt
	}
	udataParam := sdk.WorkerArgs{
		API:               h.Config.API.HTTP.URL,
		Name:              name,
		Token:             token,
		Model:             model.Group.Name + "/" + model.Name,
		HatcheryName:      h.Name(),
		TTL:               h.Config.WorkerTTL,
		FromWorkerImage:   true,
		GraylogHost:       h.Config.Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Config.Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Config.Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Config.Provision.WorkerLogsOptions.Graylog.ExtraValue,
		WorkflowJobID:     jobID,
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, udataParam); err != nil {
		return err
	}

	if err := h.checkVirtualMachineIsReady(ctx, model, vm); err != nil {
		log.Error(ctx, "virtual machine %q is not ready: %v", vm.Name(), err)
		log.Warn(ctx, "shutdown virtual machine %q", vm.Name())
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, vm); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", model, err)
		}
		h.markToDelete(ctx, vm)
		return err
	}

	if _, err := h.launchClientOp(ctx, vm, model.ModelVirtualMachine, buffer.String(), env); err != nil {
		log.Warn(ctx, "launchScript> cannot start program %s", err)
		log.Error(ctx, "cannot start program on virtual machine %q: %v", vm.Name(), err)
		log.Warn(ctx, "shutdown virtual machine %q", vm.Name())
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, vm); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", model.Name, err)
		}
		h.markToDelete(ctx, vm)
		return err
	}

	return nil
}

func (h *HatcheryVSphere) markToDelete(ctx context.Context, vm *object.VirtualMachine) {
	h.cacheToDelete.mu.Lock()
	defer h.cacheToDelete.mu.Unlock()

	// Reload the vm ref to get the annotation
	allVMRef, err := h.vSphereClient.ListVirtualMachines(ctx)
	if err != nil {
		log.Error(ctx, "unable to get virtual machines: %v", err)
		return
	}

	var vmRef *mo.VirtualMachine
	for i := range allVMRef {
		if allVMRef[i].Name == vm.Name() {
			vmRef = &allVMRef[i]
			break
		}
	}

	if vmRef == nil {
		err := sdk.WithStack(fmt.Errorf("virtual machine ref %q not found", vm.Name()))
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "unable to get virtual machines: %v", err)
		return
	}

	var annot = getVirtualMachineCDSAnnotation(ctx, *vmRef)
	if annot == nil {
		return
	}

	h.cacheToDelete.list = append(h.cacheToDelete.list, vmRef.Name)
}

const maxLength = 63

func (h *HatcheryVSphere) ProvisionWorker(ctx context.Context, m sdk.Model, workerName string) (err error) {
	vmTemplate, err := h.vSphereClient.LoadVirtualMachine(ctx, m.Name)
	if err != nil {
		return sdk.WrapError(err, "cannot find virtual machine template with this model")
	}

	annot := annotation{
		HatcheryName:            h.Name(),
		WorkerName:              workerName,
		RegisterOnly:            false,
		Provisioning:            true,
		WorkerModelLastModified: fmt.Sprintf("%d", m.UserLastModified.Unix()),
		WorkerModelPath:         m.Group.Name + "/" + m.Name,
		Created:                 time.Now(),
	}

	cloneSpec, err := h.prepareCloneSpec(ctx, vmTemplate, annot, workerName)
	if err != nil {
		return err
	}

	folder, err := h.vSphereClient.LoadFolder(ctx)
	if err != nil {
		return err
	}

	log.Info(ctx, "provisioning %q by cloning %q", workerName, vmTemplate.Name())

	cloneRef, err := h.vSphereClient.CloneVirtualMachine(ctx, vmTemplate, folder, workerName, cloneSpec)
	if err != nil {
		return err
	}

	clonedVM, err := h.vSphereClient.NewVirtualMachine(ctx, cloneSpec, cloneRef)
	if err != nil {
		return err
	}

	if err := h.vSphereClient.WaitForVirtualMachineIP(ctx, clonedVM); err != nil {
		return err
	}

	// the provisionned workers are shutdown when they are created
	if err := h.vSphereClient.ShutdownVirtualMachine(ctx, clonedVM); err != nil {
		return err
	}

	log.Info(ctx, "vm %q has been provisionned", workerName)

	return nil
}

func (h *HatcheryVSphere) FindProvisionnedWorker(ctx context.Context, m sdk.Model) (*object.VirtualMachine, error) {
	var expectedModelPath = m.Group.Name + "/" + m.Name

	log.Debug(ctx, "searching for provisionned VM for model %q", expectedModelPath)

	machines := h.getVirtualMachines(ctx)
	for _, machine := range machines {
		annot := getVirtualMachineCDSAnnotation(ctx, machine)
		if annot == nil {
			continue
		}

		h.cacheProvisioning.mu.Lock()
		if sdk.IsInArray(machine.Name, h.cacheProvisioning.pending) {
			h.cacheProvisioning.mu.Unlock()
			continue
		}
		h.cacheProvisioning.mu.Unlock()

		vm, err := h.vSphereClient.LoadVirtualMachine(ctx, machine.Name)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to load vm %q", machine.Name)
		}

		powerstate, err := h.vSphereClient.GetVirtualMachinePowerState(ctx, vm)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get vm %q powerstate", machine.Name)
		}

		// Provisionned machines are powered off
		if annot.Provisioning &&
			machine.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOn &&
			powerstate != types.VirtualMachinePowerStatePoweredOn &&
			expectedModelPath == annot.WorkerModelPath {
			return vm, nil
		}
	}

	log.Debug(ctx, "unable to find  provisionned VM for model %q", expectedModelPath)
	return nil, nil
}
