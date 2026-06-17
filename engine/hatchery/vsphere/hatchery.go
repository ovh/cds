package vsphere

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

// New instanciates a new Hatchery vsphere
func New() *HatcheryVSphere {
	s := new(HatcheryVSphere)
	s.GoRoutines = sdk.NewGoRoutines(context.Background())
	return s
}

var _ hatchery.InterfaceWithModels = new(HatcheryVSphere)

// Init cdsclient config.
func (h *HatcheryVSphere) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(HatcheryConfiguration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid vsphere hatchery configuration"))
	}
	h.Router = &api.Router{
		Mux:    mux.NewRouter(),
		Config: sConfig.HTTP,
	}
	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.TokenV2 = sConfig.API.TokenV2
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryVSphere) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return sdk.WithStack(fmt.Errorf("Invalid configuration"))
	}

	h.Common.Common.ServiceName = h.Config.Name
	h.Common.Common.ServiceType = sdk.TypeHatchery
	h.HTTPURL = h.Config.URL
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	var err error
	h.Common.Common.PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(h.Config.RSAPrivateKey))
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to parse RSA private Key: %v", err))
	}
	h.Common.Common.Region = h.Config.Provision.Region
	h.Common.Common.IgnoreJobWithNoRegion = h.Config.Provision.IgnoreJobWithNoRegion
	h.Common.Common.ModelType = h.ModelType()

	if h.Config.WorkerTTL == 0 {
		h.Config.WorkerTTL = 120
	}
	if h.Config.WorkerRegistrationTTL == 0 {
		h.Config.WorkerRegistrationTTL = 10
	}
	if h.Config.KillAwolServersInterval == 0 {
		h.Config.KillAwolServersInterval = 60
	}
	if h.Config.FinishedWorkerGracePeriod == 0 {
		h.Config.FinishedWorkerGracePeriod = 180
	}

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryVSphere) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := h.NewMonitoringStatus()
	ws, err := h.WorkersStarted(ctx)
	if err != nil {
		ctx = log.ContextWithStackTrace(ctx, err)
		log.Warn(ctx, err.Error())
	}
	maxWorkerDisplay := fmt.Sprintf("%d", h.Config.Provision.MaxWorker)
	if h.Config.Provision.MaxWorker == 0 {
		maxWorkerDisplay = "unlimited"
	}
	m.AddLine(sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%s", len(ws), maxWorkerDisplay), Status: sdk.MonitoringStatusOK})

	vms, err := h.vSphereClient.ListVirtualMachines(ctx)
	if err != nil {
		log.Error(ctx, "unable to get virtual machines: %v", err)
		m.AddLine(sdk.MonitoringStatusLine{Component: "VirtualMachines", Value: "Error listing", Status: sdk.MonitoringStatusAlert})
	} else {
		m.AddLine(sdk.MonitoringStatusLine{Component: "VirtualMachines", Value: fmt.Sprintf("%d", len(vms)), Status: sdk.MonitoringStatusOK})
	}

	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryVSphere) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return sdk.WithStack(fmt.Errorf("invalid hatchery vsphere configuration"))
	}

	if err := hconfig.Check(); err != nil {
		return sdk.WithStack(fmt.Errorf("invalid hatchery vsphere configuration: %v", err))
	}

	if hconfig.VSphereUser == "" {
		return sdk.WithStack(fmt.Errorf("vsphere-user is mandatory"))
	}

	if hconfig.VSphereEndpoint == "" {
		return sdk.WithStack(fmt.Errorf("vsphere-endpoint is mandatory"))
	}

	if hconfig.VSpherePassword == "" {
		return sdk.WithStack(fmt.Errorf("vsphere-password is mandatory"))
	}

	if hconfig.VSphereDatacenterString == "" {
		return sdk.WithStack(fmt.Errorf("vsphere-datacenter is mandatory"))
	}

	if hconfig.IPRange != "" {
		_, err := sdk.IPinRanges(context.Background(), hconfig.IPRange)
		if err != nil {
			return sdk.WithStack(fmt.Errorf("flag or environment variable ip-range error: %v", err))
		}
	}

	for i, netCfg := range hconfig.Networks {
		if netCfg.IPRange == "" {
			return sdk.WithStack(fmt.Errorf("networks[%d]: iprange is required", i))
		}
		if netCfg.Gateway == "" {
			return sdk.WithStack(fmt.Errorf("networks[%d]: gateway is required", i))
		}
		if netCfg.SubnetMask == "" {
			return sdk.WithStack(fmt.Errorf("networks[%d]: subnetMask is required", i))
		}
		_, err := sdk.IPinRanges(context.Background(), netCfg.IPRange)
		if err != nil {
			return sdk.WithStack(fmt.Errorf("networks[%d] ip-range error: %v", i, err))
		}
	}
	return nil
}

// CanSpawn return wether or not hatchery can spawn model
// some requirements are not supported
// This func is called with job v1 and job v2.
func (h *HatcheryVSphere) CanSpawn(ctx context.Context, model sdk.WorkerStarterWorkerModel, jobID string, requirements []sdk.Requirement) bool {
	ctx, end := telemetry.Span(ctx, "vsphere.CanSpawn")
	defer end()

	// Worker model v1 is no longer supported on vSphere: v1 jobs run on v2
	// worker models (resolved via GetDetaultModelV2Name).
	if model.ModelV1 != nil {
		return false
	}

	if model.ModelV2 != nil && model.ModelV2.Type != sdk.WorkerModelTypeVSphere {
		return false
	}

	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement ||
			r.Type == sdk.MemoryRequirement ||
			r.Type == sdk.HostnameRequirement ||
			model.GetCmd() == "" {
			return false
		}
	}

	// Check if there is a pending virtual machine with the same jobId in annotation - we want to avoid duplicates
	for _, vm := range h.getVirtualMachines(ctx) {
		annot := getVirtualMachineCDSAnnotation(ctx, vm)
		if annot == nil {
			continue
		}
		if annot.JobID == jobID {
			log.Info(ctx, "can't span worker for job %s because there is a registering worker %q for the same job", jobID, vm.Name)
			return false
		}
	}

	// Check in the local cache of pending StartingVM
	h.cachePendingJobID.mu.Lock()
	defer h.cachePendingJobID.mu.Unlock()
	for _, id := range h.cachePendingJobID.list {
		if id == jobID {
			return false
		}
	}

	// Workers are only started from pre-provisioned VMs (which already hold
	// their own IP, reserved at clone time): check that one is available for
	// this model instead of checking for a free IP. Checking for a free IP
	// here would wrongly refuse to spawn when the whole IP range is held by
	// provisioned machines, which is the nominal situation.
	if !h.hasAvailableProvisionedWorker(ctx, model) {
		log.Debug(ctx, "can't spawn worker for job %s: no provisioned worker available for model %q", jobID, model.GetName())
		return false
	}

	return true
}

func (h *HatcheryVSphere) CanAllocateResources(ctx context.Context, model sdk.WorkerStarterWorkerModel, jobID string, requirements []sdk.Requirement) (bool, error) {
	// Determine the resource footprint of the next worker
	var nextCPUs int
	var nextMemoryMB int

	// Amendment C: prefer flavor-defined resources over template resources
	flavorName := model.GetFlavor(requirements, h.Config.DefaultFlavor)
	if flavorName != "" {
		// Use flavor-defined resources
		flavor := h.getFlavor(flavorName)
		if flavor == nil {
			log.Warn(ctx, "CanAllocateResources> unknown flavor %q, falling back to template resources", flavorName)
		} else {
			nextCPUs = flavor.CPUs
			nextMemoryMB = int(flavor.MemoryMB)
			log.Debug(ctx, "CanAllocateResources> using flavor %q: %d vCPUs, %d MB", flavorName, nextCPUs, nextMemoryMB)
		}
	}

	// Fallback: read template resources if no flavor or flavor not found
	if nextCPUs == 0 || nextMemoryMB == 0 {
		templateName := model.GetVSphereImage()
		if templateName == "" {
			log.Warn(ctx, "CanAllocateResources> model has no vSphere image defined")
			return true, nil
		}

		var err error
		nextCPUs, nextMemoryMB, err = h.getTemplateResources(ctx, templateName)
		if err != nil {
			log.Warn(ctx, "CanAllocateResources> unable to determine resource footprint for template %q: %v", templateName, err)
			return true, nil // Graceful degradation: allow spawning if template resources can't be read
		}
	}

	// Primary check: Resource Pool capacity (always enabled, Amendment B)
	canFit, err := h.checkResourcePoolCapacity(ctx, nextCPUs, nextMemoryMB)
	if err != nil {
		log.Warn(ctx, "CanAllocateResources> Resource Pool check failed: %v", err)
		// Graceful degradation: continue to static limits if Resource Pool check fails
	} else if !canFit {
		log.Info(ctx, "CanAllocateResources> Resource Pool capacity insufficient for %d vCPUs, %d MB", nextCPUs, nextMemoryMB)
		return false, nil
	}

	// Supplementary checks: static limits (if configured)
	usedCPUs, usedMemoryMB := h.countAllocatedResources(ctx)

	// Check CPU limit
	if h.Config.MaxCPUs > 0 {
		if usedCPUs+nextCPUs > h.Config.MaxCPUs {
			log.Info(ctx, "CanAllocateResources> CPU limit reached: %d + %d > %d",
				usedCPUs, nextCPUs, h.Config.MaxCPUs)
			return false, nil
		}

		// Amendment C: Flavor starvation prevention
		if flavorName != "" && h.Config.CountSmallerFlavorToKeep > 0 {
			smallerCPUs := h.getSmallerFlavorCPUs(flavorName)
			if smallerCPUs > 0 && smallerCPUs != nextCPUs {
				needed := nextCPUs + h.Config.CountSmallerFlavorToKeep*smallerCPUs
				available := h.Config.MaxCPUs - usedCPUs
				if needed > available {
					log.Info(ctx, "CanAllocateResources> starvation prevention: need %d CPUs (%d + %d×%d reserve) but only %d available",
						needed, nextCPUs, h.Config.CountSmallerFlavorToKeep, smallerCPUs, available)
					return false, nil
				}
			}
		}
	}

	// Check memory limit
	if h.Config.MaxMemoryMB > 0 {
		if usedMemoryMB+nextMemoryMB > h.Config.MaxMemoryMB {
			log.Info(ctx, "CanAllocateResources> Memory limit reached: %d + %d > %d",
				usedMemoryMB, nextMemoryMB, h.Config.MaxMemoryMB)
			return false, nil
		}
	}

	return true, nil
}

// countAllocatedResources returns the total vCPUs and memory (MB) currently allocated
// by VMs managed by this hatchery (excluding template VMs and powered-off VMs).
// Powered-off VMs (e.g. provisioned workers waiting for a job) do not consume CPU or RAM
// in vSphere Resource Pools and are therefore excluded from the count.
func (h *HatcheryVSphere) countAllocatedResources(ctx context.Context) (int, int) {
	srvs := h.getRawVMs(ctx)

	var totalCPUs int
	var totalMemoryMB int

	for _, s := range srvs {
		annot := getVirtualMachineCDSAnnotation(ctx, s)
		if annot == nil || annot.HatcheryName != h.Name() {
			continue
		}
		// Exclude template VMs
		if annot.Model {
			continue
		}
		// Exclude powered-off VMs: they don't consume CPU/RAM in vSphere Resource Pools
		if s.Summary.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOff {
			continue
		}

		totalCPUs += int(s.Summary.Config.NumCpu)
		totalMemoryMB += int(s.Summary.Config.MemorySizeMB)
	}

	return totalCPUs, totalMemoryMB
}

// getTemplateResources returns the vCPUs and memory (MB) of a vSphere template.
func (h *HatcheryVSphere) getTemplateResources(ctx context.Context, templateName string) (int, int, error) {
	vm, err := h.getVirtualMachineTemplateByName(ctx, templateName)
	if err != nil {
		return 0, 0, sdk.WithStack(err)
	}

	return int(vm.Summary.Config.NumCpu), int(vm.Summary.Config.MemorySizeMB), nil
}

// getFlavor returns the flavor configuration for a given flavor name.
// Returns nil if the flavor is not found.
func (h *HatcheryVSphere) getFlavor(name string) *VSphereFlavorConfig {
	for i := range h.Config.Flavors {
		if strings.EqualFold(h.Config.Flavors[i].Name, name) {
			return &h.Config.Flavors[i]
		}
	}
	return nil
}

// getSmallerFlavorCPUs returns the smallest flavor CPU count that is smaller than the current flavor.
// Used for starvation prevention: reserve capacity for smaller flavors when spawning large flavors.
// Returns 0 if no smaller flavor exists.
func (h *HatcheryVSphere) getSmallerFlavorCPUs(currentFlavorName string) int {
	currentFlavor := h.getFlavor(currentFlavorName)
	if currentFlavor == nil {
		return 0
	}

	var smallestCPUs int
	for i := range h.Config.Flavors {
		f := &h.Config.Flavors[i]
		if strings.EqualFold(f.Name, currentFlavorName) {
			continue
		}
		if f.CPUs < currentFlavor.CPUs {
			if smallestCPUs == 0 || f.CPUs < smallestCPUs {
				smallestCPUs = f.CPUs
			}
		}
	}
	return smallestCPUs
}

// checkResourcePoolCapacity verifies if the Resource Pool has enough unreserved CPU and memory
// to accommodate a VM with the specified resource requirements.
func (h *HatcheryVSphere) checkResourcePoolCapacity(ctx context.Context, requiredCPUs int, requiredMemoryMB int) (bool, error) {
	pool, err := h.vSphereClient.LoadResourcePool(ctx)
	if err != nil {
		return false, sdk.WithStack(err)
	}

	var poolMo mo.ResourcePool
	if err := pool.Properties(ctx, pool.Reference(), []string{"runtime"}, &poolMo); err != nil {
		return false, sdk.WithStack(err)
	}

	// CPU is in MHz, not vCPUs — this is imprecise but it's what vSphere provides
	// We can't accurately convert MHz to vCPUs, so we check if there's *any* unreserved capacity
	if poolMo.Runtime.Cpu.UnreservedForVm <= 0 {
		log.Info(ctx, "checkResourcePoolCapacity> no unreserved CPU in Resource Pool")
		return false, nil
	}

	// Memory is in bytes
	requiredMemoryBytes := int64(requiredMemoryMB) * 1024 * 1024
	if poolMo.Runtime.Memory.UnreservedForVm < requiredMemoryBytes {
		log.Info(ctx, "checkResourcePoolCapacity> insufficient memory: need %d MB, available %d MB",
			requiredMemoryMB, poolMo.Runtime.Memory.UnreservedForVm/(1024*1024))
		return false, nil
	}

	return true, nil
}

func (h *HatcheryVSphere) Signin(ctx context.Context, clientConfig cdsclient.ServiceConfig, srvConfig interface{}) error {
	if err := h.Common.Signin(ctx, clientConfig, srvConfig); err != nil {
		return err
	}
	if err := h.Common.SigninV2(ctx, clientConfig, srvConfig); err != nil {
		return err
	}
	return nil
}

// Start inits client and routines for hatchery
func (h *HatcheryVSphere) Start(ctx context.Context) error {
	return hatchery.Create(ctx, h)
}

// Serve start the hatchery server
func (h *HatcheryVSphere) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

// Configuration returns Hatchery CommonConfiguration
func (h *HatcheryVSphere) Configuration() service.HatcheryCommonConfiguration {
	return h.Config.HatcheryCommonConfiguration
}

func getVirtualMachineCDSAnnotation(_ context.Context, srv mo.VirtualMachine) *annotation {
	if srv.Config == nil {
		return nil
	}
	if srv.Config.Annotation == "" {
		return nil
	}
	var annot annotation
	if err := sdk.JSONUnmarshal([]byte(srv.Config.Annotation), &annot); err != nil {
		return nil
	}
	return &annot
}

// NeedRegistration always returns false: worker model v1 (the only kind that
// registers into vSphere) is no longer supported. v2 worker models reference an
// existing vSphere template image and never need registration.
func (h *HatcheryVSphere) NeedRegistration(_ context.Context, _ *sdk.Model) bool {
	return false
}

// WorkerModelsEnabled returns Worker model enabled
func (h *HatcheryVSphere) WorkerModelsEnabled() ([]sdk.Model, error) {
	return h.CDSClient().WorkerModelEnabledList()
}

// WorkerModelSecretList returns secret for given model.
func (h *HatcheryVSphere) WorkerModelSecretList(m sdk.Model) (sdk.WorkerModelSecrets, error) {
	return h.CDSClient().WorkerModelSecretList(m.Group.Name, m.Name)
}

// WorkersStarted returns the list of workers started but
// not necessarily registered on CDS yet
func (h *HatcheryVSphere) WorkersStarted(ctx context.Context) ([]string, error) {
	srvs := h.getVirtualMachines(ctx)
	res := make([]string, 0, len(srvs))
	for _, s := range srvs {
		if strings.HasPrefix(s.Name, "provision-") {
			continue
		}
		res = append(res, s.Name)
	}
	return res, nil
}

// ModelType returns type of hatchery
func (*HatcheryVSphere) ModelType() string {
	return sdk.VSphere
}

func (h *HatcheryVSphere) GetDetaultModelV2Name(ctx context.Context, requirements []sdk.Requirement) string {
	if len(h.Config.DefaultWorkerModelsV2) == 0 {
		return ""
	}

	var binaries []string

	for _, req := range requirements {
		if req.Type == sdk.BinaryRequirement {
			binaries = append(binaries, req.Value)
		}
	}

	// no binary in job v1, take the first default model configured
	if len(binaries) == 0 {
		log.Debug(ctx, "GetDetaultModelVx2Name choose default model v2:%v", h.Config.DefaultWorkerModelsV2[0].WorkerModelV2)
		return h.Config.DefaultWorkerModelsV2[0].WorkerModelV2
	}

	// here, we have to search a worker model v2, matching all binaries existing in the job pre-requisite
	for _, modelV2 := range h.Config.DefaultWorkerModelsV2 {
		for _, binary := range binaries {
			if sdk.IsInArray(binary, modelV2.Binaries) {
				log.Debug(ctx, "GetDetaultModelV2Name choose default model v2 %v matching binaries %v", modelV2.WorkerModelV2, binaries)
				return modelV2.WorkerModelV2
			}
		}
	}
	// No default worker model v2 found
	return ""
}

// killDisabledWorkers kill workers which are disabled
func (h *HatcheryVSphere) killDisabledWorkers(ctx context.Context) {
	workerPoolDisabled, err := hatchery.WorkerPool(ctx, h, sdk.StatusDisabled)
	if err != nil {
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "killDisabledWorkers> Pool> Error: %v", err)
		return
	}

	srvs := h.getVirtualMachines(ctx)
	for _, w := range workerPoolDisabled {
		for _, s := range srvs {
			if s.Name == w.GetName() {
				log.Info(ctx, " killDisabledWorkers markToDelete %v", s.Name)
				h.markToDelete(ctx, s.Name)
				break
			}
		}
	}
}

func (h *HatcheryVSphere) isMarkedToDelete(s mo.VirtualMachine) bool {
	h.cacheToDelete.mu.Lock()
	isMarkToDelete := sdk.IsInArray(s.Name, h.cacheToDelete.list)
	h.cacheToDelete.mu.Unlock()
	return isMarkToDelete
}

// killAwolServers kill unused servers
func (h *HatcheryVSphere) killAwolServers(ctx context.Context) {
	allWorkers, err := h.WorkerList(ctx)
	if err != nil {
		ctx := sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "unable to load workers from CDS: %v", err)
		return
	}

	srvs := h.getVirtualMachines(ctx)

	// Track whether anything was deleted so we can trigger an immediate
	// provisioning refill once resources (and their reserved IPs) are freed.
	var deleted bool
	defer func() {
		if deleted {
			h.requestProvisioning(ctx)
		}
	}()

	for _, s := range srvs {
		ctx = context.WithValue(ctx, cdslog.AuthWorkerName, s.Name)

		annot := getVirtualMachineCDSAnnotation(ctx, s)
		if annot == nil {
			continue
		}
		if annot.HatcheryName != h.Name() {
			continue
		}

		// if VM is marked to be deleted by the spawn goroutine (could be provision- or real worker), then delete it now.
		if h.isMarkedToDelete(s) {
			log.Info(ctx, "deleting machine %q as it's already marked to be deleted", s.Name)
			if err := h.deleteServer(ctx, s); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "killAwolServers> cannot delete server (markedToDelete) %s: %v", s.Name, err)
			} else {
				deleted = true
			}
			continue
		}

		// Pooled provisions (still in the pool, holding their reserved IP on
		// purpose) and model templates must not be reaped here.
		if strings.HasPrefix(s.Name, "provision-") {
			continue
		}
		if annot.Model {
			continue
		}

		// Determine when this worker actually started. Prefer the worker start
		// time stamped on the annotation at claim time (persistent, survives a
		// restart, does not age out like vSphere events). For VMs created before
		// this field existed (transitional / upgrade), fall back to the VM's
		// vSphere creation date so they are still handled and eventually cleaned.
		startTime := annot.WorkerStartTime
		if startTime.IsZero() && s.Config != nil && s.Config.CreateDate != nil {
			startTime = *s.Config.CreateDate
		}
		if startTime.IsZero() {
			// No reliable timestamp at all (should not happen). Keep it rather than
			// risk deleting a VM we cannot reason about.
			log.Warn(ctx, "killAwolServers> no start time for %q, keeping it", s.Name)
			continue
		}

		poweredOff := s.Summary.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOff

		var existsOnAPISide bool
		for _, w := range allWorkers {
			if w.GetName() == s.Name {
				existsOnAPISide = true
				break
			}
		}

		// Decide when this VM expires:
		// - powered off and still registered on the API: the worker booted, ran, and
		//   is now down — the job is over. Reclaim it immediately.
		// - powered off and no longer on the API: a finished+deregistered worker, or
		//   an in-flight spawn between rename and power-on. The short
		//   FinishedWorkerGracePeriod reclaims the former quickly while still covering
		//   the latter (its WorkerStartTime is "now").
		// - powered on and registered: running worker, capped at WorkerTTL.
		// - powered on but not registered: still booting/registering, give it
		//   WorkerRegistrationTTL to show up.
		var expire time.Time
		switch {
		case poweredOff && existsOnAPISide:
			expire = startTime
		case poweredOff:
			expire = startTime.Add(time.Duration(h.Config.FinishedWorkerGracePeriod) * time.Second)
		case existsOnAPISide:
			expire = startTime.Add(time.Duration(h.Config.WorkerTTL) * time.Minute)
		default:
			expire = startTime.Add(time.Duration(h.Config.WorkerRegistrationTTL) * time.Minute)
		}

		if !time.Now().After(expire) {
			log.Debug(ctx, "killAwolServers> keeping %q (poweredOff=%t existsOnAPISide=%t startTime=%v expire=%v)", s.Name, poweredOff, existsOnAPISide, startTime, expire)
			continue
		}

		log.Info(ctx, "deleting machine %q - expired (poweredOff=%t existsOnAPISide=%t startTime=%v)", s.Name, poweredOff, existsOnAPISide, startTime)
		if err := h.deleteServer(ctx, s); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "killAwolServers> cannot delete server (expire) %s: %v", s.Name, err)
		} else {
			deleted = true
		}
	}
}

// roundRobinInterleave takes an ordered list of model keys and a deficit count
// per model, and returns a flat queue where models are interleaved one task at
// a time in config order. E.g. with deficits A=3, B=1, C=2 the result is
// [A, B, C, A, C, A].
func roundRobinInterleave(modelOrder []string, deficits map[string]int) []string {
	total := 0
	for _, c := range deficits {
		total += c
	}
	queue := make([]string, 0, total)
	remaining := make(map[string]int, len(deficits))
	for k, v := range deficits {
		remaining[k] = v
	}
	for len(remaining) > 0 {
		for _, model := range modelOrder {
			if remaining[model] <= 0 {
				continue
			}
			queue = append(queue, model)
			remaining[model]--
			if remaining[model] == 0 {
				delete(remaining, model)
			}
		}
	}
	return queue
}

func (h *HatcheryVSphere) provisioningV2(ctx context.Context) {
	h.cacheProvisioning.mu.Lock()

	// --- Step 1: compute current state and deficit per model ---
	// Count ALL VMs with Provisioning annotation regardless of power state,
	// so orphaned VMs (powered on, stuck) are included. This ensures that
	// reconciled VMs don't cause additional clones in the deficit calculation.
	mapAlreadyProvisionned := make(map[string]int)
	mapProvisionedMachines := make(map[string][]mo.VirtualMachine)
	machines := h.getVirtualMachines(ctx)
	for _, machine := range machines {
		if !strings.HasPrefix(machine.Name, "provision-v2") {
			continue
		}
		annot := getVirtualMachineCDSAnnotation(ctx, machine)
		if annot == nil {
			continue
		}
		if annot.HatcheryName != h.Name() {
			continue
		}
		if annot.Provisioning {
			mapAlreadyProvisionned[annot.VMwareModelPath]++
			mapProvisionedMachines[annot.VMwareModelPath] = append(
				mapProvisionedMachines[annot.VMwareModelPath], machine)
		}
	}
	h.cacheProvisioning.mu.Unlock()

	// Build configured counts and compute deficit for each model
	configuredCounts := make(map[string]int)
	deficits := make(map[string]int)
	for i := range h.Config.WorkerProvisioning {
		modelVMware := h.Config.WorkerProvisioning[i].ModelVMWare
		number := h.Config.WorkerProvisioning[i].Number
		if modelVMware == "" {
			continue
		}
		configuredCounts[modelVMware] = number
		if number == 0 {
			continue
		}
		existing := mapAlreadyProvisionned[modelVMware]
		log.Info(ctx, "model vmware %q provisioning: %d/%d",
			modelVMware, existing, number)
		if delta := number - existing; delta > 0 {
			deficits[modelVMware] = delta
		}
	}

	// Remove excess provisioned VMs when config is decreased or model removed
	for modelPath, provisionedMachines := range mapProvisionedMachines {
		desired := configuredCounts[modelPath]
		if excess := len(provisionedMachines) - desired; excess > 0 {
			log.Info(ctx, "model vmware %q deprovisioning: removing %d excess (have %d, want %d)",
				modelPath, excess, len(provisionedMachines), desired)
			for i := 0; i < excess; i++ {
				h.markToDelete(ctx, provisionedMachines[i].Name)
			}
		}
	}

	// --- Step 2: interleave models using round-robin ---
	// Produces a fair ordering so that models with larger counts don't
	// monopolize the provisioning queue.
	modelOrder := make([]string, 0, len(h.Config.WorkerProvisioning))
	for i := range h.Config.WorkerProvisioning {
		m := h.Config.WorkerProvisioning[i].ModelVMWare
		if _, ok := deficits[m]; ok {
			modelOrder = append(modelOrder, m)
		}
	}
	provisionQueue := roundRobinInterleave(modelOrder, deficits)

	// --- Step 3: cap by available IPs and enqueue tasks ---
	ipBudget := -1
	if len(h.availableIPAddresses) > 0 {
		ipBudget = h.countAvailableIPs(ctx)
		log.Info(ctx, "provisioning IP budget: %d available", ipBudget)
	}
	if ipBudget == 0 {
		log.Warn(ctx, "provisioning stopped: no IPs available")
		provisionQueue = nil
	} else if ipBudget > 0 && len(provisionQueue) > ipBudget {
		log.Warn(ctx, "provisioning capped to %d tasks (IP budget)", ipBudget)
		provisionQueue = provisionQueue[:ipBudget]
	}

	var batch sync.WaitGroup
	h.reconcileProvisionedVMs(ctx, machines, "provision-v2", &batch)

	for _, modelVMware := range provisionQueue {
		batch.Add(1)
		h.provisioningPool.tasks <- provisionTask{
			cloneV2: modelVMware,
			wg:      &batch,
		}
	}

	batch.Wait()
}
