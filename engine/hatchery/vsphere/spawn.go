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
func (h *HatcheryVSphere) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	if spawnArgs.JobID == 0 && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

	var vm *object.VirtualMachine
	_, err := h.getModelByName(ctx, spawnArgs.Model.Name)

	if err != nil || spawnArgs.Model.NeedRegistration {
		// Generate worker model vm
		vm, err = h.createVMModel(*spawnArgs.Model, spawnArgs.WorkerName)
		if err != nil {
			log.Error(ctx, "Unable to create VM Model: %v", err)
			return err
		}
	}

	if vm == nil {
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

	cloneSpec, folder, err := h.createVMConfig(vm, annot, spawnArgs.WorkerName)
	if err != nil {
		return sdk.WrapError(err, "cannot create VM configuration")
	}

	log.Info(ctx, "Create vm to exec worker %s", spawnArgs.WorkerName)
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

	return h.launchScriptWorker(spawnArgs.WorkerName, spawnArgs.JobID, spawnArgs.WorkerToken, *spawnArgs.Model, spawnArgs.RegisterOnly, vmWorker)
}

// createVMModel create a model for a specific worker model
func (h *HatcheryVSphere) createVMModel(model sdk.Model, workerName string) (*object.VirtualMachine, error) {
	ctx := context.Background()
	log.Info(ctx, "Create vm model %s from %s", model.Name, model.ModelVirtualMachine.Image)
	defer func() {
		log.Info(ctx, "Create vm model %s from %s DONE", model.Name, model.ModelVirtualMachine.Image)
	}()

	vm, errV := h.finder.VirtualMachine(ctx, model.ModelVirtualMachine.Image)
	if errV != nil {
		return vm, sdk.WrapError(errV, "createVMModel> Cannot find virtual machine")
	}

	annot := annotation{
		HatcheryName:            h.Name(),
		WorkerModelLastModified: fmt.Sprintf("%d", model.UserLastModified.Unix()),
		WorkerModelPath:         model.Path(),
		Model:                   true,
		Created:                 time.Now(),
	}

	cloneSpec, folder, errCfg := h.createVMConfig(vm, annot, workerName)
	if errCfg != nil {
		return vm, sdk.WrapError(errCfg, "createVMModel> cannot create VM configuration")
	}

	task, errC := vm.Clone(ctx, folder, model.Name+"-tmp", *cloneSpec)
	if errC != nil {
		return vm, sdk.WrapError(errC, "createVMModel> cannot clone VM")
	}

	info, errWr := task.WaitForResult(ctx, nil)
	if errWr != nil || info.State == types.TaskInfoStateError {
		return vm, sdk.WrapError(errWr, "createVMModel> state in error")
	}

	vm = object.NewVirtualMachine(h.vclient.Client, info.Result.(types.ManagedObjectReference))

	_, errW := vm.WaitForIP(ctx, true)
	if errW != nil {
		return vm, sdk.WrapError(errW, "createVMModel> cannot get an ip")
	}

	if _, errS := h.launchClientOp(vm, model.ModelVirtualMachine, model.ModelVirtualMachine.PostCmd, nil); errS != nil {
		log.Warn(ctx, "createVMModel> cannot start program %s", errS)
		annot := annotation{ToDelete: true}
		if annotStr, err := json.Marshal(annot); err == nil {
			vm.Reconfigure(ctx, types.VirtualMachineConfigSpec{
				Annotation: string(annotStr),
			})
		}
	}

	ctxTo, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
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
	task, errR := vm.Rename(ctxTo, model.Name)
	if errR != nil {
		return vm, sdk.WrapError(errR, "createVMModel> Cannot rename model %s", model.Name)
	}

	ctxTo, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if _, err := task.WaitForResult(ctxTo, nil); err != nil {
		return vm, sdk.WrapError(err, "error on waiting result for vm renaming %s", model.Name)
	}

	if err := vm.MarkAsTemplate(ctx); err != nil {
		return vm, sdk.WrapError(err, "unable to mark vm as template")
	}

	return vm, nil
}

// launchScriptWorker launch a script on the worker
func (h *HatcheryVSphere) launchScriptWorker(name string, jobID int64, token string, model sdk.Model, registerOnly bool, vm *object.VirtualMachine) error {
	ctx := context.Background()

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

	if _, errS := h.launchClientOp(vm, model.ModelVirtualMachine, buffer.String(), env); errS != nil {
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
