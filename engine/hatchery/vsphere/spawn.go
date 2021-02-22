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

	if _, err := h.getVirtualMachineTemplateByName(ctx, spawnArgs.Model.Name); err != nil || spawnArgs.Model.NeedRegistration {
		// Generate worker model vm
		log.Info(ctx, "creating virtual machine model %q", spawnArgs.Model.Name)
		vm, err = h.createVirtualMachineTemplate(ctx, *spawnArgs.Model, spawnArgs.WorkerName)
		if err != nil {
			log.Error(ctx, "Unable to create VM Model: %v", err)
			return err
		}
	}

	if vm == nil {
		var err error
		log.Info(ctx, "creating virtual machine %q", spawnArgs.Model.Name)
		if vm, err = h.vSphereClient.LoadVirtualMachine(ctx, spawnArgs.Model.Name); err != nil {
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

	cloneSpec, err := h.prepareCloneSpec(ctx, vm, annot, spawnArgs.WorkerName)
	if err != nil {
		return err
	}

	folder, err := h.vSphereClient.LoadFolder(ctx)
	if err != nil {
		return err
	}

	log.Info(ctx, "Create vm to execute worker %q, cloneSpec: %+v", spawnArgs.WorkerName, *cloneSpec)
	defer log.Info(ctx, "Terminate to create vm for worker %s", spawnArgs.WorkerName)

	cloneRef, err := h.vSphereClient.CloneVirtualMachine(ctx, vm, folder, spawnArgs.WorkerName, cloneSpec)
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

	if _, err := h.launchClientOp(ctx, clonedVM, model.ModelVirtualMachine, model.ModelVirtualMachine.PostCmd, nil); err != nil {
		log.Warn(ctx, "createVMModel> cannot start program %s", err)
		h.markToDelete(ctx, vm)
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

	if _, err := h.launchClientOp(ctx, vm, model.ModelVirtualMachine, buffer.String(), env); err != nil {
		log.Warn(ctx, "launchScript> cannot start program %s", err)
		h.markToDelete(ctx, vm)
		return err
	}

	return nil
}

func (h *HatcheryVSphere) markToDelete(ctx context.Context, vm *object.VirtualMachine) {
	annot := annotation{ToDelete: true}
	if annotStr, err := json.Marshal(annot); err == nil {
		if err := h.vSphereClient.ReconfigureVirtualMachine(ctx, vm, types.VirtualMachineConfigSpec{
			Annotation: string(annotStr),
		}); err != nil {
			log.Error(ctx, "unable to mark %q as delete", vm.String())
		}
	}
}
