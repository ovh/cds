package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/slug"
	"github.com/ovh/cds/sdk/telemetry"
)

// New instanciates a new hatchery local
func New() *HatcheryKubernetes {
	s := new(HatcheryKubernetes)
	s.GoRoutines = sdk.NewGoRoutines(context.Background())
	return s
}

var _ hatchery.InterfaceWithModels = new(HatcheryKubernetes)

// InitHatchery register local hatchery with its worker model
func (h *HatcheryKubernetes) InitHatchery(ctx context.Context) error {
	if err := h.Common.Init(ctx, h); err != nil {
		return err
	}
	h.GoRoutines.Run(ctx, "hatchery kubernetes routines", func(ctx context.Context) {
		h.routines(ctx)
	})

	h.GoRoutines.RunWithRestart(ctx, "hatchery kubernetes watcher", func(ctx context.Context) {
		if err := h.WatchPodEvents(ctx); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	})

	return nil
}

func (h *HatcheryKubernetes) WatchPodEvents(ctx context.Context) error {
	opts := metav1.ListOptions{
		FieldSelector: "involvedObject.kind=Pod",
		Watch:         true,
	}
	// requires "watch" permission on events in clusterrole
	watcher, err := h.kubeClient.Events(ctx, h.Config.Namespace, opts)
	if err != nil {
		return err
	}
	watchCh := watcher.ResultChan()
	defer watcher.Stop()
	for event := range watchCh {
		switch x := event.Object.(type) {
		case *corev1.Event:
			log.Info(ctx, "object: %s, reason: %s, message: %s, component: %s, host: %s", x.ObjectMeta.Name, x.Reason, x.Message, x.Source.Component, x.Source.Host)
		}
	}
	return nil
}

// Init cdsclient config.
func (h *HatcheryKubernetes) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(HatcheryConfiguration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid kubernetes hatchery configuration"))
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
func (h *HatcheryKubernetes) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return sdk.WithStack(fmt.Errorf("invalid configuration"))
	}

	var err error
	h.kubeClient, err = initKubeClient(h.Config)
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
  h.Common.Common.ModelType = h.ModelType()

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryKubernetes) Status(ctx context.Context) *sdk.MonitoringStatus {
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
func (h *HatcheryKubernetes) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return sdk.WithStack(fmt.Errorf("invalid hatchery kubernetes configuration"))
	}

	if err := hconfig.Check(); err != nil {
		return sdk.WithStack(fmt.Errorf("invalid hatchery kubernetes configuration: %v", err))
	}

	if hconfig.Namespace == "" {
		return sdk.WithStack(fmt.Errorf("missing valid kubernetes namespace"))
	}

	return nil
}

func (h *HatcheryKubernetes) Signin(ctx context.Context, clientConfig cdsclient.ServiceConfig, srvConfig interface{}) error {
	if err := h.Common.Signin(ctx, clientConfig, srvConfig); err != nil {
		return err
	}
	if err := h.Common.SigninV2(ctx, clientConfig, srvConfig); err != nil {
		return err
	}
	return nil
}

// Start inits client and routines for hatchery
func (h *HatcheryKubernetes) Start(ctx context.Context) error {
	return hatchery.Create(ctx, h)
}

// Serve start the hatchery server
func (h *HatcheryKubernetes) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

// Configuration returns Hatchery CommonConfiguration
func (h *HatcheryKubernetes) Configuration() service.HatcheryCommonConfiguration {
	return h.Config.HatcheryCommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryKubernetes) ModelType() string {
	return sdk.Docker
}

// WorkerModelsEnabled returns Worker model enabled.
func (h *HatcheryKubernetes) WorkerModelsEnabled() ([]sdk.Model, error) {
	return h.CDSClient().WorkerModelEnabledList()
}

// WorkerModelSecretList returns secret for given model.
func (h *HatcheryKubernetes) WorkerModelSecretList(m sdk.Model) (sdk.WorkerModelSecrets, error) {
	return h.CDSClient().WorkerModelSecretList(m.Group.Name, m.Name)
}

// CanSpawn return wether or not hatchery can spawn model.
// requirements are not supported
func (h *HatcheryKubernetes) CanSpawn(ctx context.Context, _ *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	// Service and Hostname requirement are not supported
	for _, r := range requirements {
		if r.Type == sdk.HostnameRequirement {
			log.Debug(ctx, "CanSpawn> Job %d has a hostname requirement. Kubernetes can't spawn a worker for this job", jobID)
			return false
		}
	}
	return true
}

// SpawnWorker starts a new worker process
func (h *HatcheryKubernetes) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	ctx, end := telemetry.Span(ctx, "HatcheryKubernetes.SpawnWorker",
		telemetry.Tag(telemetry.TagWorkflowNodeJobRun, spawnArgs.JobID),
		telemetry.Tag(telemetry.TagWorker, spawnArgs.WorkerName))
	defer end()

	if spawnArgs.JobID == 0 && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

	var logJob string
	if spawnArgs.JobID > 0 {
		logJob = fmt.Sprintf("for workflow job %d,", spawnArgs.JobID)
	}

	cpu := h.Config.DefaultCPU
	if cpu == "" {
		cpu = "500m"
	}

	memory := int64(h.Config.DefaultMemory)
	if memory == 0 {
		memory = 1024
	}
	for _, r := range spawnArgs.Requirements {
		if r.Type == sdk.MemoryRequirement {
			var err error
			memory, err = strconv.ParseInt(r.Value, 10, 64)
			if err != nil {
				log.Warn(ctx, "spawnKubernetesDockerWorker> %s unable to parse memory requirement %d: %v", logJob, memory, err)
				return err
			}
			break
		}
	}

	ephemeralStorage := h.Config.DefaultEphemeralStorage
	if ephemeralStorage == "" {
		ephemeralStorage = "1Gi"
	}

	workerConfig := h.GenerateWorkerConfig(ctx, h, spawnArgs)
	udataParam := struct {
		API string
	}{
		API: workerConfig.APIEndpoint,
	}

	tmpl, errt := template.New("cmd").Parse(spawnArgs.Model.ModelDocker.Cmd)
	if errt != nil {
		return errt
	}
	var buffer bytes.Buffer
	if errTmpl := tmpl.Execute(&buffer, udataParam); errTmpl != nil {
		return errTmpl
	}

	cmd := buffer.String()
	if spawnArgs.RegisterOnly {
		cmd += " register"
		memory = hatchery.MemoryRegisterContainer
	}

	if spawnArgs.Model.ModelDocker.Envs == nil {
		spawnArgs.Model.ModelDocker.Envs = map[string]string{}
	}
	envsWm := workerConfig.InjectEnvVars
	envsWm["CDS_MODEL_MEMORY"] = fmt.Sprintf("%d", memory)
	envsWm["CDS_FROM_WORKER_IMAGE"] = "true"

	for envName, envValue := range spawnArgs.Model.ModelDocker.Envs {
		envsWm[envName] = envValue
	}

	envs := make([]apiv1.EnvVar, len(envsWm))
	i := 0
	for envName, envValue := range envsWm {
		envs[i] = apiv1.EnvVar{Name: envName, Value: envValue}
		i++
	}

	// Create secret for worker config
	configSecretName, err := h.createConfigSecret(ctx, workerConfig)
	if err != nil {
		return sdk.WrapError(err, "cannot create secret for config %s", workerConfig.Name)
	}
	envs = append(envs, apiv1.EnvVar{
		Name: "CDS_CONFIG",
		ValueFrom: &apiv1.EnvVarSource{
			SecretKeyRef: &apiv1.SecretKeySelector{
				LocalObjectReference: apiv1.LocalObjectReference{
					Name: configSecretName,
				},
				Key: "CDS_CONFIG",
			},
		},
	})

	var limits apiv1.ResourceList
	if h.Config.DisableCPULimit {
		limits = apiv1.ResourceList{
			apiv1.ResourceMemory:           *resource.NewScaledQuantity(memory, resource.Mega),
			apiv1.ResourceEphemeralStorage: resource.MustParse(ephemeralStorage),
		}
	} else {
		limits = apiv1.ResourceList{
			apiv1.ResourceCPU:              resource.MustParse(cpu),
			apiv1.ResourceMemory:           *resource.NewScaledQuantity(memory, resource.Mega),
			apiv1.ResourceEphemeralStorage: resource.MustParse(ephemeralStorage),
		}
	}

	var gracePeriodSecs int64
	podSchema := apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       spawnArgs.WorkerName,
			Namespace:                  h.Config.Namespace,
			DeletionGracePeriodSeconds: &gracePeriodSecs,
			Labels: map[string]string{
				LABEL_HATCHERY_NAME:     h.Configuration().Name,
				LABEL_WORKER_NAME:       workerConfig.Name,
				LABEL_WORKER_MODEL_PATH: slug.Convert(spawnArgs.Model.Path()),
			},
			Annotations: map[string]string{},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy:                 apiv1.RestartPolicyNever,
			TerminationGracePeriodSeconds: &gracePeriodSecs,
			Containers: []apiv1.Container{
				{
					Name:            spawnArgs.WorkerName,
					Image:           spawnArgs.Model.ModelDocker.Image,
					ImagePullPolicy: apiv1.PullAlways,
					Env:             envs,
					Command:         strings.Fields(spawnArgs.Model.ModelDocker.Shell),
					Args:            []string{cmd},
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{
							apiv1.ResourceCPU:              resource.MustParse(cpu),
							apiv1.ResourceMemory:           *resource.NewScaledQuantity(memory, resource.Mega),
							apiv1.ResourceEphemeralStorage: resource.MustParse(ephemeralStorage),
						},
						Limits: limits,
					},
				},
			},
		},
	}

	// Set custom annotation on pod if needed
	for _, a := range h.Config.CustomAnnotations {
		if a.Key != "" && a.Value != "" {
			podSchema.Annotations[a.Key] = a.Value
		}
	}

	// Check here to add secret if needed
	if spawnArgs.Model.ModelDocker.Private || (spawnArgs.Model.ModelDocker.Username != "" && spawnArgs.Model.ModelDocker.Password != "") {
		secretRegistryName, err := h.createRegistrySecret(ctx, *spawnArgs.Model)
		if err != nil {
			return sdk.WrapError(err, "cannot create secret for model %s", spawnArgs.Model.Path())
		}
		podSchema.Spec.ImagePullSecrets = []apiv1.LocalObjectReference{{Name: secretRegistryName}}
	}

	var services []sdk.Requirement
	for _, req := range spawnArgs.Requirements {
		if req.Type == sdk.ServiceRequirement {
			services = append(services, req)
		}
	}

	if len(services) > 0 {
		podSchema.Spec.HostAliases = make([]apiv1.HostAlias, 1)
		podSchema.Spec.HostAliases[0] = apiv1.HostAlias{IP: "127.0.0.1", Hostnames: make([]string, len(services)+1)}
		podSchema.Spec.HostAliases[0].Hostnames[0] = "worker"
	}

	serviceCPU := h.Config.DefaultServiceCPU
	if serviceCPU == "" {
		serviceCPU = "256m"
	}

	serviceMemory := int64(h.Config.DefaultServiceMemory)
	if serviceMemory == 0 {
		serviceMemory = 512
	}

	serviceEphemeralStorage := h.Config.DefaultServiceEphemeralStorage
	if serviceEphemeralStorage == "" {
		serviceEphemeralStorage = "512Mi"
	}

	for i, serv := range services {
		//name= <alias> => the name of the host put in /etc/hosts of the worker
		//value= "postgres:latest env_1=blabla env_2=blabla"" => we can add env variables in requirement name
		img, envm := hatchery.ParseRequirementModel(serv.Value)

		if sm, ok := envm["CDS_SERVICE_MEMORY"]; ok {
			var err error
			serviceMemory, err = strconv.ParseInt(sm, 10, 64)
			if err != nil {
				log.Warn(ctx, "hatchery> kubernetes> SpawnWorker> Unable to parse CDS_SERVICE_MEMORY value '%s': %s", sm, err)
				continue
			}
			delete(envm, "CDS_SERVICE_MEMORY")
		}

		servContainer := apiv1.Container{
			Name:  fmt.Sprintf("service-%d-%s", serv.ID, strings.ToLower(serv.Name)),
			Image: img,
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceCPU:              resource.MustParse(serviceCPU),
					apiv1.ResourceMemory:           *resource.NewScaledQuantity(serviceMemory, resource.Mega),
					apiv1.ResourceEphemeralStorage: resource.MustParse(serviceEphemeralStorage),
				},
				Limits: apiv1.ResourceList{
					apiv1.ResourceCPU:              resource.MustParse(serviceCPU),
					apiv1.ResourceMemory:           *resource.NewScaledQuantity(serviceMemory, resource.Mega),
					apiv1.ResourceEphemeralStorage: resource.MustParse(serviceEphemeralStorage),
				},
			},
		}

		if sa, ok := envm["CDS_SERVICE_ARGS"]; ok {
			servContainer.Args = hatchery.ParseArgs(sa)
			delete(envm, "CDS_SERVICE_ARGS")
		}

		if len(envm) > 0 {
			servContainer.Env = make([]apiv1.EnvVar, 0, len(envm))
			for key, val := range envm {
				servContainer.Env = append(servContainer.Env, apiv1.EnvVar{Name: key, Value: val})
			}
		}

		podSchema.ObjectMeta.Labels[hatchery.LabelServiceJobID] = fmt.Sprintf("%d", spawnArgs.JobID)
		podSchema.ObjectMeta.Labels[hatchery.LabelServiceNodeRunID] = fmt.Sprintf("%d", spawnArgs.NodeRunID)
		podSchema.ObjectMeta.Labels[hatchery.LabelServiceProjectKey] = spawnArgs.ProjectKey
		podSchema.ObjectMeta.Labels[hatchery.LabelServiceWorkflowName] = spawnArgs.WorkflowName
		podSchema.ObjectMeta.Labels[hatchery.LabelServiceWorkflowID] = fmt.Sprintf("%d", spawnArgs.WorkflowID)
		podSchema.ObjectMeta.Labels[hatchery.LabelServiceRunID] = fmt.Sprintf("%d", spawnArgs.RunID)
		podSchema.ObjectMeta.Labels[hatchery.LabelServiceNodeRunName] = spawnArgs.NodeRunName
		podSchema.ObjectMeta.Annotations[hatchery.LabelServiceJobName] = spawnArgs.JobName

		podSchema.Spec.Containers = append(podSchema.Spec.Containers, servContainer)
		podSchema.Spec.HostAliases[0].Hostnames[i+1] = strings.ToLower(serv.Name)
	}

	_, err = h.kubeClient.PodCreate(ctx, h.Config.Namespace, &podSchema, metav1.CreateOptions{})
	log.Debug(ctx, "hatchery> kubernetes> SpawnWorker> %s > Pod created", spawnArgs.WorkerName)
	return sdk.WithStack(err)
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryKubernetes) WorkersStarted(ctx context.Context) ([]string, error) {
	list, err := h.kubeClient.PodList(ctx, h.Config.Namespace, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s", LABEL_HATCHERY_NAME, h.Config.Name, LABEL_WORKER_NAME),
	})
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list pods on namespace %s", h.Config.Namespace)
	}
	workerNames := make([]string, 0, list.Size())
	for _, pod := range list.Items {
		workerNames = append(workerNames, pod.GetName())
	}
	return workerNames, nil
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryKubernetes) NeedRegistration(_ context.Context, m *sdk.Model) bool {
	if m.NeedRegistration || m.LastRegistration.Unix() < m.UserLastModified.Unix() {
		return true
	}
	return false
}

func (h *HatcheryKubernetes) routines(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.GoRoutines.Exec(ctx, "getServicesLogs", func(ctx context.Context) {
				if err := h.getServicesLogs(ctx); err != nil {
					log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "cannot get service logs"))
				}
			})

			h.GoRoutines.Exec(ctx, "killAwolWorker", func(ctx context.Context) {
				if err := h.killAwolWorkers(ctx); err != nil {
					log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "cannot delete awol worker"))
				}
			})

			h.GoRoutines.Exec(ctx, "deleteSecrets", func(ctx context.Context) {
				if err := h.deleteSecrets(ctx); err != nil {
					log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "cannot delete secrets"))
				}
			})
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Hatchery> Kubernetes> Exiting routines")
			}
			return
		}
	}
}
