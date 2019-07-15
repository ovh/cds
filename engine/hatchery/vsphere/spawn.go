package vsphere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
)

type annotation struct {
	HatcheryName            string    `json:"hatchery_name"`
	WorkerName              string    `json:"worker_name"`
	RegisterOnly            bool      `json:"register_only"`
	WorkerModelName         string    `json:"worker_model_name"`
	WorkerModelPath         string    `json:"worker_model_path"`
	WorkerModelLastModified string    `json:"worker_model_last_modified"`
	Model                   bool      `json:"model"`
	ToDelete                bool      `json:"to_delete"`
	Created                 time.Time `json:"created"`
}

// SpawnWorker creates a new vm instance
func (h *HatcheryVSphere) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	var vm *object.VirtualMachine
	var errV error
	name := "worker-" + spawnArgs.Model.Name + "-" + strings.Replace(namesgenerator.GetRandomNameCDS(0), "_", "-", -1)
	if spawnArgs.RegisterOnly {
		name = "register-" + name
	}

	_, errM := h.getModelByName(spawnArgs.Model.Name)

	if errM != nil || spawnArgs.Model.NeedRegistration {
		// Generate worker model vm
		vm, errV = h.createVMModel(*spawnArgs.Model)
	}

	if vm == nil || errV != nil {
		spawnArgs.Model.NeedRegistration = errV != nil // if we haven't registered
		if vm, errV = h.finder.VirtualMachine(ctx, spawnArgs.Model.Name); errV != nil {
			return sdk.WrapError(errV, "cannot find virtual machine with this model")
		}
	}

	annot := annotation{
		HatcheryName:            h.Name,
		WorkerName:              name,
		RegisterOnly:            spawnArgs.RegisterOnly,
		WorkerModelLastModified: fmt.Sprintf("%d", spawnArgs.Model.UserLastModified.Unix()),
		WorkerModelName:         spawnArgs.ModelName(),
		Created:                 time.Now(),
	}

	cloneSpec, folder, errCfg := h.createVMConfig(vm, annot)
	if errCfg != nil {
		return sdk.WrapError(errCfg, "cannot create VM configuration")
	}

	log.Info("Create vm to exec worker %s", name)
	defer log.Info("Terminate to create vm for worker %s", name)
	task, errC := vm.Clone(ctx, folder, name, *cloneSpec)
	if errC != nil {
		return sdk.WrapError(errC, "cannot clone VM")
	}

	info, errW := task.WaitForResult(ctx, nil)
	if errW != nil || info.State == types.TaskInfoStateError {
		return sdk.WrapError(errW, "state in error")
	}

	return h.launchScriptWorker(name, spawnArgs.JobID, *spawnArgs.Model, spawnArgs.RegisterOnly, info.Result.(types.ManagedObjectReference))
}

// createVMModel create a model for a specific worker model
func (h *HatcheryVSphere) createVMModel(model sdk.Model) (*object.VirtualMachine, error) {
	log.Info("Create vm model %s", model.Name)
	ctx := context.Background()

	vm, errV := h.finder.VirtualMachine(ctx, model.ModelVirtualMachine.Image)
	if errV != nil {
		return vm, sdk.WrapError(errV, "createVMModel> Cannot find virtual machine")
	}

	annot := annotation{
		HatcheryName:            h.Name,
		WorkerModelLastModified: fmt.Sprintf("%d", model.UserLastModified.Unix()),
		WorkerModelName:         model.Name,
		Model:                   true,
		Created:                 time.Now(),
	}

	cloneSpec, folder, errCfg := h.createVMConfig(vm, annot)
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

	if _, errW := vm.WaitForIP(ctx); errW != nil {
		return vm, sdk.WrapError(errW, "createVMModel> cannot get an ip")
	}

	if _, errS := h.launchClientOp(vm, model.ModelVirtualMachine.PreCmd+"; \n"+model.ModelVirtualMachine.Cmd+"; \n"+model.ModelVirtualMachine.PostCmd, nil); errS != nil {
		log.Warning("createVMModel> cannot start program %s", errS)
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
	log.Info("createVMModel> model %s is build", model.Name)

	modelFound, errM := h.getModelByName(model.Name)
	if errM == nil {
		if errD := h.deleteServer(modelFound); errD != nil {
			log.Warning("createVMModel> Cannot delete previous model %s : %s", model.Name, errD)
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

	return vm, nil
}

// launchScriptWorker launch a script on the worker
func (h *HatcheryVSphere) launchScriptWorker(name string, jobID int64, model sdk.Model, registerOnly bool, vmInfo types.ManagedObjectReference) error {
	ctx := context.Background()
	// Retrieve the new VM
	vm := object.NewVirtualMachine(h.vclient.Client, vmInfo)

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
	udata += ("\n" + model.ModelVirtualMachine.PostCmd)

	tmpl, errt := template.New("udata").Parse(udata)
	if errt != nil {
		return errt
	}
	udataParam := sdk.WorkerArgs{
		API:               h.Configuration().API.HTTP.URL,
		Name:              name,
		Token:             h.Configuration().API.Token,
		Model:             model.Group.Name + "/" + model.Name,
		HatcheryName:      h.Name,
		TTL:               h.Config.WorkerTTL,
		FromWorkerImage:   true,
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
	}

	udataParam.WorkflowJobID = jobID

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, udataParam); err != nil {
		return err
	}

	if _, errS := h.launchClientOp(vm, buffer.String(), env); errS != nil {
		log.Warning("launchScript> cannot start program %s", errS)

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
