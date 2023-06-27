package vsphere

import (
	"context"
	"fmt"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/namesgenerator"
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
	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryVSphere) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return sdk.WithStack(fmt.Errorf("Invalid hatchery vsphere configuration"))
	}

	if err := hconfig.Check(); err != nil {
		return sdk.WithStack(fmt.Errorf("Invalid hatchery vsphere configuration: %v", err))
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
func (h *HatcheryVSphere) CanSpawn(ctx context.Context, model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	if model.Type != sdk.VSphere {
		return false
	}
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement ||
			r.Type == sdk.MemoryRequirement ||
			r.Type == sdk.HostnameRequirement ||
			model.ModelVirtualMachine.Cmd == "" {
			return false
		}
	}

	if jobID <= 0 {
		// If jobID <= 0, it means that it's a call for a registration
		// So we have to check if there is no pending registration at this time
		// ie. virtual machine with name "<model>-tmp" or "register-<model>"

		for _, vm := range h.getVirtualMachines(ctx) {
			var vmAnnotation = getVirtualMachineCDSAnnotation(ctx, vm)
			if vmAnnotation == nil {
				continue
			}

			switch {
			case vm.Name == model.Name+"-tmp":
				log.Warn(ctx, "can't span worker for model %q registration because there is a temporary machine %q", model.Name, vm.Name)
				return false
			case strings.HasPrefix(vm.Name, "register-") && model.Name == vmAnnotation.WorkerModelPath:
				log.Warn(ctx, "can't span worker for model %q registration because there is a registering worker %q", model.Name, vm.Name)
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
			log.Info(ctx, "can't span worker for job %d because there is a registering worker %q for the same job", jobID, vm.Name)
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

func getVirtualMachineCDSAnnotation(ctx context.Context, srv mo.VirtualMachine) *annotation {
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
// not necessarily register on CDS yet
func (h *HatcheryVSphere) WorkersStarted(ctx context.Context) ([]string, error) {
	srvs := h.getVirtualMachines(ctx)
	res := make([]string, 0, len(srvs))
	for _, s := range srvs {
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
				log.Info(ctx, " killDisabledWorkers %v", s.Name)
				_ = h.deleteServer(ctx, s)
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
	allWorkers, err := h.CDSClient().WorkerList(ctx)
	if err != nil {
		ctx := sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "unable to load workers from CDS: %v", err)
		return
	}

	srvs := h.getVirtualMachines(ctx)

	for _, s := range srvs {
		var annot = getVirtualMachineCDSAnnotation(ctx, s)
		if annot == nil {
			continue
		}

		var isMarkToDelete = h.isMarkedToDelete(s)
		var isPoweredOff = s.Summary.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOn

		if !isPoweredOff && !isMarkToDelete {
			var bootTime = annot.Created
			if s.Runtime.BootTime != nil {
				bootTime = *s.Runtime.BootTime
			}

			// If the worker is not registered on CDS API the TTL is WorkerRegistrationTTL (default 10 minutes)
			var expire = bootTime.Add(time.Duration(h.Config.WorkerRegistrationTTL) * time.Minute)
			// Else it's WorkerTTL (default 120 minutes)
			for _, w := range allWorkers {
				if w.Name == s.Name {
					expire = bootTime.Add(time.Duration(h.Config.WorkerTTL) * time.Minute)
					break
				}
			}

			if sdk.IsInArray(s.Name, h.cacheProvisioning.restarting) {
				continue
			}

			log.Debug(ctx, "checking if %v is outdated. Created on :%v. Expires on %v", s.Name, bootTime, expire)
			// If the VM is older that the WorkerTTL config, let's mark it as delete

			if time.Now().After(expire) {
				vm, err := h.vSphereClient.LoadVirtualMachine(ctx, s.Name)
				if err != nil {
					ctx = sdk.ContextWithStacktrace(ctx, err)
					log.Error(ctx, "unable to load vm %s: %v", s.Name, err)
					continue
				}
				log.Info(ctx, "virtual machine %q as been created on %q, it has to be deleted", s.Name, bootTime)
				h.markToDelete(ctx, vm)
			}
		}

		// If the VM is mark as delete or is OFF and is not a model or a register-only VM, let's delete it
		// We also exclude not used provisionned VM from deletion
		isNotUsedProvisionned := annot.Provisioning && annot.WorkerName == s.Name
		if isMarkToDelete || (isPoweredOff && (!annot.Model || annot.RegisterOnly) && !isNotUsedProvisionned) {
			if err := h.deleteServer(ctx, s); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "killAwolServers> cannot delete server %s", s.Name)
			}
		}
	}
}

func (h *HatcheryVSphere) provisioning(ctx context.Context) {
	if len(h.Config.WorkerProvisioning) == 0 {
		log.Debug(ctx, "provisioning is disabled")
		return
	}

	if len(h.cacheProvisioning.pending) > 0 {
		log.Debug(ctx, "provisioning is still on going")
		return
	}

	h.cacheProvisioning.mu.Lock()

	var mapAlreadyProvisionned = make(map[string]int)
	machines := h.getVirtualMachines(ctx)
	for _, machine := range machines {
		annot := getVirtualMachineCDSAnnotation(ctx, machine)
		if annot == nil {
			continue
		}
		// Provisionned machines are powered off
		if annot.Provisioning && machine.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOn {
			mapAlreadyProvisionned[annot.WorkerModelPath] = mapAlreadyProvisionned[annot.WorkerModelPath] + 1
		}
	}

	h.cacheProvisioning.mu.Unlock()

	for i := range h.Config.WorkerProvisioning {
		modelPath := h.Config.WorkerProvisioning[i].ModelPath
		number := h.Config.WorkerProvisioning[i].Number

		if number == 0 {
			continue // If provisioning is disabled
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

		for i := 0; i < int(number)-mapAlreadyProvisionned[modelPath]; i++ {
			workerName := namesgenerator.GenerateWorkerName(modelPath, "provision")

			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.pending = append(h.cacheProvisioning.pending, workerName)
			h.cacheProvisioning.mu.Unlock()

			if err := h.ProvisionWorker(ctx, model, workerName); err != nil {
				ctx = log.ContextWithStackTrace(ctx, err)
				log.Error(ctx, "unable to provision model %q: %v", modelPath, err)
			}

			h.cacheProvisioning.mu.Lock()
			h.cacheProvisioning.pending = sdk.DeleteFromArray(h.cacheProvisioning.pending, workerName)
			h.cacheProvisioning.mu.Unlock()
		}
	}
}
