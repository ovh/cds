package vsphere

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	cdslog "github.com/ovh/cds/sdk/log"
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
	JobID                   string    `json:"job_id,omitempty"`
	IPAddress               string    `json:"ip_address,omitempty"`
}

// SpawnWorker creates a new vm instance
func (h *HatcheryVSphere) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) (err error) {
	ctx = context.WithValue(ctx, cdslog.AuthWorkerName, spawnArgs.WorkerName)
	defer func() {
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "SpawnWorker %q from model %q: ERROR: %v", spawnArgs.WorkerName, spawnArgs.ModelName(), err)

			h.cachePendingJobID.mu.Lock()
			h.cachePendingJobID.list = sdk.DeleteFromArray(h.cachePendingJobID.list, spawnArgs.JobID)
			h.cachePendingJobID.mu.Unlock()
		} else {
			log.Info(ctx, "SpawnWorker %q from model %q: DONE", spawnArgs.WorkerName, spawnArgs.ModelName())
		}
	}()

	if spawnArgs.JobID == "0" && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

	if spawnArgs.JobID != "0" {
		h.cachePendingJobID.mu.Lock()
		h.cachePendingJobID.list = append(h.cachePendingJobID.list, spawnArgs.JobID)
		defer h.cachePendingJobID.mu.Unlock()

		go func() {
			time.Sleep(3 * time.Minute)
			h.cachePendingJobID.mu.Lock()
			h.cachePendingJobID.list = sdk.DeleteFromArray(h.cachePendingJobID.list, spawnArgs.JobID)
			h.cachePendingJobID.mu.Unlock()
		}()
	}

	var vmTemplate *object.VirtualMachine

	if _, err := h.getVirtualMachineTemplateByName(ctx, spawnArgs.Model.GetName()); err != nil || (spawnArgs.Model.ModelV1 != nil && spawnArgs.Model.ModelV1.NeedRegistration) {
		// Generate worker model vm
		log.Info(ctx, "creating virtual machine model %q", spawnArgs.Model.GetName())
		vmTemplate, err = h.createVirtualMachineTemplate(ctx, spawnArgs.Model, spawnArgs.WorkerName)
		if err != nil {
			log.Error(ctx, "Unable to create VM Model: %v", err)
			return err
		}
	}

	if vmTemplate == nil {
		var err error
		log.Info(ctx, "loading virtual machine template %q", spawnArgs.Model.GetName())
		if vmTemplate, err = h.vSphereClient.LoadVirtualMachine(ctx, spawnArgs.Model.GetName()); err != nil {
			return sdk.WrapError(err, "cannot find virtual machine template with this model")
		}
	}

	// Try to find a provisionned worker
	if !spawnArgs.RegisterOnly {
		provisionnedVMWorker, err := h.FindProvisionnedWorker(ctx, spawnArgs.Model)
		if err != nil {
			return err
		}

		if provisionnedVMWorker != nil {
			log.Info(ctx, "starting worker %q with provisionned machine %q", spawnArgs.Model.GetName(), provisionnedVMWorker.Name())

			if err := h.vSphereClient.RenameVirtualMachine(ctx, provisionnedVMWorker, spawnArgs.WorkerName); err != nil {
				return sdk.WrapError(err, "unable to rename VM %q", provisionnedVMWorker.Name())
			}

			// Before restart it, keep it in the cache for a few minutes to avoid the "killAwolServer" to delete it
			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.restarting = append(h.cacheProvisioning.restarting, spawnArgs.WorkerName)
			h.cacheProvisioning.mu.Unlock()

			time.Sleep(2 * time.Second)

			if err := h.vSphereClient.StartVirtualMachine(ctx, provisionnedVMWorker); err != nil {
				h.cacheProvisioning.mu.Lock()
				h.cacheProvisioning.restarting = sdk.DeleteFromArray(h.cacheProvisioning.restarting, spawnArgs.WorkerName)
				h.cacheProvisioning.mu.Unlock()

				_ = h.vSphereClient.ShutdownVirtualMachine(ctx, provisionnedVMWorker)
				h.markToDelete(ctx, provisionnedVMWorker)
				return sdk.WrapError(err, "unable to start VM %q", spawnArgs.WorkerName)
			}

			// wait for the right IP, probably keep track of the IP address in the server annotations
			// to avoid having two provisionned VM with the same IP address
			// so we if to peek a random IP address by considering already provisionned IP addresses
			moProvisionnedVMWorker, err := h.getVirtualMachineByName(ctx, provisionnedVMWorker.Name())
			if err != nil {
				return sdk.WrapError(err, "unable to find VM %q", spawnArgs.WorkerName)
			}
			var annot = getVirtualMachineCDSAnnotation(ctx, *moProvisionnedVMWorker)

			if err := h.vSphereClient.WaitForVirtualMachineIP(ctx, provisionnedVMWorker, &annot.IPAddress); err != nil {
				h.cacheProvisioning.mu.Lock()
				h.cacheProvisioning.restarting = sdk.DeleteFromArray(h.cacheProvisioning.restarting, spawnArgs.WorkerName)
				h.cacheProvisioning.mu.Unlock()

				_ = h.vSphereClient.ShutdownVirtualMachine(ctx, provisionnedVMWorker)
				h.markToDelete(ctx, provisionnedVMWorker)
				return sdk.WrapError(err, "unable to get VM %q IP Address", spawnArgs.WorkerName)
			}

			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.restarting = sdk.DeleteFromArray(h.cacheProvisioning.restarting, spawnArgs.WorkerName)
			h.cacheProvisioning.mu.Unlock()

			return h.launchScriptWorker(ctx, spawnArgs, provisionnedVMWorker)
		}
	}

	annot := annotation{
		HatcheryName:            h.Name(),
		WorkerName:              spawnArgs.WorkerName,
		RegisterOnly:            spawnArgs.RegisterOnly,
		WorkerModelLastModified: spawnArgs.Model.GetLastModified(),
		WorkerModelPath:         spawnArgs.Model.GetFullPath(),
		Created:                 time.Now(),
		JobID:                   spawnArgs.JobID,
	}

	cloneSpec, err := h.prepareCloneSpec(ctx, vmTemplate, &annot, spawnArgs.WorkerName)
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

	return h.launchScriptWorker(ctx, spawnArgs, vmWorker)
}

// createVirtualMachineTemplate create a model for a specific worker model
func (h *HatcheryVSphere) createVirtualMachineTemplate(ctx context.Context, model sdk.WorkerStarterWorkerModel, workerName string) (vm *object.VirtualMachine, err error) {
	// If the vmTemplate already exist, let's remove it:
	if tmpl, err := h.getVirtualMachineTemplateByName(ctx, model.GetName()); err == nil {
		// remove the template
		log.Warn(ctx, "removing vm template %q to create a new one for model %q", tmpl.Name, model.GetPath())
		if err := h.deleteServer(ctx, tmpl); err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to delete vm template %q: %v", tmpl.Name, err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	log.Info(ctx, "Create vm model %q from %q", model.GetName(), model.GetVSphereImage())
	defer func() {
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "Create vm model %q from %q ERROR: %v", model.GetName(), model.GetVSphereImage(), err)
		} else {
			log.Info(ctx, "Create vm model %q from %q DONE", model.GetName(), model.GetVSphereImage())
		}
	}()

	vm, err = h.vSphereClient.LoadVirtualMachine(ctx, model.GetVSphereImage())
	if err != nil {
		return vm, sdk.WrapError(err, "unable to find virtual machine %q", model.GetVSphereImage())
	}

	log.Debug(ctx, "found virtual machine image %q: %+v", model.GetVSphereImage(), vm)

	annot := annotation{
		HatcheryName:            h.Name(),
		WorkerModelLastModified: model.GetLastModified(),
		WorkerModelPath:         model.GetFullPath(),
		Model:                   true,
		Created:                 time.Now(),
	}

	cloneSpec, err := h.prepareCloneSpec(ctx, vm, &annot, workerName)
	if err != nil {
		return nil, sdk.WrapError(err, "createVMModel> cannot create VM configuration")
	}

	name := model.GetName() + "-tmp"
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
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", model.GetName(), err)
		}
		h.markToDelete(ctx, clonedVM)
		return nil, err
	}

	if err := h.launchClientOp(ctx, clonedVM, model, model.GetPostCmd(), nil); err != nil {
		log.Error(ctx, "cannot start program on virtual machine %q: %v", clonedVM.Name(), err)
		log.Warn(ctx, "shutdown virtual machine %q", clonedVM.Name())
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, clonedVM); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", model.GetName(), err)
		}
		h.markToDelete(ctx, clonedVM)
		return nil, err
	}

	if err := h.vSphereClient.WaitForVirtualMachineShutdown(ctx, clonedVM); err != nil {
		return nil, err
	}

	modelFound, err := h.getVirtualMachineTemplateByName(ctx, model.GetName())
	if err == nil {
		if err := h.deleteServer(ctx, modelFound); err != nil {
			log.Warn(ctx, "createVMModel> Cannot delete previous model %s : %s", model.GetName(), err)
		}
	}

	if err := h.vSphereClient.RenameVirtualMachine(ctx, clonedVM, model.GetName()); err != nil {
		return nil, err
	}
	log.Debug(ctx, "renaming virtual machine %q to %q: DONE", clonedVM.String(), model.GetName())

	log.Info(ctx, "mark virtual machine %q as template %q", name, model.GetName())
	if err := h.vSphereClient.MarkVirtualMachineAsTemplate(ctx, clonedVM); err != nil {
		return nil, err
	}

	return vm, nil
}

func (h *HatcheryVSphere) checkVirtualMachineIsReady(ctx context.Context, model sdk.WorkerStarterWorkerModel, vm *object.VirtualMachine) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var latestError error
	for {
		if ctx.Err() != nil {
			return sdk.WithStack(fmt.Errorf("vm %q is not ready: %v - %v", vm.Name(), latestError, ctx.Err()))
		}
		if err := h.launchClientOp(ctx, vm, model, "env", nil); err != nil {
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
func (h *HatcheryVSphere) launchScriptWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments, vm *object.VirtualMachine) error {
	if err := h.vSphereClient.WaitForVirtualMachineIP(ctx, vm, nil); err != nil {
		return err
	}

	workerConfig := h.GenerateWorkerConfig(ctx, h, spawnArgs)

	udata := spawnArgs.Model.GetPreCmd() + "\n" + spawnArgs.Model.GetCmd()

	// Redirect worker stdout and stderr in /tmp
	if spawnArgs.RegisterOnly {
		udata += " register 1>/root/worker.register.log 2>&1"
	} else {
		udata += " 1>/root/worker.log 2>&1;"
	}
	udata += "\n" + spawnArgs.Model.GetPostCmd()

	tmpl, err := template.New("udata").Parse(udata)
	if err != nil {
		return sdk.NewErrorFrom(err, "unable to parse template: %v", err)
	}

	udataParam := struct {
		// All fields below are deprecated
		API               string
		Token             string
		Name              string
		BaseDir           string
		HTTPInsecure      bool
		Model             string
		HatcheryName      string
		WorkflowJobID     int64
		TTL               int
		FromWorkerImage   bool
		GraylogHost       string
		GraylogPort       int
		GraylogExtraKey   string
		GraylogExtraValue string
		WorkerBinary      string
		InjectEnvVars     map[string]string
		// All fields above are deprecated
		Config string
	}{
		API:             workerConfig.APIEndpoint,
		FromWorkerImage: true,
		Config:          workerConfig.EncodeBase64(),
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, udataParam); err != nil {
		return sdk.NewErrorFrom(err, "unable to execute template: %v", err)
	}

	if err := h.checkVirtualMachineIsReady(ctx, spawnArgs.Model, vm); err != nil {
		log.Error(ctx, "virtual machine %q is not ready: %v", vm.Name(), err)
		log.Warn(ctx, "shutdown virtual machine %q", vm.Name())
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, vm); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", spawnArgs.Model.GetPath(), err)
		}
		h.markToDelete(ctx, vm)
		return err
	}

	env := []string{
		"CDS_CONFIG=" + workerConfig.EncodeBase64(),
	}
	for k, v := range workerConfig.InjectEnvVars {
		env = append(env, k+"="+v)
	}

	if err := h.launchClientOp(ctx, vm, spawnArgs.Model, buffer.String(), env); err != nil {
		log.Warn(ctx, "launchScript> cannot start program %s", err)
		log.Error(ctx, "cannot start program on virtual machine %q: %v", vm.Name(), err)
		log.Warn(ctx, "shutdown virtual machine %q", vm.Name())
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, vm); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", spawnArgs.Model.GetName(), err)
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

	cloneSpec, err := h.prepareCloneSpec(ctx, vmTemplate, &annot, workerName)
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

	if err := h.vSphereClient.WaitForVirtualMachineIP(ctx, clonedVM, &annot.IPAddress); err != nil {
		return err
	}

	// the provisionned workers are shutdown when they are created
	if err := h.vSphereClient.ShutdownVirtualMachine(ctx, clonedVM); err != nil {
		return err
	}

	log.Info(ctx, "vm %q has been provisionned", workerName)

	return nil
}

func (h *HatcheryVSphere) FindProvisionnedWorker(ctx context.Context, m sdk.WorkerStarterWorkerModel) (*object.VirtualMachine, error) {
	expectedModelPath := m.GetFullPath()

	log.Debug(ctx, "searching for provisionned VM for model %q", expectedModelPath)

	machines := h.getVirtualMachines(ctx)
	for _, machine := range machines {
		if !strings.HasPrefix(machine.Name, "provision-") {
			continue
		}

		annot := getVirtualMachineCDSAnnotation(ctx, machine)
		if annot == nil {
			continue
		}

		h.cacheProvisioning.mu.Lock()
		if sdk.IsInArray(machine.Name, h.cacheProvisioning.pending) {
			h.cacheProvisioning.mu.Unlock()
			continue
		}

		if sdk.IsInArray(machine.Name, h.cacheProvisioning.restarting) {
			h.cacheProvisioning.mu.Unlock()
			continue
		}

		h.cacheProvisioning.mu.Unlock()

		h.cacheToDelete.mu.Lock()
		if sdk.IsInArray(machine.Name, h.cacheToDelete.list) {
			h.cacheToDelete.mu.Unlock()
			continue
		}
		h.cacheToDelete.mu.Unlock()

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
