package vsphere

import (
	"context"
	"encoding/json"
	"fmt"

	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/slug"
)

// New instanciates a new Hatchery vsphere
func New() *HatcheryVSphere {
	s := new(HatcheryVSphere)
	s.GoRoutines = sdk.NewGoRoutines()
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
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

	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

func (h *HatcheryVSphere) GetLogger() *logrus.Logger {
	return h.ServiceLogger
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryVSphere) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	h.Common.Common.ServiceName = h.Config.Name
	h.Common.Common.ServiceType = sdk.TypeHatchery
	h.HTTPURL = h.Config.URL
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	var err error
	h.Common.Common.PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(h.Config.RSAPrivateKey))
	if err != nil {
		return fmt.Errorf("unable to parse RSA private Key: %v", err)
	}

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryVSphere) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := h.NewMonitoringStatus()
	m.AddLine(sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(h.WorkersStarted(ctx)), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})
	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryVSphere) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid hatchery vsphere configuration")
	}

	if err := hconfig.Check(); err != nil {
		return fmt.Errorf("Invalid hatchery vsphere configuration: %v", err)
	}

	if hconfig.VSphereUser == "" {
		return fmt.Errorf("vsphere-user is mandatory")
	}

	if hconfig.VSphereEndpoint == "" {
		return fmt.Errorf("vsphere-endpoint is mandatory")
	}

	if hconfig.VSpherePassword == "" {
		return fmt.Errorf("vsphere-password is mandatory")
	}

	if hconfig.VSphereDatacenterString == "" {
		return fmt.Errorf("vsphere-datacenter is mandatory")
	}

	if hconfig.IPRange != "" {
		_, err := sdk.IPinRanges(context.Background(), hconfig.IPRange)
		if err != nil {
			return fmt.Errorf("flag or environment variable openstack-ip-range error: %v", err)
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

	if jobID > 0 {
		return true
	}

	// If jobID <= 0, it means that it's a call for a registration
	// So we have to check if there is no pending registration at this time
	// ie. virtual machine with name "<model>-tmp" or "register-<model>"

	for _, vm := range h.getVirtualMachines(ctx) {
		switch {
		case vm.Name == model.Name+"-tmp":
			log.Warn(ctx, "can't span worker for model %q registration because, there is a temporary machine %q", model.Name, vm.Name)
			return false
		case strings.HasPrefix(vm.Name, "register-"+slug.Convert(model.Name)):
			log.Warn(ctx, "can't span worker for model %q registration because, there is a registering worker %q", model.Name, vm.Name)
			return false
		}
	}

	return true
}

// Start inits client and routines for hatchery
func (h *HatcheryVSphere) Start(ctx context.Context) error {
	return hatchery.Create(ctx, h)
}

// Serve start the hatchery server
func (h *HatcheryVSphere) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryVSphere) Configuration() service.HatcheryCommonConfiguration {
	return h.Config.HatcheryCommonConfiguration
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryVSphere) NeedRegistration(ctx context.Context, m *sdk.Model) bool {
	model, err := h.getVirtualMachineTemplateByName(ctx, m.Name)
	if err != nil {
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "unable to get get vm template %q: %v", m.Name, err)
		return true
	}
	if model.Config == nil || model.Config.Annotation == "" {
		return true
	}

	var annot annotation
	if err := json.Unmarshal([]byte(model.Config.Annotation), &annot); err != nil {
		return true
	}

	isTemplateOutdated := fmt.Sprintf("%d", m.UserLastModified.Unix()) != annot.WorkerModelLastModified

	log.Debug(ctx, "%v %v %v", annot.ToDelete, m.NeedRegistration, isTemplateOutdated)

	return !annot.ToDelete && (m.NeedRegistration || isTemplateOutdated)
}

// WorkerModelsEnabled returns Worker model enabled
func (h *HatcheryVSphere) WorkerModelsEnabled() ([]sdk.Model, error) {
	return h.CDSClient().WorkerModelEnabledList()
}

// WorkerModelSecretList returns secret for given model.
func (h *HatcheryVSphere) WorkerModelSecretList(m sdk.Model) (sdk.WorkerModelSecrets, error) {
	return h.CDSClient().WorkerModelSecretList(m.Group.Name, m.Name)
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryVSphere) WorkersStartedByModel(ctx context.Context, model *sdk.Model) int {
	var x int
	for _, s := range h.getVirtualMachines(ctx) {
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(model.Name)) {
			x++
		}
	}
	log.Debug(ctx, "WorkersStartedByModel> %s : %d", model.Name, x)

	return x
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryVSphere) WorkersStarted(ctx context.Context) []string {
	srvs := h.getVirtualMachines(ctx)
	res := make([]string, len(srvs))
	for i, s := range srvs {
		if strings.Contains(strings.ToLower(s.Name), "worker") {
			res[i] = s.Name
		}
	}
	return res
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

// killAwolServers kill unused servers
func (h *HatcheryVSphere) killAwolServers(ctx context.Context) {
	srvs := h.getVirtualMachines(ctx)

	for _, s := range srvs {
		var annot annotation
		if s.Config == nil || s.Config.Annotation == "" {
			continue
		}
		if err := json.Unmarshal([]byte(s.Config.Annotation), &annot); err != nil {
			log.Warn(ctx, "unable to parse annotations %q on %q: %v", s.Config.Annotation, s.Name, err)
			continue
		}

		var isPoweredOff = s.Summary.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOn
		if annot.ToDelete || (isPoweredOff && (!annot.Model || annot.RegisterOnly)) {
			if err := h.deleteServer(ctx, s); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "killAwolServers> cannot delete server %s", s.Name)
			}
		}
	}
}
