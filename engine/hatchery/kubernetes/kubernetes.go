package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
)

// New instanciates a new hatchery local
func New() *HatcheryKubernetes {
	s := new(HatcheryKubernetes)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryKubernetes) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	configK8s, err := clientcmd.BuildConfigFromKubeconfigGetter(h.Config.KubernetesMasterURL, h.getStartingConfig)
	if err != nil {
		return sdk.WrapError(err, "Cannot build config from flags")
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(configK8s)
	if err != nil {
		return sdk.WrapError(err, "Cannot create new config")
	}

	h.k8sClient = clientset

	if h.Config.KubernetesNamespace != apiv1.NamespaceDefault {
		if _, err := clientset.CoreV1().Namespaces().Get(h.Config.KubernetesNamespace, metav1.GetOptions{}); err != nil {
			ns := apiv1.Namespace{}
			ns.SetName(h.Config.KubernetesNamespace)
			if _, errC := clientset.CoreV1().Namespaces().Create(&ns); errC != nil {
				return sdk.WrapError(errC, "Cannot create namespace %s in kubernetes", h.Config.KubernetesNamespace)
			}
		}
	}

	h.hatch = &sdk.Hatchery{
		Name:    h.Configuration().Name,
		Version: sdk.VERSION,
	}

	h.Client = cdsclient.NewHatchery(
		h.Configuration().API.HTTP.URL,
		h.Configuration().API.Token,
		h.Configuration().Provision.RegisterFrequency,
		h.Configuration().API.HTTP.Insecure,
		h.hatch.Name,
	)

	h.API = h.Config.API.HTTP.URL
	h.Name = h.Config.Name
	h.HTTPURL = h.Config.URL
	h.Token = h.Config.API.Token
	h.Type = services.TypeHatchery
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryKubernetes) Status() sdk.MonitoringStatus {
	m := h.CommonMonitoring()
	if h.IsInitialized() {
		m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", h.WorkersStarted(), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})
	}
	return m
}

// getStartingConfig implements ConfigAccess
func (h *HatcheryKubernetes) getStartingConfig() (*clientcmdapi.Config, error) {
	defaultClientConfigRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrideCfg := clientcmd.ConfigOverrides{
		AuthInfo: clientcmdapi.AuthInfo{
			Username: h.Config.KubernetesUsername,
			Password: h.Config.KubernetesPassword,
			Token:    h.Config.KubernetesToken,
		},
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(defaultClientConfigRules, &overrideCfg)
	rawConfig, err := clientConfig.RawConfig()
	if os.IsNotExist(err) {
		return clientcmdapi.NewConfig(), nil
	}
	if err != nil {
		return nil, err
	}

	return &rawConfig, nil
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryKubernetes) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	if hconfig.API.HTTP.URL == "" {
		return fmt.Errorf("API HTTP(s) URL is mandatory")
	}

	if hconfig.API.Token == "" {
		return fmt.Errorf("API Token URL is mandatory")
	}

	if hconfig.Name == "" {
		return fmt.Errorf("please enter a name in your kubernetes hatchery configuration")
	}

	if hconfig.KubernetesNamespace == "" {
		return fmt.Errorf("please enter a valid kubernetes namespace")
	}

	if hconfig.KubernetesMasterURL == "" {
		return fmt.Errorf("please enter a valid kubernetes master URL")
	}

	return nil
}

// ID must returns hatchery id
func (h *HatcheryKubernetes) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

//Hatchery returns hatchery instance
func (h *HatcheryKubernetes) Hatchery() *sdk.Hatchery {
	return h.hatch
}

// Serve start the hatchery server
func (h *HatcheryKubernetes) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryKubernetes) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryKubernetes) ModelType() string {
	return sdk.Docker
}

// CanSpawn return wether or not hatchery can spawn model.
// requirements are not supported
func (h *HatcheryKubernetes) CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement {
			return false
		}
	}
	return true
}

// SpawnWorker starts a new worker process
func (h *HatcheryKubernetes) SpawnWorker(spawnArgs hatchery.SpawnArguments) (string, error) {
	name := fmt.Sprintf("k8s-%s-%s", strings.ToLower(spawnArgs.Model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1))
	label := "execution"
	if spawnArgs.RegisterOnly {
		name = "register-" + name
		label = "register"
	}

	var logJob string
	if spawnArgs.JobID > 0 {
		if spawnArgs.IsWorkflowJob {
			logJob = fmt.Sprintf("for workflow job %d,", spawnArgs.JobID)
		} else {
			logJob = fmt.Sprintf("for pipeline build job %d,", spawnArgs.JobID)
		}
	}

	memory := int64(h.Config.DefaultMemory)
	for _, r := range spawnArgs.Requirements {
		if r.Type == sdk.MemoryRequirement {
			var err error
			memory, err = strconv.ParseInt(r.Value, 10, 64)
			if err != nil {
				log.Warning("spawnKubernetesDockerWorker> %s unable to parse memory requirement %s:%s", logJob, memory, err)
				return "", err
			}
		}
	}

	udataParam := sdk.WorkerArgs{
		API:               h.Configuration().API.HTTP.URL,
		Token:             h.Configuration().API.Token,
		HTTPInsecure:      h.Config.API.HTTP.Insecure,
		Name:              name,
		Model:             spawnArgs.Model.ID,
		Hatchery:          h.hatch.ID,
		HatcheryName:      h.hatch.Name,
		TTL:               h.Config.WorkerTTL,
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
		GrpcAPI:           h.Configuration().API.GRPC.URL,
		GrpcInsecure:      h.Configuration().API.GRPC.Insecure,
	}

	if spawnArgs.IsWorkflowJob {
		udataParam.WorkflowJobID = spawnArgs.JobID
	} else {
		udataParam.PipelineBuildJobID = spawnArgs.JobID
	}

	tmpl, errt := template.New("cmd").Parse(spawnArgs.Model.ModelDocker.Cmd)
	if errt != nil {
		return "", errt
	}
	var buffer bytes.Buffer
	if errTmpl := tmpl.Execute(&buffer, udataParam); errTmpl != nil {
		return "", errTmpl
	}

	cmd := buffer.String()
	if spawnArgs.RegisterOnly {
		cmd += " register"
		memory = hatchery.MemoryRegisterContainer
	}

	if spawnArgs.Model.ModelDocker.Envs == nil {
		spawnArgs.Model.ModelDocker.Envs = map[string]string{}
	}

	envsWm, errEnv := sdk.TemplateEnvs(udataParam, spawnArgs.Model.ModelDocker.Envs)
	if errEnv != nil {
		return "", errEnv
	}

	envsWm["CDS_FORCE_EXIT"] = "1"
	envsWm["CDS_API"] = udataParam.API
	envsWm["CDS_TOKEN"] = udataParam.Token
	envsWm["CDS_NAME"] = udataParam.Name
	envsWm["CDS_MODEL"] = fmt.Sprintf("%d", udataParam.Model)
	envsWm["CDS_HATCHERY"] = fmt.Sprintf("%d", udataParam.Hatchery)
	envsWm["CDS_HATCHERY_NAME"] = udataParam.HatcheryName
	envsWm["CDS_FROM_WORKER_IMAGE"] = fmt.Sprintf("%v", udataParam.FromWorkerImage)
	envsWm["CDS_INSECURE"] = fmt.Sprintf("%v", udataParam.HTTPInsecure)

	if spawnArgs.JobID > 0 {
		if spawnArgs.IsWorkflowJob {
			envsWm["CDS_BOOKED_WORKFLOW_JOB_ID"] = fmt.Sprintf("%d", spawnArgs.JobID)
		} else {
			envsWm["CDS_BOOKED_PB_JOB_ID"] = fmt.Sprintf("%d", spawnArgs.JobID)
		}
	}

	if udataParam.GrpcAPI != "" && spawnArgs.Model.Communication == sdk.GRPC {
		envsWm["CDS_GRPC_API"] = udataParam.GrpcAPI
		envsWm["CDS_GRPC_INSECURE"] = fmt.Sprintf("%v", udataParam.GrpcInsecure)
	}

	envs := make([]apiv1.EnvVar, len(envsWm))
	i := 0
	for envName, envValue := range envsWm {
		envs[i] = apiv1.EnvVar{Name: envName, Value: envValue}
		i++
	}

	var gracePeriodSecs int64
	pod, err := h.k8sClient.CoreV1().Pods(h.Config.KubernetesNamespace).Create(&apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			DeletionGracePeriodSeconds: &gracePeriodSecs,
			Labels: map[string]string{
				LABEL_WORKER:        label,
				LABEL_WORKER_MODEL:  strings.ToLower(spawnArgs.Model.Name),
				LABEL_HATCHERY_NAME: h.Configuration().Name,
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy:                 apiv1.RestartPolicyNever,
			TerminationGracePeriodSeconds: &gracePeriodSecs,
			Containers: []apiv1.Container{
				{
					Name:    name,
					Image:   spawnArgs.Model.ModelDocker.Image,
					Env:     envs,
					Command: strings.Fields(spawnArgs.Model.ModelDocker.Shell),
					Args:    []string{cmd},
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{
							apiv1.ResourceMemory: resource.MustParse(fmt.Sprintf("%d", memory)),
						},
					},
				},
			},
		},
	})

	return pod.Name, err
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryKubernetes) WorkersStarted() int {
	workersLen := 0
	list, err := h.k8sClient.CoreV1().Pods(h.Config.KubernetesNamespace).List(metav1.ListOptions{LabelSelector: LABEL_HATCHERY_NAME})
	if err != nil {
		return workersLen
	}
	for _, pod := range list.Items {
		labels := pod.GetLabels()
		if labels[LABEL_HATCHERY_NAME] == h.Configuration().Name {
			workersLen++
		}
	}

	return workersLen
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryKubernetes) WorkersStartedByModel(model *sdk.Model) int {
	list, err := h.k8sClient.CoreV1().Pods(h.Config.KubernetesNamespace).List(metav1.ListOptions{LabelSelector: LABEL_WORKER_MODEL})
	if err != nil {
		log.Error("WorkersStartedByModel> Cannot get list of workers started (%s)", err)
		return 0
	}
	workersLen := 0
	for _, pod := range list.Items {
		labels := pod.GetLabels()
		if labels[LABEL_WORKER_MODEL] == model.Name {
			workersLen++
		}
	}

	return workersLen
}

// Init register local hatchery with its worker model
func (h *HatcheryKubernetes) Init() error {

	if err := hatchery.Register(h); err != nil {
		return fmt.Errorf("Cannot register: %s", err)
	}

	go h.startKillAwolWorkerRoutine()
	return nil
}

func (h *HatcheryKubernetes) startKillAwolWorkerRoutine() {
	for {
		time.Sleep(10 * time.Second)
		h.killAwolWorkers()
	}
}

func (h *HatcheryKubernetes) killAwolWorkers() error {
	pods, err := h.k8sClient.CoreV1().Pods(h.Config.KubernetesNamespace).List(metav1.ListOptions{LabelSelector: LABEL_WORKER})
	if err != nil {
		return err
	}

	var globalErr error
	for _, pod := range pods.Items {
		toDelete := false
		for _, container := range pod.Status.ContainerStatuses {
			if (container.State.Terminated != nil && (container.State.Terminated.Reason == "Completed" || container.State.Terminated.Reason == "Error")) ||
				(container.State.Waiting != nil && container.State.Waiting.Reason == "ErrImagePull") {
				toDelete = true
			}
		}
		if toDelete {
			if err := h.k8sClient.CoreV1().Pods(pod.Namespace).Delete(pod.Name, nil); err != nil {
				globalErr = err
				log.Error("hatchery:kubernetes> killAwolWorkers> Cannot delete pod %s (%s)", pod.Name, err)
			}
		}
	}
	return globalErr
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryKubernetes) NeedRegistration(m *sdk.Model) bool {
	if m.NeedRegistration || m.LastRegistration.Unix() < m.UserLastModified.Unix() {
		return true
	}
	return false
}
