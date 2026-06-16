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
	HatcheryName    string `json:"hatchery_name,omitempty"`
	WorkerName      string `json:"worker_name,omitempty"`
	Provisioning    bool   `json:"provisioning,omitempty"`
	WorkerModelPath string `json:"worker_model_path,omitempty"`
	VMwareModelPath string `json:"vmware_model_path,omitempty"`
	// Model is true for VM template used by provision / new worker without provision
	// we don't want to destroy (with killawolServer for exemple) a vm with model = true
	Model     bool      `json:"model,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	JobID     string    `json:"job_id,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
}

// SpawnWorker creates a new vm instance
func (h *HatcheryVSphere) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) (err error) {
	ctx = context.WithValue(ctx, cdslog.AuthWorkerName, spawnArgs.WorkerName)

	defer func() {
		h.cachePendingJobID.mu.Lock()
		h.cachePendingJobID.list = sdk.DeleteFromArray(h.cachePendingJobID.list, spawnArgs.JobID)
		h.cachePendingJobID.mu.Unlock()
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "SpawnWorker %q from model %q: ERROR: %v", spawnArgs.WorkerName, spawnArgs.ModelName(), err)
		} else {
			log.Info(ctx, "SpawnWorker %q from model %q: DONE", spawnArgs.WorkerName, spawnArgs.ModelName())
		}
	}()

	if spawnArgs.JobID == "0" {
		return sdk.WithStack(fmt.Errorf("no job ID"))
	}

	h.cachePendingJobID.mu.Lock()
	h.cachePendingJobID.list = append(h.cachePendingJobID.list, spawnArgs.JobID)
	h.cachePendingJobID.mu.Unlock()

	if spawnArgs.Model.ModelV2 == nil {
		return sdk.WithStack(fmt.Errorf("worker model v1 is no longer supported on vSphere"))
	}

	// Amendment C: Resolve flavor before spawning
	var flavor *VSphereFlavorConfig
	flavorName := spawnArgs.Model.GetFlavor(spawnArgs.Requirements, h.Config.DefaultFlavor)
	if flavorName != "" {
		flavor = h.getFlavor(flavorName)
		if flavor == nil {
			return sdk.WithStack(fmt.Errorf("flavor %q not found in hatchery configuration", flavorName))
		}
		log.Info(ctx, "SpawnWorker> using flavor %q (%d vCPUs, %d MB RAM)", flavorName, flavor.CPUs, flavor.MemoryMB)
	}

	// Send spawn info for V2 jobs
	if sdk.IsValidUUID(spawnArgs.JobID) {
		var flavorInfo sdk.V2SendJobRunInfo
		flavorInfo.Level = sdk.WorkflowRunInfoLevelInfo
		flavorInfo.Time = time.Now()

		if flavor != nil {
			if flavor.DiskSizeGB > 0 {
				flavorInfo.Message = fmt.Sprintf("Worker %q will use flavor %q (%d vCPUs, %d MB RAM, %d GB disk)", spawnArgs.WorkerName, flavorName, flavor.CPUs, flavor.MemoryMB, flavor.DiskSizeGB)
			} else {
				flavorInfo.Message = fmt.Sprintf("Worker %q will use flavor %q (%d vCPUs, %d MB RAM)", spawnArgs.WorkerName, flavorName, flavor.CPUs, flavor.MemoryMB)
			}
		} else {
			flavorInfo.Message = fmt.Sprintf("Worker %q will use template resources (no flavor)", spawnArgs.WorkerName)
		}

		if err := h.CDSClientV2().V2QueuePushJobInfo(ctx, spawnArgs.Region, spawnArgs.JobID, flavorInfo); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	}

	// Workers are only started from pre-provisioned VMs: claim one. There is
	// no fallback cloning the template directly, CanSpawn only accepts a job
	// when a provisioned VM is available for the model.
	provisionnedVMWorker, err := h.FindProvisionnedWorker(ctx, spawnArgs.Model)
	if err != nil {
		return err
	}
	if provisionnedVMWorker == nil {
		return sdk.WithStack(fmt.Errorf("no provisioned worker available for model %q", spawnArgs.Model.GetName()))
	}

	log.Info(ctx, "starting worker %q with provisionned machine %q", spawnArgs.Model.GetName(), provisionnedVMWorker.Name())

	// Amendment C: Reconfigure VM to flavor before starting (if flavor requested)
	if flavor != nil {
		log.Info(ctx, "reconfiguring provisioned VM %q to flavor %q", provisionnedVMWorker.Name(), flavorName)
		if err := h.reconfigureVM(ctx, provisionnedVMWorker, flavor); err != nil {
			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.using = sdk.DeleteFromArray(h.cacheProvisioning.using, provisionnedVMWorker.Name())
			h.cacheProvisioning.mu.Unlock()
			return sdk.WrapError(err, "unable to reconfigure VM %q to flavor %q", provisionnedVMWorker.Name(), flavorName)
		}
	}

	if err := h.vSphereClient.RenameVirtualMachine(ctx, provisionnedVMWorker, spawnArgs.WorkerName); err != nil {
		h.cacheProvisioning.mu.Lock()
		h.cacheProvisioning.using = sdk.DeleteFromArray(h.cacheProvisioning.using, provisionnedVMWorker.Name())
		h.cacheProvisioning.mu.Unlock()
		return sdk.WrapError(err, "unable to rename VM %q", provisionnedVMWorker.Name())
	}

	time.Sleep(2 * time.Second)

	if err := h.vSphereClient.StartVirtualMachine(ctx, provisionnedVMWorker); err != nil {
		h.cacheProvisioning.mu.Lock()
		h.cacheProvisioning.using = sdk.DeleteFromArray(h.cacheProvisioning.using, provisionnedVMWorker.Name())
		h.cacheProvisioning.mu.Unlock()

		_ = h.vSphereClient.ShutdownVirtualMachine(ctx, provisionnedVMWorker)
		h.markToDelete(ctx, provisionnedVMWorker.Name())
		return sdk.WrapError(err, "unable to start VM %q", spawnArgs.WorkerName)
	}

	// wait for the right IP, probably keep track of the IP address in the server annotations
	// to avoid having two provisionned VM with the same IP address
	// so we if to peek a random IP address by considering already provisionned IP addresses
	moProvisionnedVMWorker, err := h.getVirtualMachineByName(ctx, provisionnedVMWorker.Name())
	if err != nil {
		return sdk.WrapError(err, "unable to find VM %q", spawnArgs.WorkerName)
	}
	annot := getVirtualMachineCDSAnnotation(ctx, *moProvisionnedVMWorker)

	if err := h.vSphereClient.WaitForVirtualMachineIP(ctx, provisionnedVMWorker, &annot.IPAddress, spawnArgs.WorkerName); err != nil {
		h.cacheProvisioning.mu.Lock()
		h.cacheProvisioning.using = sdk.DeleteFromArray(h.cacheProvisioning.using, provisionnedVMWorker.Name())
		h.cacheProvisioning.mu.Unlock()

		_ = h.vSphereClient.ShutdownVirtualMachine(ctx, provisionnedVMWorker)
		h.markToDelete(ctx, provisionnedVMWorker.Name())
		return sdk.WrapError(err, "unable to get VM %q IP Address", spawnArgs.WorkerName)
	}

	errLaunch := h.launchScriptWorker(ctx, spawnArgs, provisionnedVMWorker, spawnArgs.WorkerName)

	h.cacheProvisioning.mu.Lock()
	h.cacheProvisioning.using = sdk.DeleteFromArray(h.cacheProvisioning.using, provisionnedVMWorker.Name())
	h.cacheProvisioning.mu.Unlock()

	return errLaunch
}

func (h *HatcheryVSphere) checkVirtualMachineIsReady(ctx context.Context, model sdk.WorkerStarterWorkerModel, vm *object.VirtualMachine, vmName string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var latestError error
	for {
		if ctx.Err() != nil {
			return sdk.WithStack(fmt.Errorf("vm %q is not ready (ctx err): %v - %v", vmName, latestError, ctx.Err()))
		}
		if err := h.launchClientOp(ctx, vm, model, "env", nil); err != nil {
			log.Warn(ctx, "virtual machine %q is not ready (env cmd): %v", vmName, err)
			latestError = err
			time.Sleep(time.Second)
			continue // If it failing, wait and retry
		}
		break // else it means that it is ready
	}
	return nil
}

// launchScriptWorker launch a script on the worker
func (h *HatcheryVSphere) launchScriptWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments, vm *object.VirtualMachine, vmName string) error {
	if err := h.vSphereClient.WaitForVirtualMachineIP(ctx, vm, nil, vmName); err != nil {
		return err
	}

	workerConfig := h.GenerateWorkerConfig(ctx, h, spawnArgs)

	udata := spawnArgs.Model.GetPreCmd() + "\n" + spawnArgs.Model.GetCmd()

	// Redirect worker stdout and stderr in /tmp
	udata += " 1>/tmp/worker.log 2>&1;"
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

	if err := h.checkVirtualMachineIsReady(ctx, spawnArgs.Model, vm, spawnArgs.WorkerName); err != nil {
		log.Error(ctx, "virtual machine %q is not ready: %v", spawnArgs.WorkerName, err)
		log.Warn(ctx, "shutdown virtual machine %q", spawnArgs.WorkerName)
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, vm); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", spawnArgs.Model.GetPath(), err)
		}
		h.markToDelete(ctx, spawnArgs.WorkerName)
		return err
	}

	// Execute pre-start script if configured for this model
	if modelCfg := h.getModelConfig(spawnArgs.Model); modelCfg != nil && modelCfg.PreStartScript != "" {
		log.Info(ctx, "launchScriptWorker: executing pre-start script on %q", spawnArgs.WorkerName)
		if err := h.launchClientOp(ctx, vm, spawnArgs.Model, modelCfg.PreStartScript, nil); err != nil {
			log.Error(ctx, "launchScriptWorker: pre-start script failed on %q: %v", spawnArgs.WorkerName, err)
			h.markToDelete(ctx, spawnArgs.WorkerName)
			return err
		}
	}

	env := []string{
		"CDS_CONFIG=" + workerConfig.EncodeBase64(),
	}
	for k, v := range workerConfig.InjectEnvVars {
		env = append(env, k+"="+v)
	}

	if err := h.launchClientOp(ctx, vm, spawnArgs.Model, buffer.String(), env); err != nil {
		log.Warn(ctx, "launchScript> cannot start program %s", err)
		log.Error(ctx, "cannot start program on virtual machine %q: %v", spawnArgs.WorkerName, err)
		log.Warn(ctx, "shutdown virtual machine %q", spawnArgs.WorkerName)
		if err := h.vSphereClient.ShutdownVirtualMachine(ctx, vm); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "createVMModel> unable to shutdown vm %q: %v", spawnArgs.Model.GetName(), err)
		}
		h.markToDelete(ctx, spawnArgs.WorkerName)
		return err
	}

	return nil
}

func (h *HatcheryVSphere) markToDelete(ctx context.Context, vmName string) {
	h.cacheToDelete.mu.Lock()
	defer h.cacheToDelete.mu.Unlock()

	log.Debug(ctx, "markToDelete %q", vmName)

	// Reload the vm ref to get the annotation
	allVMRef, err := h.vSphereClient.ListVirtualMachines(ctx)
	if err != nil {
		log.Error(ctx, "unable to get virtual machines: %v", err)
		return
	}

	var vmRef *mo.VirtualMachine
	for i := range allVMRef {
		if allVMRef[i].Name == vmName {
			vmRef = &allVMRef[i]
			break
		}
	}

	if vmRef == nil {
		err := sdk.WithStack(fmt.Errorf("virtual machine ref %q not found", vmName))
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "unable to get virtual machines: %v", err)
		return
	}

	annot := getVirtualMachineCDSAnnotation(ctx, *vmRef)
	if annot == nil {
		return
	}

	h.cacheToDelete.list = append(h.cacheToDelete.list, vmRef.Name)
}

func (h *HatcheryVSphere) ProvisionWorkerV2(ctx context.Context, vmwareModel string, workerName string) error {
	vmTemplate, err := h.vSphereClient.LoadVirtualMachine(ctx, vmwareModel)
	if err != nil {
		return sdk.WrapError(err, "cannot find virtual machine template with VMware model %v", vmwareModel)
	}

	annot := annotation{
		HatcheryName:    h.Name(),
		WorkerName:      workerName,
		Provisioning:    true,
		VMwareModelPath: vmwareModel,
		Created:         time.Now(),
	}

	return h.cloneProvisionedWorker(ctx, vmTemplate, annot, workerName)
}

// cloneProvisionedWorker clones a VM template for provisioning. After the clone
// the VM is powered on but not yet shut down — the caller is responsible for
// completing provisioning via finishProvisioning.
func (h *HatcheryVSphere) cloneProvisionedWorker(ctx context.Context, vmTemplate *object.VirtualMachine, annot annotation, workerName string) error {
	cloneSpec, err := h.prepareCloneSpec(ctx, vmTemplate, &annot)
	if err != nil {
		return err
	}

	folder, err := h.vSphereClient.LoadFolder(ctx)
	if err != nil {
		return err
	}

	log.Info(ctx, "provisoning %q by cloning %q", workerName, vmTemplate.Name())

	cloneRef, err := h.vSphereClient.CloneVirtualMachine(ctx, vmTemplate, folder, workerName, cloneSpec)
	if err != nil {
		return err
	}

	if _, err := h.vSphereClient.NewVirtualMachine(ctx, cloneSpec, cloneRef, workerName); err != nil {
		return err
	}

	return nil
}

// hasAvailableProvisionedWorker reports whether a provisioned VM matching the
// given model is ready to be claimed by SpawnWorker. It mirrors the matching
// logic of FindProvisionnedWorker but is read-only: it does not reserve the VM
// (no mutation of cacheProvisioning.using) and avoids per-VM vSphere API calls,
// so it is cheap enough to be called from CanSpawn.
func (h *HatcheryVSphere) hasAvailableProvisionedWorker(ctx context.Context, model sdk.WorkerStarterWorkerModel) bool {
	expectedModelPath := model.GetVSphereImage()

	h.cacheProvisioning.mu.Lock()
	pending := make([]string, len(h.cacheProvisioning.pending))
	copy(pending, h.cacheProvisioning.pending)
	using := make([]string, len(h.cacheProvisioning.using))
	copy(using, h.cacheProvisioning.using)
	h.cacheProvisioning.mu.Unlock()

	h.cacheToDelete.mu.Lock()
	toDelete := make([]string, len(h.cacheToDelete.list))
	copy(toDelete, h.cacheToDelete.list)
	h.cacheToDelete.mu.Unlock()

	for _, machine := range h.getVirtualMachines(ctx) {
		if !strings.HasPrefix(machine.Name, "provision-v2") {
			continue
		}

		annot := getVirtualMachineCDSAnnotation(ctx, machine)
		if annot == nil || !annot.Provisioning {
			continue
		}

		if expectedModelPath != annot.VMwareModelPath {
			continue
		}

		// A finished provision is powered off, waiting to be claimed. A
		// powered-on one is either still being provisioned or already starting.
		if machine.Summary.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOff {
			continue
		}

		if sdk.IsInArray(machine.Name, pending) ||
			sdk.IsInArray(machine.Name, using) ||
			sdk.IsInArray(machine.Name, toDelete) {
			continue
		}

		return true
	}

	return false
}

func (h *HatcheryVSphere) FindProvisionnedWorker(ctx context.Context, model sdk.WorkerStarterWorkerModel) (*object.VirtualMachine, error) {
	// worker model v2, it's the vmWare model name
	expectedModelPath := model.GetVSphereImage()

	log.Debug(ctx, "searching for provisionned VM for model %q", expectedModelPath)

	machines := h.getVirtualMachines(ctx)
	for _, machine := range machines {
		if !strings.HasPrefix(machine.Name, "provision-v2") {
			continue
		}

		annot := getVirtualMachineCDSAnnotation(ctx, machine)
		if annot == nil {
			continue
		}

		log.Debug(ctx, "checking provision %q expectedModelPath:%v annot.Provisioning:%v", machine.Name, expectedModelPath, annot.Provisioning)

		// Provisionned machines contains provisioning flag to true
		if !annot.Provisioning {
			continue
		}

		if expectedModelPath != annot.VMwareModelPath {
			log.Debug(ctx, "provision %q - expectedModelPath:%s annotModelPath:%s - skip it", machine.Name, expectedModelPath, annot.VMwareModelPath)
			continue
		}

		h.cacheProvisioning.mu.Lock()
		if sdk.IsInArray(machine.Name, h.cacheProvisioning.pending) {
			h.cacheProvisioning.mu.Unlock()
			log.Debug(ctx, "provision %q is in pending provisioning - skip it", machine.Name)
			continue
		}

		h.cacheProvisioning.mu.Unlock()

		h.cacheToDelete.mu.Lock()
		if sdk.IsInArray(machine.Name, h.cacheToDelete.list) {
			h.cacheToDelete.mu.Unlock()
			log.Debug(ctx, "provision %q already mark to be deleted - skip it", machine.Name)
			continue
		}
		h.cacheToDelete.mu.Unlock()

		vm, err := h.vSphereClient.LoadVirtualMachine(ctx, machine.Name)
		if err != nil && strings.Contains(err.Error(), "not found") {
			log.Debug(ctx, "provision %q already used by another worker starter - skip it", machine.Name)
			continue
		} else if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to load vm provision %q", machine.Name)
			continue
		}

		vmEvents, err := h.vSphereClient.LoadVirtualMachineEvents(ctx, vm, "VmPoweredOffEvent")
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to load VmStartingEvent events: %v", err)
			continue
		}

		if len(vmEvents) == 0 {
			log.Debug(ctx, "no VmPoweredOffEvent found - we skip this provision")
			continue
		}

		h.cacheProvisioning.mu.Lock()
		if sdk.IsInArray(machine.Name, h.cacheProvisioning.using) {
			log.Debug(ctx, "provision %q already used - skipping", machine.Name)
			h.cacheProvisioning.mu.Unlock()
			continue
		}

		h.cacheProvisioning.using = append(h.cacheProvisioning.using, machine.Name)
		h.cacheProvisioning.mu.Unlock()

		log.Debug(ctx, "we use this provision %q", machine.Name)
		return vm, nil
	}

	log.Debug(ctx, "unable to find provisionned VM for model %q", expectedModelPath)
	return nil, nil
}
