package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

type annotation struct {
	HatcheryName            string    `json:"hatchery_name"`
	WorkerName              string    `json:"worker_name"`
	RegisterOnly            bool      `json:"register_only"`
	WorkerModelName         string    `json:"worker_model_name"`
	WorkerModelLastModified string    `json:"worker_model_last_modified"`
	Model                   bool      `json:"model"`
	ToDelete                bool      `json:"to_delete"`
	Created                 time.Time `json:"created"`
}

type imageConfiguration struct {
	OS       string `json:"os"`
	UserData string `json:"user_data"` //Commands to execute when create vm model
}

// SpawnWorker creates a new vm instance
func (h *HatcheryVSphere) SpawnWorker(spawnArgs hatchery.SpawnArguments) (string, error) {
	var vm *object.VirtualMachine
	var errV error
	ctx := context.Background()
	name := "worker-" + spawnArgs.Model.Name + "-" + strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)
	if spawnArgs.RegisterOnly {
		name = "register-" + name
	}

	_, errM := h.getModelByName(spawnArgs.Model.Name)

	if errM != nil || spawnArgs.Model.NeedRegistration {
		// Generate worker model vm
		vm, errV = h.createVMModel(spawnArgs.Model)
	}

	if vm == nil || errV != nil {
		spawnArgs.Model.NeedRegistration = errV != nil // if we haven't registered
		if vm, errV = h.finder.VirtualMachine(ctx, spawnArgs.Model.Name); errV != nil {
			return "", sdk.WrapError(errV, "SpawnWorker> Cannot find virtual machine with this model")
		}
	}

	annot := annotation{
		HatcheryName:            h.Hatchery().Name,
		WorkerName:              name,
		RegisterOnly:            spawnArgs.RegisterOnly,
		WorkerModelLastModified: fmt.Sprintf("%d", spawnArgs.Model.UserLastModified.Unix()),
		WorkerModelName:         spawnArgs.Model.Name,
		Created:                 time.Now(),
	}

	cloneSpec, folder, errCfg := h.createVMConfig(vm, annot)
	if errCfg != nil {
		return "", sdk.WrapError(errCfg, "SpawnWorker> cannot create VM configuration")
	}

	log.Info("Create vm to exec worker %s", name)
	defer log.Info("Terminate to create vm for worker %s", name)
	task, errC := vm.Clone(ctx, folder, name, *cloneSpec)
	if errC != nil {
		return "", sdk.WrapError(errC, "SpawnWorker> cannot clone VM")
	}

	info, errW := task.WaitForResult(ctx, nil)
	if errW != nil || info.State == types.TaskInfoStateError {
		return "", sdk.WrapError(errW, "SpawnWorker> state in error")
	}

	return "", h.launchScriptWorker(name, spawnArgs.IsWorkflowJob, spawnArgs.JobID, spawnArgs.Model, spawnArgs.RegisterOnly, info.Result.(types.ManagedObjectReference))
}

// createVMModel create a model for a specific worker model
func (h *HatcheryVSphere) createVMModel(model sdk.Model) (*object.VirtualMachine, error) {
	log.Info("Create vm model %s", model.Name)
	ctx := context.Background()
	imgCfg := imageConfiguration{}

	if err := json.Unmarshal([]byte(model.Image), &imgCfg); err != nil {
		return nil, sdk.WrapError(err, "createVMModel> Cannot unmarshal image")
	}

	vm, errV := h.finder.VirtualMachine(ctx, imgCfg.OS)
	if errV != nil {
		return vm, sdk.WrapError(errV, "createVMModel> Cannot find virtual machine")
	}

	annot := annotation{
		HatcheryName:            h.Hatchery().Name,
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

	if _, errS := h.launchClientOp(vm, imgCfg.UserData+"; shutdown -h now", nil); errS != nil {
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
		return nil, sdk.WrapError(err, "createVMModel > cannot wait for power state result")
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
		return vm, sdk.WrapError(err, "createVMModel> error on waiting result for vm renaming %s", model.Name)
	}

	return vm, nil
}

// launchScriptWorker launch a script on the worker
func (h *HatcheryVSphere) launchScriptWorker(name string, isWorkflowJob bool, jobID int64, model sdk.Model, registerOnly bool, vmInfo types.ManagedObjectReference) error {
	ctx := context.Background()
	// Retrieve the new VM
	vm := object.NewVirtualMachine(h.vclient.Client, vmInfo)

	ctxTo, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if _, errW := vm.WaitForIP(ctxTo); errW != nil {
		return sdk.WrapError(errW, "createVMModel> error on waiting ip")
	}

	env := []string{
		"CDS_SINGLE_USE=1",
		"CDS_FORCE_EXIT=1",
		"CDS_FROM_WORKER_IMAGE=true",
		"CDS_API=" + h.Configuration().API.HTTP.URL,
		"CDS_TOKEN=" + h.Configuration().API.Token,
		"CDS_NAME=" + name,
		"CDS_MODEL=" + fmt.Sprintf("%d", model.ID),
		"CDS_HATCHERY=" + fmt.Sprintf("%d", h.Hatchery().ID),
		"CDS_HATCHERY_NAME=" + h.Hatchery().Name,
		"CDS_TTL=" + fmt.Sprintf("%d", h.workerTTL),
	}

	if isWorkflowJob {
		env = append(env, fmt.Sprintf("CDS_BOOKED_WORKFLOW_JOB_ID=%d", jobID))
	} else {
		env = append(env, fmt.Sprintf("CDS_BOOKED_PB_JOB_ID=%d", jobID))
	}

	env = append(env, h.getGraylogGrpcEnv(model)...)

	script := fmt.Sprintf(
		`cd $HOME; rm -f worker; curl "%s/download/worker/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C - >> /tmp/user_data 2>&1; chmod +x worker; PATH=$PATH ./worker`,
		h.Configuration().API.HTTP.URL,
	)

	if registerOnly {
		script += " register"
	}
	script += " ; shutdown -h now;"

	if _, errS := h.launchClientOp(vm, script, env); errS != nil {
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
