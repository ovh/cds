package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/viper"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

type annotation struct {
	HatcheryName            string    `json:"hatchery_name"`
	WorkerName              string    `json:"worker_name"`
	RegisterOnly            bool      `json:"register_only"`
	WorkerModelName         string    `json:"worker_model_name"`
	WorkerModelLastModified time.Time `json:"worker_model_last_modified"`
	Model                   bool      `json:"model"`
	ToDelete                bool      `json:"to_delete"`
}

type imageConfiguration struct {
	OS       string `json:"os"`
	UserData string `json:"user_data"` //Commands to execute when create vm model
}

// SpawnWorker creates a new cloud instances
func (h *HatcheryVSphere) SpawnWorker(model *sdk.Model, jobID int64, requirements []sdk.Requirement, registerOnly bool, logInfo string) (string, error) {
	var vm *object.VirtualMachine
	var errV error
	ctx := context.Background()
	name := model.Name + "-" + strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)
	if registerOnly {
		name = "register-" + name
	}

	_, errM := h.getModelByName(model.Name)

	if errM != nil || model.NeedRegistration {
		// Generate worker model vm
		model.NeedRegistration = false
		vm, errV = h.createVMModel(model)
	}

	if vm == nil || errV != nil {
		model.NeedRegistration = errV != nil // if we haven't registered
		if vm, errV = h.finder.VirtualMachine(ctx, model.Name); errV != nil {
			return "", sdk.WrapError(errV, "SpawnWorker> Cannot find virtual machine with this model")
		}
	}

	annot := annotation{
		HatcheryName:            h.Hatchery().Name,
		WorkerName:              name,
		RegisterOnly:            registerOnly,
		WorkerModelLastModified: model.UserLastModified,
		WorkerModelName:         model.Name,
	}

	cloneSpec, folder, errCfg := h.createVMConfig(vm, annot)
	if errCfg != nil {
		return "", sdk.WrapError(errCfg, "SpawnWorker> cannot create VM configuration")
	}

	log.Info("Create vm to exec worker")
	defer log.Info("Terminate to create vm for worker")
	task, errC := vm.Clone(ctx, folder, name, *cloneSpec)
	if errC != nil {
		log.Warning("SpawnWorker> cannot clone VM %s", errC)
		return "", errC
	}

	info, errW := task.WaitForResult(ctx, nil)
	if errW != nil || info.State == types.TaskInfoStateError {
		log.Warning("SpawnWorker> state in error %s", errW)
		return "", errW
	}

	return "", h.launchScript(name, jobID, model, registerOnly, info.Result.(types.ManagedObjectReference))
}

func (h *HatcheryVSphere) createVMModel(model *sdk.Model) (*object.VirtualMachine, error) {
	log.Info("Create vm model %s", model.Name)
	ctx := context.Background()
	imgCfg := imageConfiguration{}

	if err := json.Unmarshal([]byte(model.Image), &imgCfg); err != nil {
		return nil, err
	}

	vm, errV := h.finder.VirtualMachine(ctx, imgCfg.OS)
	if errV != nil {
		log.Warning("createVMModel> Cannot find virtual machine")
		return vm, errV
	}

	annot := annotation{
		HatcheryName:            h.Hatchery().Name,
		WorkerModelLastModified: model.UserLastModified,
		WorkerModelName:         model.Name,
		Model:                   true,
	}

	cloneSpec, folder, errCfg := h.createVMConfig(vm, annot)
	if errCfg != nil {
		return vm, sdk.WrapError(errCfg, "createVMModel> cannot create VM configuration")
	}

	task, errC := vm.Clone(ctx, folder, model.Name+"-tmp", *cloneSpec)
	if errC != nil {
		log.Warning("createVMModel> cannot clone VM %s", errC)
		return vm, errC
	}

	info, errW := task.WaitForResult(ctx, nil)
	if errW != nil || info.State == types.TaskInfoStateError {
		log.Warning("createVMModel> state in error %s", errW)
		return vm, errW
	}

	vm = object.NewVirtualMachine(h.vclient.Client, info.Result.(types.ManagedObjectReference))

	if _, errW := vm.WaitForIP(ctx); errW != nil {
		log.Warning("createVMModel> cannot get an ip %s", errW)
		return vm, errW
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

	task, errR := vm.Rename(ctx, model.Name)
	if errR != nil {
		return vm, sdk.WrapError(errR, "createVMModel> Cannot rename model %s", model.Name)
	}

	if _, err := task.WaitForResult(ctx, nil); err != nil {
		return vm, sdk.WrapError(err, "createVMModel> error on waiting result for vm renaming %s", model.Name)
	}

	// FIXME doesn't work, give me a forbidden
	return vm, h.client.WorkerModelUpdate(model.ID, model.Name, model.Type, model.Image, cdsclient.WorkerModelOpts.WithoutRegistrationNeed())
}

func (h *HatcheryVSphere) launchScript(name string, jobID int64, model *sdk.Model, registerOnly bool, vmInfo types.ManagedObjectReference) error {
	ctx := context.TODO()
	// Retrieve the new VM
	vm := object.NewVirtualMachine(h.vclient.Client, vmInfo)

	if _, errW := vm.WaitForIP(ctx); errW != nil {
		return errW
	}

	env := []string{
		"CDS_SINGLE_USE=1",
		"CDS_FORCE_EXIT=1",
		"CDS_API=" + viper.GetString("api"),
		"CDS_TOKEN=" + viper.GetString("token"),
		"CDS_NAME=" + name,
		"CDS_MODEL=" + fmt.Sprintf("%d", model.ID),
		"CDS_HATCHERY=" + fmt.Sprintf("%d", h.Hatchery().ID),
		"CDS_HATCHERY_NAME=" + h.Hatchery().Name,
		"CDS_BOOKED_JOB_ID=" + fmt.Sprintf("%d", jobID),
		"CDS_TTL=" + fmt.Sprintf("%d", h.workerTTL),
	}

	env = append(env, getGraylogGrpcEnv(model)...)

	script := fmt.Sprintf(
		`cd $HOME; rm -f worker; curl "%s/download/worker/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C - >> /tmp/user_data 2>&1; chmod +x worker; ./worker`,
		viper.GetString("api"),
	)

	if registerOnly {
		script += " register"
	}
	script += " ; shutdown -h now;"

	if _, errS := h.launchClientOp(vm, script, env); errS != nil {

		// ----------------------------------------------
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
