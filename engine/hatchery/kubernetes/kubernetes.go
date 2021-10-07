package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
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
	if err := h.Common.RefreshServiceLogger(ctx); err != nil {
		log.Error(ctx, "hatchery> kubernetes> cannot get cdn configuration : %v", err)
	}
	h.GoRoutines.Run(context.Background(), "hatchery kubernetes routines", func(ctx context.Context) {
		h.routines(ctx)
	})
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

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryKubernetes) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := h.NewMonitoringStatus()
	m.AddLine(sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(h.WorkersStarted(ctx)), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})

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

// Start inits client and routines for hatchery
func (h *HatcheryKubernetes) Start(ctx context.Context) error {
	return hatchery.Create(ctx, h)
}

// Serve start the hatchery server
func (h *HatcheryKubernetes) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

//Configuration returns Hatchery CommonConfiguration
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
	if spawnArgs.JobID == 0 && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

	label := "execution"
	if spawnArgs.RegisterOnly {
		label = "register"
	}

	var logJob string
	if spawnArgs.JobID > 0 {
		logJob = fmt.Sprintf("for workflow job %d,", spawnArgs.JobID)
	}

	memory := int64(h.Config.DefaultMemory)
	for _, r := range spawnArgs.Requirements {
		if r.Type == sdk.MemoryRequirement {
			var err error
			memory, err = strconv.ParseInt(r.Value, 10, 64)
			if err != nil {
				log.Warn(ctx, "spawnKubernetesDockerWorker> %s unable to parse memory requirement %d: %v", logJob, memory, err)
				return err
			}
		}
	}

	udataParam := h.GenerateWorkerArgs(ctx, h, spawnArgs)
	udataParam.TTL = h.Config.WorkerTTL

	udataParam.WorkflowJobID = spawnArgs.JobID

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
	envsWm := udataParam.InjectEnvVars
	envsWm["CDS_MODEL_MEMORY"] = fmt.Sprintf("%d", memory)
	envsWm["CDS_API"] = udataParam.API
	envsWm["CDS_TOKEN"] = udataParam.Token
	envsWm["CDS_NAME"] = udataParam.Name
	envsWm["CDS_MODEL_PATH"] = udataParam.Model
	envsWm["CDS_HATCHERY_NAME"] = udataParam.HatcheryName
	envsWm["CDS_FROM_WORKER_IMAGE"] = fmt.Sprintf("%v", udataParam.FromWorkerImage)
	envsWm["CDS_INSECURE"] = fmt.Sprintf("%v", udataParam.HTTPInsecure)

	if spawnArgs.JobID > 0 {
		envsWm["CDS_BOOKED_WORKFLOW_JOB_ID"] = fmt.Sprintf("%d", spawnArgs.JobID)
	}

	envTemplated, errEnv := sdk.TemplateEnvs(udataParam, spawnArgs.Model.ModelDocker.Envs)
	if errEnv != nil {
		return errEnv
	}

	for envName, envValue := range envTemplated {
		envsWm[envName] = envValue
	}

	envs := make([]apiv1.EnvVar, len(envsWm))
	i := 0
	for envName, envValue := range envsWm {
		envs[i] = apiv1.EnvVar{Name: envName, Value: envValue}
		i++
	}

	pullPolicy := "IfNotPresent"
	if strings.HasSuffix(spawnArgs.Model.ModelDocker.Image, ":latest") {
		pullPolicy = "Always"
	}

	var gracePeriodSecs int64
	podSchema := apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       spawnArgs.WorkerName,
			Namespace:                  h.Config.Namespace,
			DeletionGracePeriodSeconds: &gracePeriodSecs,
			Labels: map[string]string{
				LABEL_WORKER:        label,
				LABEL_WORKER_MODEL:  strings.ToLower(spawnArgs.Model.Name),
				LABEL_HATCHERY_NAME: h.Configuration().Name,
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
					ImagePullPolicy: apiv1.PullPolicy(pullPolicy),
					Env:             envs,
					Command:         strings.Fields(spawnArgs.Model.ModelDocker.Shell),
					Args:            []string{cmd},
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{
							apiv1.ResourceMemory: resource.MustParse(fmt.Sprintf("%d", memory)),
						},
					},
				},
			},
		},
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

	// Check here to add secret if needed
	secretName := "cds-credreg-" + spawnArgs.Model.Name
	if spawnArgs.Model.ModelDocker.Private {
		if err := h.createSecret(ctx, secretName, *spawnArgs.Model); err != nil {
			return sdk.WrapError(err, "cannot create secret for model %s", spawnArgs.Model.Path())
		}
		podSchema.Spec.ImagePullSecrets = []apiv1.LocalObjectReference{{Name: secretName}}
		podSchema.ObjectMeta.Labels[LABEL_SECRET] = secretName
	}

	for i, serv := range services {
		//name= <alias> => the name of the host put in /etc/hosts of the worker
		//value= "postgres:latest env_1=blabla env_2=blabla"" => we can add env variables in requirement name
		img, envm := hatchery.ParseRequirementModel(serv.Value)

		servContainer := apiv1.Container{
			Name:  fmt.Sprintf("service-%d-%s", serv.ID, strings.ToLower(serv.Name)),
			Image: img,
		}

		if sm, ok := envm["CDS_SERVICE_MEMORY"]; ok {
			mq, err := resource.ParseQuantity(sm)
			if err != nil {
				log.Warn(ctx, "hatchery> kubernetes> SpawnWorker> Unable to parse CDS_SERVICE_MEMORY value '%s': %s", sm, err)
				continue
			}
			servContainer.Resources = apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceMemory: mq,
				},
			}
			delete(envm, "CDS_SERVICE_MEMORY")
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

	_, err := h.kubeClient.PodCreate(ctx, h.Config.Namespace, &podSchema, metav1.CreateOptions{})
	log.Debug(ctx, "hatchery> kubernetes> SpawnWorker> %s > Pod created", spawnArgs.WorkerName)
	return sdk.WithStack(err)
}

func (h *HatcheryKubernetes) GetLogger() *logrus.Logger {
	return h.ServiceLogger
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryKubernetes) WorkersStarted(ctx context.Context) []string {
	list, err := h.kubeClient.PodList(ctx, h.Config.Namespace, metav1.ListOptions{LabelSelector: LABEL_HATCHERY_NAME})
	if err != nil {
		log.Warn(ctx, "WorkersStarted> unable to list pods on namespace %s", h.Config.Namespace)
		return nil
	}
	workerNames := make([]string, 0, list.Size())
	for _, pod := range list.Items {
		labels := pod.GetLabels()
		if labels[LABEL_HATCHERY_NAME] == h.Configuration().Name {
			workerNames = append(workerNames, pod.GetName())
		}
	}
	return workerNames
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
			h.GoRoutines.Exec(ctx, "getCDNConfiguration", func(ctx context.Context) {
				if err := h.Common.RefreshServiceLogger(ctx); err != nil {
					log.Error(ctx, "hatchery> kubernetes> cannot get cdn configuration : %v", err)
				}
			})

			h.GoRoutines.Exec(ctx, "getServicesLogs", func(ctx context.Context) {
				if err := h.getServicesLogs(ctx); err != nil {
					log.Error(ctx, "Hatchery> Kubernetes> Cannot get service logs : %v", err)
				}
			})

			h.GoRoutines.Exec(ctx, "killAwolWorker", func(ctx context.Context) {
				_ = h.killAwolWorkers(ctx)
			})

			h.GoRoutines.Exec(ctx, "deleteSecrets", func(ctx context.Context) {
				if err := h.deleteSecrets(ctx); err != nil {
					log.Error(ctx, "hatchery> kubernetes> cannot handle secrets : %v", err)
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
