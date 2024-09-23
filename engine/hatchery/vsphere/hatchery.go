package vsphere

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
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
	m.AddLine(sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(ws), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})

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
	return nil
}

// CanSpawn return wether or not hatchery can spawn model
// requirements are not supported
func (h *HatcheryVSphere) CanSpawn(ctx context.Context, model sdk.WorkerStarterWorkerModel, jobID string, requirements []sdk.Requirement) bool {
	ctx, end := telemetry.Span(ctx, "vsphere.CanSpawn")
	defer end()
	if (model.ModelV1 != nil && model.ModelV1.Type != sdk.VSphere) || (model.ModelV2 != nil && model.ModelV2.Type != sdk.WorkerModelTypeVSphere) {
		return false
	}
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement ||
			r.Type == sdk.MemoryRequirement ||
			r.Type == sdk.HostnameRequirement ||
			r.Type == sdk.FlavorRequirement ||
			model.GetCmd() == "" {
			return false
		}
	}

	if sdk.IsJobIDForRegister(jobID) {
		// If jobID <= 0, it means that it's a call for a registration
		// So we have to check if there is no pending registration at this time
		// ie. virtual machine with name "<model>-tmp" or "register-<model>"

		for _, vm := range h.getVirtualMachines(ctx) {
			var vmAnnotation = getVirtualMachineCDSAnnotation(ctx, vm)
			if vmAnnotation == nil {
				continue
			}

			if model.ModelV1 == nil {
				log.Warn(ctx, "can't register a worker model v2: %s", model.GetName())
				return false
			}
			switch {
			case vm.Name == model.ModelV1.Name+"-tmp":
				log.Warn(ctx, "can't span worker for model %q registration because there is a temporary machine %q", model.GetName(), vm.Name)
				return false
			case strings.HasPrefix(vm.Name, "register-") && model.ModelV1.Name == vmAnnotation.WorkerModelPath:
				log.Warn(ctx, "can't span worker for model %q registration because there is a registering worker %q", model.GetName(), vm.Name)
				return false
			}
		}

		return true
	}

	// Check if there is a pending virtual machine with the same jobId in annotation - we want to avoid duplicates
	for _, vm := range h.getVirtualMachines(ctx) {
		var annot = getVirtualMachineCDSAnnotation(ctx, vm)
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

	// Check if there is one ip available
	if len(h.availableIPAddresses) > 0 {
		if _, err := h.findAvailableIP(ctx); err != nil {
			return false
		}
	}

	return true
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

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryVSphere) NeedRegistration(ctx context.Context, m *sdk.Model) bool {
	model, err := h.getVirtualMachineTemplateByName(ctx, m.Name)
	if err != nil {
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Warn(ctx, "unable to find vm template %q: %v", m.Name, err)
		return true
	}

	var annot = getVirtualMachineCDSAnnotation(ctx, model)
	if annot == nil {
		return true
	}

	isTemplateOutdated := fmt.Sprintf("%d", m.UserLastModified.Unix()) != annot.WorkerModelLastModified
	return !h.isMarkedToDelete(model) && (m.NeedRegistration || isTemplateOutdated)
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
			if s.Name == w.Name {
				log.Info(ctx, " killDisabledWorkers markToDelete %v", s.Name)
				h.markToDelete(ctx, s.Name)
				break
			}
		}
	}
}

func (h *HatcheryVSphere) isMarkedToDelete(s mo.VirtualMachine) bool {
	h.cacheToDelete.mu.Lock()
	var isMarkToDelete = sdk.IsInArray(s.Name, h.cacheToDelete.list)
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

	for _, s := range srvs {
		ctx = context.WithValue(ctx, cdslog.AuthWorkerName, s.Name)

		var annot = getVirtualMachineCDSAnnotation(ctx, s)
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
				log.Error(ctx, "killAwolServers> cannot delete server (markedToDelete) %s", s.Name)
			}
			continue
		}

		// skipping vm starting with provision-
		if strings.HasPrefix(s.Name, "provision-") {
			continue
		}

		if annot.Model {
			continue
		}

		// reload virtual machine to have fresh data from vsphere
		vm, err := h.vSphereClient.LoadVirtualMachine(ctx, s.Name)
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to load vm: %v", err)
			return
		}

		// gettings events for this vm, we have to check if we have a types.VmStartingEvent
		vmEvents, err := h.vSphereClient.LoadVirtualMachineEvents(ctx, vm, "VmStartingEvent", "VmPoweredOffEvent")
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to load VmStartingEvent events: %v", err)
			return
		}

		// if we don't have a types.VmStartingEvent, we skip this vm
		if len(vmEvents) == 0 {
			log.Debug(ctx, "killAwolServers> no VmStartingEvent found - we keep this vm")
			continue
		}
		// we can have many VmStartingEvent: one for the provision, another with the worker from on the provision

		var vmStartedTime, vmPoweredOffTime time.Time
		var foundStarted, foundPoweredOff bool
		for _, event := range vmEvents {
			// events on the vm can contain event from the provision.
			// Skipping this event if it's a provision's event
			if strings.Contains(event.GetEvent().FullFormattedMessage, "provision-") {
				log.Debug(ctx, "killAwolServers> skipping event %+v with provision text", event)
				continue
			}

			// take most recent VmStartingEvent and vmPoweredOffTime
			switch reflect.TypeOf(event).Elem().Name() {
			case "VmStartingEvent":
				// first event found, take it
				if !foundStarted {
					vmStartedTime = event.GetEvent().CreatedTime
					foundStarted = true
					continue
				}
				// if current event is youngest than previous, take it
				if event.GetEvent().CreatedTime.After(vmStartedTime) {
					vmStartedTime = event.GetEvent().CreatedTime
				}
			case "VmPoweredOffEvent":
				if !foundPoweredOff {
					vmPoweredOffTime = event.GetEvent().CreatedTime
					foundPoweredOff = true
					continue
				}
				if event.GetEvent().CreatedTime.After(vmPoweredOffTime) {
					vmPoweredOffTime = event.GetEvent().CreatedTime
				}
			}
		}

		if !foundStarted {
			log.Debug(ctx, "killAwolServers> vmStartedTime NOT found for %v - events analyzed:%d", s.Name, len(vmEvents))
			continue
		}
		if foundPoweredOff {
			// this should not happen, but we check if the powerOff Time is AFTER the started time
			if vmPoweredOffTime.Before(vmStartedTime) {
				foundPoweredOff = false
			}
		}

		log.Debug(ctx, "killAwolServers> vmStartedTime %v found: %s events analyzed:%d", s.Name, vmStartedTime, len(vmEvents))

		var existsOnAPISide bool
		for _, w := range allWorkers {
			if w.Name() == s.Name {
				existsOnAPISide = true
				break
			}
		}

		var toDelete bool

		log.Debug(ctx, "checking if %v is outdated. vmStartedTime: %v. existsOnAPISide:%t foundPoweredOff:%t vmPoweredOffTime:%v", s.Name, vmStartedTime, existsOnAPISide, foundPoweredOff, vmPoweredOffTime)

		if existsOnAPISide {
			expire := vmStartedTime.Add(time.Duration(h.Config.WorkerTTL) * time.Minute)
			if time.Now().After(expire) {
				toDelete = true
			}
		} else {
			// not currently visible on API. Can be:
			// - worker starting, not yet visible on api
			// - worker ended (job finished), removed on api

			// if we found a foundPoweredOff event, it's a worker used, and terminated. We can delete it
			if foundPoweredOff {
				// poweredOff found, we can delete this vm
				toDelete = true
			} else { // worker starting
				// not poweredOff found, check the WorkerRegistrationTTL
				expire := vmStartedTime.Add(time.Duration(h.Config.WorkerRegistrationTTL) * time.Minute)
				if time.Now().After(expire) {
					toDelete = true
				}
			}
		}

		if !toDelete {
			log.Debug(ctx, "We keep %v as not expired", s.Name)
			continue
		}

		// debug with event if necessary
		if log.Factory().GetLevel() == log.LevelDebug {
			events, err := h.vSphereClient.LoadVirtualMachineEvents(ctx, vm, "")
			if err != nil {
				log.Error(ctx, "event machine %q - can't load LoadVirtualMachineEvents", s.Name, err)
			}
			for _, e := range events {
				log.Debug(ctx, "event machine %q - event: %T time:%v msg:%+v", s.Name, e, e.GetEvent().CreatedTime, e.GetEvent().FullFormattedMessage)
			}
		}

		log.Info(ctx, "deleting machine %q - expired. vmStartedTime: %v. existsOnAPISide:%t foundPoweredOff:%t", s.Name, vmStartedTime, existsOnAPISide, foundPoweredOff)

		if err := h.deleteServer(ctx, s); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "killAwolServers> cannot delete server (expire) %s", s.Name)
		}
	}
}

func (h *HatcheryVSphere) provisioningV2(ctx context.Context) {
	h.cacheProvisioning.mu.Lock()

	// Count exiting provisionned machine for each vmware model
	var mapAlreadyProvisionned = make(map[string]int)
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
			mapAlreadyProvisionned[annot.VMwareModelPath] = mapAlreadyProvisionned[annot.VMwareModelPath] + 1
		}
	}
	h.cacheProvisioning.mu.Unlock()

	var toProvisionVMWareModelNames []string
	for i := range h.Config.WorkerProvisioning {
		modelVMware := h.Config.WorkerProvisioning[i].ModelVMWare
		number := h.Config.WorkerProvisioning[i].Number
		if modelVMware == "" || number == 0 {
			continue
		}

		log.Info(ctx, "model vmware %q provisioning: %d/%d", modelVMware, mapAlreadyProvisionned[modelVMware], number)

		nbToProvision := int(number) - mapAlreadyProvisionned[modelVMware]
		if nbToProvision == 0 {
			continue
		}
		for i = 1; i <= nbToProvision; i++ {
			toProvisionVMWareModelNames = append(toProvisionVMWareModelNames, modelVMware)
		}
	}

	if len(toProvisionVMWareModelNames) == 0 {
		return
	}

	poolSize := h.Config.WorkerProvisioningPoolSize
	if poolSize == 0 {
		poolSize = 1
	}

	// Provision workers
	wg := new(sync.WaitGroup)
	for i := 0; i < len(toProvisionVMWareModelNames) && i < poolSize; i++ {
		wg.Add(1)
		go func(modelVMware string) {
			defer wg.Done()

			workerName := namesgenerator.GenerateWorkerName("provision-v2")

			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.pending = append(h.cacheProvisioning.pending, workerName)
			h.cacheProvisioning.mu.Unlock()

			if err := h.ProvisionWorkerV2(ctx, modelVMware, workerName); err != nil {
				ctx = log.ContextWithStackTrace(ctx, err)
				log.Error(ctx, "unable to provision vmware model %q: %v", modelVMware, err)
			}

			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.pending = sdk.DeleteFromArray(h.cacheProvisioning.pending, workerName)
			h.cacheProvisioning.mu.Unlock()
		}(toProvisionVMWareModelNames[i])
	}
	wg.Wait()
}

func (h *HatcheryVSphere) provisioningV1(ctx context.Context) {
	h.cacheProvisioning.mu.Lock()

	// Count exiting provisionned machine for each model
	var mapAlreadyProvisionned = make(map[string]int)
	machines := h.getVirtualMachines(ctx)
	for _, machine := range machines {
		if !strings.HasPrefix(machine.Name, "provision-v1") {
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
			mapAlreadyProvisionned[annot.WorkerModelPath] = mapAlreadyProvisionned[annot.WorkerModelPath] + 1
		}
	}
	h.cacheProvisioning.mu.Unlock()

	// Count provision to create for each model
	mapToProvision := make(map[string]int)
	mapModels := make(map[string]sdk.Model)
	for i := range h.Config.WorkerProvisioning {
		modelPath := h.Config.WorkerProvisioning[i].ModelPath
		number := h.Config.WorkerProvisioning[i].Number
		if modelPath == "" || number == 0 {
			continue
		}

		tuple := strings.Split(modelPath, "/")
		if len(tuple) != 2 {
			log.Error(ctx, "invalid model name %q", modelPath)
			continue
		}

		model, err := h.Client.WorkerModelGet(tuple[0], tuple[1])
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Warn(ctx, "unable to get model name %q: %v", modelPath, err)
			continue
		}

		if model.CheckRegistration || model.NeedRegistration {
			log.Info(ctx, "model %q needs registration. skip provisioning.", modelPath)
			continue
		}

		log.Info(ctx, "model %q provisioning: %d/%d", modelPath, mapAlreadyProvisionned[modelPath], number)

		mapModels[modelPath] = model
		count := int(number) - mapAlreadyProvisionned[modelPath]
		if count > 0 {
			mapToProvision[modelPath] = count
		}
	}

	// Distribute models in provision queue
	countModelToProvision := len(mapToProvision)
	if countModelToProvision == 0 {
		return
	}
	poolSize := h.Config.WorkerProvisioningPoolSize
	if poolSize == 0 {
		poolSize = 1
	}
	var provisionQueue []string
	for len(mapToProvision) > 0 {
		for i := range h.Config.WorkerProvisioning {
			modelPath := h.Config.WorkerProvisioning[i].ModelPath
			count, ok := mapToProvision[modelPath]
			if !ok {
				continue
			}
			if count == 0 {
				delete(mapToProvision, modelPath)
				continue
			}
			provisionQueue = append(provisionQueue, modelPath)
			mapToProvision[modelPath] = mapToProvision[modelPath] - 1
		}
	}

	// Provision workers
	wg := new(sync.WaitGroup)
	for i := 0; i < len(provisionQueue) && i < poolSize; i++ {
		modelPath := provisionQueue[i]
		wg.Add(1)
		go func() {
			defer wg.Done()

			workerName := namesgenerator.GenerateWorkerName("provision-v1")

			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.pending = append(h.cacheProvisioning.pending, workerName)
			h.cacheProvisioning.mu.Unlock()

			if err := h.ProvisionWorkerV1(ctx, mapModels[modelPath], workerName); err != nil {
				ctx = log.ContextWithStackTrace(ctx, err)
				log.Error(ctx, "unable to provision model %q: %v", modelPath, err)
			}

			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.pending = sdk.DeleteFromArray(h.cacheProvisioning.pending, workerName)
			h.cacheProvisioning.mu.Unlock()
		}()
	}
	wg.Wait()
}
