package vsphere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

type annotation struct {
	HatcheryName            string    `json:"hatchery_name"`
	WorkerName              string    `json:"worker_name"`
	RegisterOnly            bool      `json:"register_only"`
	WorkerModelPath         string    `json:"worker_model_path"`
	WorkerModelLastModified string    `json:"worker_model_last_modified"`
	Model                   bool      `json:"model"`
	ToDelete                bool      `json:"to_delete"`
	Created                 time.Time `json:"created"`
}

// SpawnWorker creates a new vm instance
func (h *HatcheryVSphere) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) (err error) {
	defer func() {
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "HatcheryVSphere> SpawnWorker %q from model %q: ERROR: %v", spawnArgs.WorkerName, spawnArgs.ModelName(), err)
		} else {
			log.Info(ctx, "HatcheryVSphere> SpawnWorker %q from model %q: DONE", spawnArgs.WorkerName, spawnArgs.ModelName())
		}
	}()

	if spawnArgs.JobID == 0 && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

	var vm *object.VirtualMachine

	if _, err := h.getModelByName(ctx, spawnArgs.Model.Name); err != nil || spawnArgs.Model.NeedRegistration {
		// Generate worker model vm
		log.Info(ctx, "creating virtual machine model %q", spawnArgs.Model.Name)
		vm, err = h.createVMModel(ctx, *spawnArgs.Model, spawnArgs.WorkerName)
		if err != nil {
			log.Error(ctx, "Unable to create VM Model: %v", err)
			return err
		}
	}

	if vm == nil {
		var err error
		log.Info(ctx, "creating virtual machine %q", spawnArgs.Model.Name)
		if vm, err = h.finder.VirtualMachine(ctx, spawnArgs.Model.Name); err != nil {
			return sdk.WrapError(err, "cannot find virtual machine with this model")
		}
	}

	annot := annotation{
		HatcheryName:            h.Name(),
		WorkerName:              spawnArgs.WorkerName,
		RegisterOnly:            spawnArgs.RegisterOnly,
		WorkerModelLastModified: fmt.Sprintf("%d", spawnArgs.Model.UserLastModified.Unix()),
		WorkerModelPath:         spawnArgs.ModelName(),
		Created:                 time.Now(),
	}

	cloneSpec, folder, err := h.createVMConfig(ctx, vm, annot, spawnArgs.WorkerName)
	if err != nil {
		return sdk.WrapError(err, "cannot create VM configuration")
	}

	log.Info(ctx, "Create vm to execute worker %q, cloneSpec: %+v", spawnArgs.WorkerName, *cloneSpec)
	defer log.Info(ctx, "Terminate to create vm for worker %s", spawnArgs.WorkerName)

	task, errC := vm.Clone(ctx, folder, spawnArgs.WorkerName, *cloneSpec)
	if errC != nil {
		return sdk.WrapError(errC, "cannot clone VM")
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil || info.State == types.TaskInfoStateError {
		return sdk.WrapError(err, "state in error")
	}

	// Wait for IP
	vmWorker := object.NewVirtualMachine(h.vclient.Client, info.Result.(types.ManagedObjectReference))
	ip, err := vmWorker.WaitForIP(ctx, true)
	if err != nil {
		return sdk.WrapError(err, "SpawnWorker> cannot get an ip")
	}
	log.Debug(ctx, "SpawnWorker>  New IP: %s", ip)

	return h.launchScriptWorker(ctx, spawnArgs.WorkerName, spawnArgs.JobID, spawnArgs.WorkerToken, *spawnArgs.Model, spawnArgs.RegisterOnly, vmWorker)
}

// createVMModel create a model for a specific worker model
func (h *HatcheryVSphere) createVMModel(ctx context.Context, model sdk.Model, workerName string) (vm *object.VirtualMachine, err error) {
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

	vm, err = h.finder.VirtualMachine(ctx, model.ModelVirtualMachine.Image)
	if err != nil {
		return vm, sdk.WrapError(err, "createVMModel> Cannot find virtual machine")
	}

	log.Debug(ctx, "found virtual machine %q: %+v", model.ModelVirtualMachine.Image, vm)

	annot := annotation{
		HatcheryName:            h.Name(),
		WorkerModelLastModified: fmt.Sprintf("%d", model.UserLastModified.Unix()),
		WorkerModelPath:         model.Path(),
		Model:                   true,
		Created:                 time.Now(),
	}

	cloneSpec, folder, err := h.createVMConfig(ctx, vm, annot, workerName)
	if err != nil {
		return vm, sdk.WrapError(err, "createVMModel> cannot create VM configuration")
	}

	name := model.Name + "-tmp"
	log.Info(ctx, "creating worker %q by cloning vm to %q ", workerName, name)

	task, err := vm.Clone(ctx, folder, name, *cloneSpec)
	if err != nil {
		return vm, sdk.WrapError(err, "createVMModel> cannot clone VM")
	}

	log.Debug(ctx, "waiting for result...")

	info, err := task.WaitForResult(ctx, nil)
	if err != nil || info.State == types.TaskInfoStateError {
		return vm, sdk.WrapError(err, "createVMModel> state in error")
	}

	log.Debug(ctx, "new virtual machine...")
	vm = object.NewVirtualMachine(h.vclient.Client, info.Result.(types.ManagedObjectReference))
	log.Debug(ctx, "waiting for IP...")

	ip, err := vm.WaitForIP(ctx, true)
	if err != nil {
		return vm, sdk.WrapError(err, "createVMModel> cannot get an ip")
	}
	log.Info(ctx, "virtual machine %q has IP %q", name, ip)

	if _, err := h.launchClientOp(ctx, vm, model.ModelVirtualMachine, model.ModelVirtualMachine.PostCmd, nil); err != nil {
		log.Warn(ctx, "createVMModel> cannot start program %s", err)
		annot := annotation{ToDelete: true}
		if annotStr, err := json.Marshal(annot); err == nil {
			vm.Reconfigure(ctx, types.VirtualMachineConfigSpec{
				Annotation: string(annotStr),
			})
		}
	}

	ctxTo, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	log.Debug(ctx, "waiting virtual machine %q to be powered off...", name)
	if err := vm.WaitForPowerState(ctxTo, types.VirtualMachinePowerStatePoweredOff); err != nil {
		return nil, sdk.WrapError(err, "cannot wait for power state result")
	}

	modelFound, err := h.getModelByName(ctx, model.Name)
	if err == nil {
		if err := h.deleteServer(modelFound); err != nil {
			log.Warn(ctx, "createVMModel> Cannot delete previous model %s : %s", model.Name, err)
		}
	}

	ctxTo, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	log.Debug(ctx, "renaming virtual machine %q to %q...", name, model.Name)

	task, errR := vm.Rename(ctxTo, model.Name)
	if errR != nil {
		return vm, sdk.WrapError(errR, "createVMModel> Cannot rename model %s", model.Name)
	}

	ctxTo, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if _, err := task.WaitForResult(ctxTo, nil); err != nil {
		return vm, sdk.WrapError(err, "error on waiting result for vm renaming %s", model.Name)
	}

	log.Debug(ctx, "renaming virtual machine %q to %q DONE", name, model.Name)

	log.Info(ctx, "mark virtual machine %q as template", name, model.Name)
	if err := vm.MarkAsTemplate(ctx); err != nil {
		return vm, sdk.WrapError(err, "unable to mark vm as template")
	}

	return vm, nil
}

// launchScriptWorker launch a script on the worker
func (h *HatcheryVSphere) launchScriptWorker(ctx context.Context, name string, jobID int64, token string, model sdk.Model, registerOnly bool, vm *object.VirtualMachine) error {
	ctxTo, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if _, errW := vm.WaitForIP(ctxTo); errW != nil {
		return sdk.WrapError(errW, "createVMModel> error on waiting ip")
	}

	env := []string{
		"CDS_FROM_WORKER_IMAGE=true",
	}

	env = append(env, h.getGraylogGrpcEnv(model)...)

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
		API:               h.Configuration().API.HTTP.URL,
		Name:              name,
		Token:             token,
		Model:             model.Group.Name + "/" + model.Name,
		HatcheryName:      h.Name(),
		TTL:               h.Config.WorkerTTL,
		FromWorkerImage:   true,
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
		WorkflowJobID:     jobID,
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, udataParam); err != nil {
		return err
	}

	if _, errS := h.launchClientOp(ctx, vm, model.ModelVirtualMachine, buffer.String(), env); errS != nil {
		log.Warn(ctx, "launchScript> cannot start program %s", errS)

		// tag vm to delete
		annot := annotation{ToDelete: true}
		if annotStr, err := json.Marshal(annot); err == nil {
			vm.Reconfigure(ctx, types.VirtualMachineConfigSpec{
				Annotation: string(annotStr),
			})
		}

		return errS
	}

	return nil
}
