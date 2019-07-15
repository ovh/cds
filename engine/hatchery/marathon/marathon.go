package marathon

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gambol99/go-marathon"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
)

// New instanciates a new Hatchery Marathon
func New() *HatcheryMarathon {
	s := new(HatcheryMarathon)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

func (s *HatcheryMarathon) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(HatcheryConfiguration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid marathon hatchery configuration"))
	}

	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryMarathon) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	h.hatch = &sdk.Hatchery{}
	h.Name = h.Config.Name
	h.HTTPURL = h.Config.URL

	h.Type = services.TypeHatchery
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	h.Common.Common.ServiceName = "cds-hatchery-marathon"
	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryMarathon) Status() sdk.MonitoringStatus {
	m := h.CommonMonitoring()
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(h.WorkersStarted()), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})

	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryMarathon) CheckConfiguration(cfg interface{}) error {
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

	if hconfig.MarathonURL == "" {
		return fmt.Errorf("Marathon URL is mandatory")
	}

	if hconfig.MarathonIDPrefix == "" {
		return fmt.Errorf("Marathon ID Prefix is mandatory")
	}

	if hconfig.Name == "" {
		return fmt.Errorf("please enter a name in your marathon hatchery configuration")
	}

	h.marathonLabels = map[string]string{}
	if hconfig.MarathonLabels != "" {
		array := strings.Split(hconfig.MarathonLabels, ",")
		for _, s := range array {
			if !strings.Contains(s, "=") {
				continue
			}
			tuple := strings.Split(s, "=")
			if len(tuple) != 2 {
				return fmt.Errorf("malformatted configuration Marathon Labels")
			}
			h.marathonLabels[tuple[0]] = tuple[1]
		}
	}

	httpClient := cdsclient.NewHTTPClient(time.Minute, hconfig.API.HTTP.Insecure)

	config := marathon.NewDefaultConfig()
	config.URL = hconfig.MarathonURL
	config.HTTPBasicAuthUser = hconfig.MarathonUser
	config.HTTPBasicPassword = hconfig.MarathonPassword
	config.HTTPClient = httpClient

	marathonClient, err := marathon.NewClient(config)
	if err != nil {
		return fmt.Errorf("Connection failed on %s", h.Config.MarathonURL)
	}

	h.marathonClient = marathonClient
	return nil
}

//Hatchery returns hatchery instance
func (h *HatcheryMarathon) Hatchery() *sdk.Hatchery {
	return h.hatch
}

// Serve start the hatchery server
func (h *HatcheryMarathon) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryMarathon) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryMarathon) ModelType() string {
	return sdk.Docker
}

// WorkerModelsEnabled returns Worker model enabled
func (h *HatcheryMarathon) WorkerModelsEnabled() ([]sdk.Model, error) {
	return h.CDSClient().WorkerModelsEnabled()
}

// CanSpawn return wether or not hatchery can spawn model
// requirements services are not supported
func (h *HatcheryMarathon) CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	//Service requirement are not supported
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement {
			log.Debug("CanSpawn> Job %d has a service requirement. Marathon can't spawn a worker for this job", jobID)
			return false
		}
	}

	deployments, errd := h.marathonClient.Deployments()
	if errd != nil {
		log.Info("CanSpawn> Error on h.marathonClient.Deployments() : %s", errd)
		return false
	}
	// Do not DOS marathon, if deployment queue is longer than MaxConcurrentProvisioning (default 10)
	maxProvisionning := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProvisionning == 0 {
		maxProvisionning = 10
	}

	if len(deployments) >= maxProvisionning {
		log.Info("CanSpawn> %d item in deployment queue, waiting", len(deployments))
		return false
	}

	apps, err := h.listApplications(h.Config.MarathonIDPrefix)
	if err != nil {
		log.Info("CanSpawn> Error on m.listApplications() : %s", errd)
		return false
	}
	if len(apps) >= h.Configuration().Provision.MaxWorker {
		log.Info("CanSpawn> max number of containers reached, aborting. Current: %d. Max: %d", len(apps), h.Configuration().Provision.MaxWorker)
		return false
	}

	return true
}

// SpawnWorker creates an application on mesos via marathon
// requirements services are not supported
func (h *HatcheryMarathon) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	ctx, end := observability.Span(ctx, "hatcheryMarathon.SpawnWorker")
	defer end()

	if spawnArgs.JobID > 0 {
		log.Debug("spawnWorker> spawning worker %s (%s) for job %d", spawnArgs.Model.Name, spawnArgs.Model.ModelDocker.Image, spawnArgs.JobID)
	} else {
		log.Debug("spawnWorker> spawning worker %s (%s)", spawnArgs.Model.Name, spawnArgs.Model.ModelDocker.Image)
	}

	var logJob string

	// Estimate needed memory, we will set 110% of required memory
	memory := int64(h.Config.DefaultMemory)

	instance := 1
	workerName := fmt.Sprintf("%s-%s", strings.ToLower(spawnArgs.Model.Name), strings.Replace(namesgenerator.GetRandomNameCDS(0), "_", "-", -1))
	if spawnArgs.RegisterOnly {
		workerName = "register-" + workerName
	}
	forcePull := strings.HasSuffix(spawnArgs.Model.ModelDocker.Image, ":latest")

	udataParam := sdk.WorkerArgs{
		API:               h.Configuration().API.HTTP.URL,
		Token:             h.Config.API.Token,
		HTTPInsecure:      h.Config.API.HTTP.Insecure,
		Name:              workerName,
		TTL:               h.Config.WorkerTTL,
		Model:             spawnArgs.Model.GetPath(spawnArgs.Model.Group.Name),
		HatcheryName:      h.Name,
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
	}

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

	//Check if there is a memory requirement
	//if there is a service requirement: exit
	if spawnArgs.JobID > 0 {
		for _, r := range spawnArgs.Requirements {
			if r.Type == sdk.MemoryRequirement {
				var err error
				memory, err = strconv.ParseInt(r.Value, 10, 64)
				if err != nil {
					log.Warning("spawnMarathonDockerWorker> %s unable to parse memory requirement %d: %v", logJob, memory, err)
					return err
				}
			}
		}
	}

	mem := float64(memory * 110 / 100)

	if spawnArgs.Model.ModelDocker.Envs == nil {
		spawnArgs.Model.ModelDocker.Envs = map[string]string{}
	}

	envsWm := map[string]string{}
	envsWm["CDS_FORCE_EXIT"] = "0"
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

	application := &marathon.Application{
		ID:  fmt.Sprintf("%s/%s", h.Config.MarathonIDPrefix, workerName),
		Cmd: &cmd,
		Container: &marathon.Container{
			Docker: &marathon.Docker{
				ForcePullImage: &forcePull,
				Image:          spawnArgs.Model.ModelDocker.Image,
				Network:        "BRIDGE",
			},
			Type: "DOCKER",
		},
		Env:       &envsWm,
		CPUs:      0.5,
		Instances: &instance,
		Mem:       &mem,
		Labels:    &h.marathonLabels,
	}

	_, next := observability.Span(ctx, "marathonClient.CreateApplication")
	if _, err := h.marathonClient.CreateApplication(application); err != nil {
		next()
		return err
	}
	next()

	ticker := time.NewTicker(time.Second * 5)
	// ticker.Stop -> do not close goroutine..., so
	// if we range t := ticker.C --> leak goroutine
	stop := make(chan bool, 1)
	defer func() {
		stop <- true
		ticker.Stop()
	}()
	go func() {
		t0 := time.Now()
		for {
			select {
			case t := <-ticker.C:
				delta := math.Floor(t.Sub(t0).Seconds())
				log.Debug("spawnMarathonDockerWorker> %s worker %s spawning in progress [%d seconds] please wait...", logJob, application.ID, int(delta))
			case <-stop:
				return
			}
		}
	}()

	log.Debug("spawnMarathonDockerWorker> %s worker %s spawning in progress, please wait...", logJob, application.ID)
	_, next = observability.Span(ctx, "marathonClient.ApplicationDeployments")
	deployments, err := h.marathonClient.ApplicationDeployments(application.ID)
	next()
	if err != nil {
		ticker.Stop()
		return fmt.Errorf("spawnMarathonDockerWorker> %s failed to list deployments: %s", logJob, err.Error())
	}

	if len(deployments) == 0 {
		ticker.Stop()
		return nil
	}

	_, next = observability.Span(ctx, "waitDeployment")
	wg := &sync.WaitGroup{}
	var done bool
	var successChan = make(chan bool, len(deployments))
	for _, deploy := range deployments {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			go func() {
				time.Sleep((time.Duration(h.Config.WorkerSpawnTimeout) + 1) * time.Second)
				if done {
					return
				}
				// try to delete deployment
				log.Debug("spawnMarathonDockerWorker> %s timeout (%d) on deployment %s", logJob, h.Config.WorkerSpawnTimeout, id)
				if _, err := h.marathonClient.DeleteDeployment(id, true); err != nil {
					log.Warning("spawnMarathonDockerWorker> %s error on delete timeouted deployment %s: %s", logJob, id, err.Error())
				}
				successChan <- false
				wg.Done()
			}()

			if err := h.marathonClient.WaitOnDeployment(id, time.Duration(h.Config.WorkerSpawnTimeout)*time.Second); err != nil {
				log.Warning("spawnMarathonDockerWorker> %s error on deployment %s: %s", logJob, id, err.Error())
				successChan <- false
				return
			}

			log.Debug("spawnMarathonDockerWorker> %s deployment %s succeeded", logJob, id)
			successChan <- true
		}(deploy.DeploymentID)
	}

	wg.Wait()
	next()

	var success = true
	for b := range successChan {
		success = success && b
		if len(successChan) == 0 {
			break
		}
	}
	close(successChan)
	done = true

	if success {
		return nil
	}

	return fmt.Errorf("spawnMarathonDockerWorker> %s error while deploying worker", logJob)
}

func (h *HatcheryMarathon) listApplications(idPrefix string) ([]string, error) {
	values := url.Values{}
	values.Set("embed", "apps.counts")
	values.Set("id", h.Config.MarathonIDPrefix)
	return h.marathonClient.ListApplications(values)
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryMarathon) WorkersStarted() []string {
	apps, err := h.listApplications(h.Config.MarathonIDPrefix)
	if err != nil {
		log.Warning("WorkersStarted> error on list applications err:%s", err)
		return nil
	}
	res := make([]string, len(apps))
	for i, s := range apps {
		res[i] = strings.Replace(s, h.Config.MarathonIDPrefix, "", 1)
		res[i] = strings.TrimPrefix(res[i], "/")
	}
	return res
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryMarathon) WorkersStartedByModel(model *sdk.Model) int {
	apps, err := h.listApplications(h.Config.MarathonIDPrefix)
	if err != nil {
		return 0
	}

	var x int
	for _, app := range apps {
		if strings.Contains(app, strings.ToLower(model.Name)) {
			x++
		}
	}

	return x
}

// InitHatchery only starts killing routine of worker not registered
func (h *HatcheryMarathon) InitHatchery() error {
	h.startKillAwolWorkerRoutine()
	return nil
}

func (h *HatcheryMarathon) startKillAwolWorkerRoutine() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			if err := h.killDisabledWorkers(); err != nil {
				log.Warning("Cannot kill disabled workers: %s", err)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			if err := h.killAwolWorkers(); err != nil {
				log.Warning("Cannot kill awol workers: %s", err)
			}
		}
	}()
}

func (h *HatcheryMarathon) killDisabledWorkers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	workers, err := h.CDSClient().WorkerList(ctx)
	if err != nil {
		return err
	}

	apps, err := h.listApplications(h.Config.MarathonIDPrefix)
	if err != nil {
		return err
	}

	for wk, w := range workers {
		if w.Status != sdk.StatusDisabled {
			continue
		}

		// check that there is a worker matching
		for ak, app := range apps {
			if strings.HasSuffix(app, w.Name) {
				log.Info("killing disabled worker %s id:%s wk:%d ak:%d", app, w.ID, wk, ak)
				if _, err := h.marathonClient.DeleteApplication(app, true); err != nil {
					log.Warning("killDisabledWorkers> Error while delete app %s err:%s", app, err)
					// continue to next app
				}
			}
		}
	}

	return nil
}

func (h *HatcheryMarathon) killAwolWorkers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	workers, err := h.CDSClient().WorkerList(ctx)
	if err != nil {
		return err
	}

	values := url.Values{}
	values.Set("embed", "apps.counts")
	values.Set("id", h.Config.MarathonIDPrefix)

	apps, err := h.marathonClient.Applications(values)
	if err != nil {
		return err
	}

	var found bool
	// then for each RUNNING marathon application
	for _, app := range apps.Apps {
		log.Debug("killAwolWorkers> check app %s", app.ID)

		t, err := time.Parse(time.RFC3339, app.Version)
		if err != nil {
			log.Warning("killAwolWorkers> app %s - Cannot parse last update: %s", app.ID, err)
			break
		}

		// We let 2 minutes to worker to start and 5 minutes to a worker to register
		var maxDeploymentDuration = time.Duration(2) * time.Minute
		if strings.Contains(app.ID, "register-") {
			maxDeploymentDuration = time.Duration(5) * time.Minute
		}

		// check that there is a worker matching
		found = false
		for _, w := range workers {
			if strings.HasSuffix(app.ID, w.Name) && w.Status != sdk.StatusDisabled {
				found = true
				log.Debug("killAwolWorkers> apps %s is found on workers list with status %s", app.ID, w.Status)
				break
			}
		}

		// then if it's not found, kill it !
		if !found && time.Since(t) > maxDeploymentDuration {
			log.Debug("killAwolWorkers> killing awol worker %s", app.ID)
			// If its a worker "register", check registration before deleting it
			if strings.Contains(app.ID, "register-") && app.Env != nil {
				model := (*app.Env)["CDS_MODEL_PATH"]
				if err := hatchery.CheckWorkerModelRegister(h, model); err != nil {
					var spawnErr = sdk.SpawnErrorForm{
						Error: err.Error(),
					}
					tuple := strings.SplitN(model, "/", 2)
					if err := h.CDSClient().WorkerModelSpawnError(tuple[0], tuple[1], spawnErr); err != nil {
						log.Error("killAndRemove> error on call client.WorkerModelSpawnError on worker model %s for register: %s", model, err)
					}
				}
			}
			if _, err := h.marathonClient.DeleteApplication(app.ID, true); err != nil {
				log.Warning("killAwolWorkers> Error while delete app %s err:%s", app.ID, err)
				// continue to next app
			}
		}
	}

	return nil
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryMarathon) NeedRegistration(wm *sdk.Model) bool {
	if wm.NeedRegistration || wm.LastRegistration.Unix() < wm.UserLastModified.Unix() {
		return true
	}
	return false
}
