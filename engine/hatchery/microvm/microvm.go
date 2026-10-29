package microvm

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
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
	// Sent at signin: the API only dispatches the jobs of this type to the hatchery
	h.Common.Common.ModelType = h.ModelType()

	return nil
}

// applyDefaults supplies the fallbacks of the keys left out of the
// configuration file: struct defaults only apply when a config is generated.
func (c *HatcheryConfiguration) applyDefaults() {
	if c.ModelType == "" {
		c.ModelType = sdk.WorkerModelTypeMicroVM
	}
	if c.DockerBase.RootfsFormat == "" {
		c.DockerBase.RootfsFormat = "squashfs"
	}
	if len(c.OSArch) == 0 {
		c.OSArch = []string{"linux/amd64"}
	}
	if len(c.Flavors) == 0 {
		c.Flavors = slices.Clone(defaultFlavors)
	}
	if c.DefaultFlavor == "" {
		c.DefaultFlavor = "XS"
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
	if c.ExtraBootArgs == "" {
		c.ExtraBootArgs = defaultExtraBootArgs
	}
	if c.PreloadedPlugins == nil {
		c.PreloadedPlugins = []string{"script"}
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

	// The flavors are checked as the hatchery uses them, defaults included
	hconfig.applyDefaults()
	switch hconfig.ModelType {
	case sdk.WorkerModelTypeMicroVM:
	case sdk.WorkerModelTypeDocker:
		// Checked as the API checks a microvm worker model
		base := sdk.V2WorkerModel{Name: "dockerBase", OSArch: hconfig.OSArch[0], Type: sdk.WorkerModelTypeMicroVM}
		base.Spec, _ = json.Marshal(hconfig.DockerBase.spec())
		if errs := base.LintYamlDefinition(); len(errs) > 0 {
			return sdk.WithStack(fmt.Errorf("invalid dockerBase, required with modelType docker: %v", errs))
		}
	default:
		return sdk.WithStack(fmt.Errorf("invalid modelType %q: microvm or docker", hconfig.ModelType))
	}
	for i, f := range hconfig.Flavors {
		if f.Name == "" || f.VCPUs <= 0 || f.MemoryMiB <= 0 || f.DiskGiB < 0 {
			return sdk.WithStack(fmt.Errorf("invalid flavor %q: a flavor needs a name, vcpus and memoryMiB", f.Name))
		}
		for _, other := range hconfig.Flavors[:i] {
			if strings.EqualFold(other.Name, f.Name) {
				return sdk.WithStack(fmt.Errorf("flavor %q is declared twice", f.Name))
			}
		}
	}
	if hconfig.flavor(hconfig.DefaultFlavor) == nil {
		return sdk.WithStack(fmt.Errorf("defaultFlavor %q is not among the flavors %s", hconfig.DefaultFlavor, hconfig.flavorNames()))
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

// ModelType returns the configured worker model type
func (h *HatcheryMicroVM) ModelType() string {
	return h.Config.ModelType
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

// CanSpawn only accepts v2 models of the configured type whose flavor is
// configured
func (h *HatcheryMicroVM) CanSpawn(ctx context.Context, model sdk.WorkerStarterWorkerModel, jobID string, requirements []sdk.Requirement) bool {
	ctx, end := telemetry.Span(ctx, "microvm.CanSpawn")
	defer end()

	if model.ModelV2 == nil || model.ModelV2.Type != h.ModelType() {
		log.Warn(ctx, "CanSpawn> Job %s: only v2 %s worker models are supported", jobID, h.ModelType())
		return false
	}
	if !slices.Contains(h.Config.OSArch, model.ModelV2.OSArch) {
		log.Warn(ctx, "CanSpawn> Job %s with worker model %s cannot be spawned. Got osarch %s and want %s", jobID, model.ModelV2.Name, model.ModelV2.OSArch, h.Config.OSArch)
		return false
	}
	if h.ModelType() == sdk.WorkerModelTypeDocker {
		if model.DockerSpec.Image == "" {
			log.Warn(ctx, "CanSpawn> Job %s: worker model %s has no image", jobID, model.ModelV2.Name)
			return false
		}
	} else {
		spec := model.MicroVMSpec
		if spec.Kernel.URL == "" || spec.Rootfs.URL == "" {
			log.Warn(ctx, "CanSpawn> Job %s: worker model %s has no kernel or rootfs", jobID, model.ModelV2.Name)
			return false
		}
	}
	if _, err := h.Config.workerFlavor(model, requirements); err != nil {
		log.Warn(ctx, "CanSpawn> Job %s: %v", jobID, err)
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

// workerFlavor returns the flavor of a worker, the first set among: the job
// flavor (and for a microvm model, the model flavor), the smallest flavor
// holding the job memory, the default flavor. No fallback on another size:
// the job would run with less than it asked for.
func (c *HatcheryConfiguration) workerFlavor(model sdk.WorkerStarterWorkerModel, requirements []sdk.Requirement) (*FlavorConfig, error) {
	if name := model.GetFlavor(requirements, ""); name != "" {
		if f := c.flavor(name); f != nil {
			return f, nil
		}
		return nil, fmt.Errorf("unknown flavor %q, the hatchery knows %s", name, c.flavorNames())
	}
	// runs-on.memory, in MiB as the swarm hatchery reads it
	if model.Memory > 0 {
		var smallest *FlavorConfig
		for i := range c.Flavors {
			f := &c.Flavors[i]
			if int64(f.MemoryMiB) >= model.Memory && (smallest == nil || f.MemoryMiB < smallest.MemoryMiB) {
				smallest = f
			}
		}
		if smallest == nil {
			return nil, fmt.Errorf("no flavor has %d MiB of memory, the hatchery knows %s", model.Memory, c.flavorNames())
		}
		return smallest, nil
	}
	if f := c.flavor(c.DefaultFlavor); f != nil {
		return f, nil
	}
	return nil, fmt.Errorf("unknown flavor %q, the hatchery knows %s", c.DefaultFlavor, c.flavorNames())
}

// flavor returns the flavor of a name, case-insensitive, nil when unknown
func (c *HatcheryConfiguration) flavor(name string) *FlavorConfig {
	for i := range c.Flavors {
		if strings.EqualFold(c.Flavors[i].Name, name) {
			return &c.Flavors[i]
		}
	}
	return nil
}

func (c *HatcheryConfiguration) flavorNames() string {
	names := make([]string, 0, len(c.Flavors))
	for _, f := range c.Flavors {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
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
