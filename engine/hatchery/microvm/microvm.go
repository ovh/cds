package microvm

import (
	"context"
	"fmt"
	"slices"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/telemetry"
)

// New instanciates a new hatchery microvm
func New() *HatcheryMicroVM {
	s := new(HatcheryMicroVM)
	s.GoRoutines = sdk.NewGoRoutines(context.Background())
	return s
}

var _ hatchery.InterfaceWithModels = new(HatcheryMicroVM)

// InitHatchery starts the cleanup routine
func (h *HatcheryMicroVM) InitHatchery(ctx context.Context) error {
	if err := h.Common.Init(ctx, h); err != nil {
		return err
	}
	h.GoRoutines.Run(ctx, "hatchery microvm routines", func(ctx context.Context) {
		h.routines(ctx)
	})
	return nil
}

// Init cdsclient config. The hatchery is v2 only: the v1 token is ignored.
func (h *HatcheryMicroVM) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(HatcheryConfiguration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid microvm hatchery configuration"))
	}

	h.Router = &api.Router{
		Mux:    mux.NewRouter(),
		Config: sConfig.HTTP,
	}

	cfg.Host = sConfig.API.HTTP.URL
	cfg.TokenV2 = sConfig.API.TokenV2
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryMicroVM) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return sdk.WithStack(fmt.Errorf("invalid configuration"))
	}
	h.Config.applyDefaults()

	var err error
	h.kubeClient, h.dynClient, err = initKubeClients(h.Config)
	if err != nil {
		return err
	}

	h.Common.Common.ServiceName = h.Config.Name
	h.Common.Common.ServiceType = sdk.TypeHatchery
	h.HTTPURL = h.Config.URL
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	h.Common.Common.PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(h.Config.RSAPrivateKey))
	if err != nil {
		return fmt.Errorf("unable to parse RSA private Key: %v", err)
	}
	h.Common.Common.Region = h.Config.Provision.Region
	h.Common.Common.IgnoreJobWithNoRegion = h.Config.Provision.IgnoreJobWithNoRegion
	// Sent at signin: the API only dispatches microvm jobs to this hatchery
	h.Common.Common.ModelType = h.ModelType()

	return nil
}

// applyDefaults supplies the fallbacks of the keys left out of the
// configuration file: struct defaults only apply when a config is generated.
func (c *HatcheryConfiguration) applyDefaults() {
	if len(c.OSArch) == 0 {
		c.OSArch = []string{"linux/amd64"}
	}
	if c.DefaultVCPUs <= 0 {
		c.DefaultVCPUs = 2
	}
	if c.DefaultMemory <= 0 {
		c.DefaultMemory = 2048
	}
	if c.MaxVCPUs <= 0 {
		c.MaxVCPUs = 8
	}
	if c.MaxMemory <= 0 {
		c.MaxMemory = 16384
	}
	if c.RegistrationTimeout <= 0 {
		c.RegistrationTimeout = 600
	}
	if c.QuotaPendingTimeout <= 0 {
		c.QuotaPendingTimeout = 1800
	}
	if c.KillAwolWorkersInterval <= 0 {
		c.KillAwolWorkersInterval = 30
	}
}

func initKubeClients(config HatcheryConfiguration) (kubernetes.Interface, dynamic.Interface, error) {
	var cfg *rest.Config
	var err error
	if config.KubernetesConfigFile != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", config.KubernetesConfigFile)
		if err != nil {
			return nil, nil, sdk.WrapError(err, "cannot build config from file %s", config.KubernetesConfigFile)
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, sdk.WrapError(err, "unable to configure k8s InClusterConfig")
		}
	}
	cfg.Timeout = 10 * time.Second

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, sdk.WrapError(err, "unable to create kubernetes client")
	}
	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, sdk.WrapError(err, "unable to create kubernetes dynamic client")
	}
	return kubeClient, dynClient, nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryMicroVM) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := h.NewMonitoringStatus()
	mvms, err := h.listMicroVMs(ctx)
	if err != nil {
		ctx = log.ContextWithStackTrace(ctx, err)
		log.Warn(ctx, err.Error())
		m.AddLine(sdk.MonitoringStatusLine{Component: "MicroVMs", Value: "unable to list MicroVMs", Status: sdk.MonitoringStatusWarn})
		return m
	}
	var started, quotaPending int
	for i := range mvms {
		if isEnded(&mvms[i]) {
			continue
		}
		started++
		if isQuotaPending(&mvms[i]) {
			quotaPending++
		}
	}
	maxWorkerDisplay := fmt.Sprintf("%d", h.Config.Provision.MaxWorker)
	if h.Config.Provision.MaxWorker == 0 {
		maxWorkerDisplay = "unlimited"
	}
	m.AddLine(sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%s", started, maxWorkerDisplay), Status: sdk.MonitoringStatusOK})
	quotaStatus := sdk.MonitoringStatusOK
	if quotaPending > 0 {
		quotaStatus = sdk.MonitoringStatusWarn
	}
	m.AddLine(sdk.MonitoringStatusLine{Component: "MicroVMs waiting for quota", Value: fmt.Sprintf("%d", quotaPending), Status: quotaStatus})
	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryMicroVM) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return sdk.WithStack(fmt.Errorf("invalid hatchery microvm configuration"))
	}

	if err := hconfig.Check(); err != nil {
		return sdk.WithStack(fmt.Errorf("invalid hatchery microvm configuration: %v", err))
	}

	if hconfig.API.TokenV2 == "" {
		return sdk.WithStack(fmt.Errorf("invalid hatchery microvm configuration: the hatchery is v2 only, api.tokenV2 is mandatory"))
	}

	if hconfig.Namespace == "" {
		return sdk.WithStack(fmt.Errorf("missing valid kubernetes namespace"))
	}

	if hconfig.DefaultVCPUs > hconfig.MaxVCPUs && hconfig.MaxVCPUs > 0 {
		return sdk.WithStack(fmt.Errorf("defaultVCPUs %d is greater than maxVCPUs %d", hconfig.DefaultVCPUs, hconfig.MaxVCPUs))
	}
	if hconfig.DefaultMemory > hconfig.MaxMemory && hconfig.MaxMemory > 0 {
		return sdk.WithStack(fmt.Errorf("defaultMemory %d is greater than maxMemory %d", hconfig.DefaultMemory, hconfig.MaxMemory))
	}

	return nil
}

// Signin only signs in on the v2 API
func (h *HatcheryMicroVM) Signin(ctx context.Context, clientConfig cdsclient.ServiceConfig, srvConfig interface{}) error {
	return h.Common.SigninV2(ctx, clientConfig, srvConfig)
}

// Start inits client and routines for hatchery
func (h *HatcheryMicroVM) Start(ctx context.Context) error {
	return hatchery.Create(ctx, h)
}

// Serve start the hatchery server
func (h *HatcheryMicroVM) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

// Configuration returns Hatchery CommonConfiguration
func (h *HatcheryMicroVM) Configuration() service.HatcheryCommonConfiguration {
	return h.Config.HatcheryCommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryMicroVM) ModelType() string {
	return sdk.WorkerModelTypeMicroVM
}

// WorkerModelsEnabled returns no v1 worker model: the hatchery is v2 only
func (h *HatcheryMicroVM) WorkerModelsEnabled() ([]sdk.Model, error) {
	return nil, nil
}

// WorkerModelSecretList returns no secret: the hatchery is v2 only
func (h *HatcheryMicroVM) WorkerModelSecretList(_ sdk.Model) (sdk.WorkerModelSecrets, error) {
	return nil, nil
}

// NeedRegistration returns false: v2 worker models are never registered
func (h *HatcheryMicroVM) NeedRegistration(_ context.Context, _ *sdk.Model) bool {
	return false
}

// CanSpawn only accepts v2 microvm models whose size fits the configured maximum
func (h *HatcheryMicroVM) CanSpawn(ctx context.Context, model sdk.WorkerStarterWorkerModel, jobID string, _ []sdk.Requirement) bool {
	ctx, end := telemetry.Span(ctx, "microvm.CanSpawn")
	defer end()

	if model.ModelV2 == nil || model.ModelV2.Type != sdk.WorkerModelTypeMicroVM {
		log.Debug(ctx, "CanSpawn> Job %s: only v2 microvm worker models are supported", jobID)
		return false
	}
	if !slices.Contains(h.Config.OSArch, model.ModelV2.OSArch) {
		log.Debug(ctx, "CanSpawn> Job %s with worker model %s cannot be spawned. Got osarch %s and want %s", jobID, model.ModelV2.Name, model.ModelV2.OSArch, h.Config.OSArch)
		return false
	}
	spec := model.MicroVMSpec
	if spec.Kernel.URL == "" || spec.Rootfs.URL == "" {
		log.Warn(ctx, "CanSpawn> Job %s: worker model %s has no kernel or rootfs", jobID, model.ModelV2.Name)
		return false
	}
	vcpus, memory := h.size(spec)
	if vcpus > h.Config.MaxVCPUs || memory > h.Config.MaxMemory {
		log.Warn(ctx, "CanSpawn> Job %s: worker model %s asks for %d vCPUs and %d MiB, the hatchery allows %d vCPUs and %d MiB", jobID, model.ModelV2.Name, vcpus, memory, h.Config.MaxVCPUs, h.Config.MaxMemory)
		return false
	}
	return true
}

// CanAllocateResources refuses new workers while a MicroVM of the hatchery
// is refused by the namespace quota: creating more would only pile up
// MicroVMs that cannot be scheduled either.
func (h *HatcheryMicroVM) CanAllocateResources(ctx context.Context, _ sdk.WorkerStarterWorkerModel, jobID string, _ []sdk.Requirement) (bool, error) {
	mvms, err := h.listMicroVMs(ctx)
	if err != nil {
		return false, err
	}
	for i := range mvms {
		if isQuotaPending(&mvms[i]) {
			log.Info(ctx, "CanAllocateResources> Job %s: MicroVM %s is waiting for the namespace quota, no new worker for now", jobID, mvms[i].GetName())
			return false, nil
		}
	}
	return true, nil
}

// size returns the vCPUs and memory of a worker, the model values or the defaults
func (h *HatcheryMicroVM) size(spec sdk.V2WorkerModelMicroVMSpec) (int, int) {
	vcpus := spec.VCPUs
	if vcpus == 0 {
		vcpus = h.Config.DefaultVCPUs
	}
	memory := spec.Memory
	if memory == 0 {
		memory = h.Config.DefaultMemory
	}
	return vcpus, memory
}

func (h *HatcheryMicroVM) routines(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(h.Config.KillAwolWorkersInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.GoRoutines.Exec(ctx, "killAwolWorkers", func(ctx context.Context) {
				if err := h.killAwolWorkers(ctx); err != nil {
					log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "cannot delete awol workers"))
				}
				if err := h.deleteOrphanSecrets(ctx); err != nil {
					log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "cannot delete orphan secrets"))
				}
			})
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Hatchery> MicroVM> Exiting routines")
			}
			return
		}
	}
}
