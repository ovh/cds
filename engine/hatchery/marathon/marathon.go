package marathon

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/gambol99/go-marathon"
	"github.com/ovh/cds/sdk/namesgenerator"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// New instanciates a new Hatchery Marathon
func New() *HatcheryMarathon {
	return new(HatcheryMarathon)
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

	// h.API = h.Config.API.HTTP.URL
	// h.Name = h.Config.Name
	// h.HTTPURL = h.Config.URL
	// h.Token = h.Config.API.Token
	// h.Type = services.TypeHatchery
	// h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryMarathon) Status() sdk.MonitoringStatus {
	t := time.Now()
	m := sdk.MonitoringStatus{Now: t}
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

	if hconfig.MarathonUser == "" {
		return fmt.Errorf("Marathon User is mandatory")
	}

	if hconfig.MarathonPassword == "" {
		return fmt.Errorf("Marathon Password is mandatory")
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

	//Custom http client with 3 retries
	httpClient := &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout:  time.Minute,
			MaxTries:        3,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: hconfig.API.HTTP.Insecure},
		},
	}

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

// Serve start the HatcheryMarathon server
func (h *HatcheryMarathon) Serve(ctx context.Context) error {
	return hatchery.Create(h)
}

// ID must returns hatchery id
func (h *HatcheryMarathon) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

//Hatchery returns hatchery instance
func (h *HatcheryMarathon) Hatchery() *sdk.Hatchery {
	return h.hatch
}

//CDSClient returns cdsclient instance
func (h *HatcheryMarathon) CDSClient() cdsclient.Interface {
	return h.Client
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryMarathon) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryMarathon) ModelType() string {
	return sdk.Docker
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
	// Do not DOS marathon, if deployment queue is longer than 10
	if len(deployments) >= 10 {
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
func (h *HatcheryMarathon) SpawnWorker(spawnArgs hatchery.SpawnArguments) (string, error) {
	if spawnArgs.JobID > 0 {
		log.Debug("spawnWorker> spawning worker %s (%s) for job %d - %s", spawnArgs.Model.Name, spawnArgs.Model.Image, spawnArgs.JobID, spawnArgs.LogInfo)
	} else {
		log.Debug("spawnWorker> spawning worker %s (%s) - %s", spawnArgs.Model.Name, spawnArgs.Model.Image, spawnArgs.LogInfo)
	}

	var logJob string

	// Estimate needed memory, we will set 110% of required memory
	memory := int64(h.Config.DefaultMemory)

	cmd := "rm -f worker && curl ${CDS_API}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker"
	if spawnArgs.RegisterOnly {
		cmd += " register"
		memory = hatchery.MemoryRegisterContainer
	}

	instance := 1
	workerName := fmt.Sprintf("%s-%s", strings.ToLower(spawnArgs.Model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1))
	if spawnArgs.RegisterOnly {
		workerName = "register-" + workerName
	}
	forcePull := strings.HasSuffix(spawnArgs.Model.Image, ":latest")

	env := map[string]string{
		"CDS_API":           h.CDSClient().APIURL(),
		"CDS_TOKEN":         h.Configuration().API.Token,
		"CDS_NAME":          workerName,
		"CDS_MODEL":         fmt.Sprintf("%d", spawnArgs.Model.ID),
		"CDS_HATCHERY":      fmt.Sprintf("%d", h.hatch.ID),
		"CDS_HATCHERY_NAME": fmt.Sprintf("%s", h.hatch.Name),
		"CDS_SINGLE_USE":    "1",
		"CDS_TTL":           fmt.Sprintf("%d", h.Config.WorkerTTL),
	}

	if h.Configuration().Provision.WorkerLogsOptions.Graylog.Host != "" {
		env["CDS_GRAYLOG_HOST"] = h.Configuration().Provision.WorkerLogsOptions.Graylog.Host
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.Port > 0 {
		env["CDS_GRAYLOG_PORT"] = strconv.Itoa(h.Configuration().Provision.WorkerLogsOptions.Graylog.Port)
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey != "" {
		env["CDS_GRAYLOG_EXTRA_KEY"] = h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue != "" {
		env["CDS_GRAYLOG_EXTRA_VALUE"] = h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue
	}
	if h.Configuration().API.GRPC.URL != "" && spawnArgs.Model.Communication == sdk.GRPC {
		env["CDS_GRPC_API"] = h.Configuration().API.GRPC.URL
		env["CDS_GRPC_INSECURE"] = strconv.FormatBool(h.Configuration().API.GRPC.Insecure)
	}

	//Check if there is a memory requirement
	//if there is a service requirement: exit
	if spawnArgs.JobID > 0 {
		if spawnArgs.IsWorkflowJob {
			logJob = fmt.Sprintf("for workflow job %d,", spawnArgs.JobID)
			env["CDS_BOOKED_WORKFLOW_JOB_ID"] = fmt.Sprintf("%d", spawnArgs.JobID)
		} else {
			logJob = fmt.Sprintf("for pipeline build job %d,", spawnArgs.JobID)
			env["CDS_BOOKED_PB_JOB_ID"] = fmt.Sprintf("%d", spawnArgs.JobID)
		}

		for _, r := range spawnArgs.Requirements {
			if r.Type == sdk.MemoryRequirement {
				var err error
				memory, err = strconv.ParseInt(r.Value, 10, 64)
				if err != nil {
					log.Warning("spawnMarathonDockerWorker> %s unable to parse memory requirement %s:%s", logJob, memory, err)
					return "", err
				}
			}
		}
	}

	mem := float64(memory * 110 / 100)

	application := &marathon.Application{
		ID:  fmt.Sprintf("%s/%s", h.Config.MarathonIDPrefix, workerName),
		Cmd: &cmd,
		Container: &marathon.Container{
			Docker: &marathon.Docker{
				ForcePullImage: &forcePull,
				Image:          spawnArgs.Model.Image,
				Network:        "BRIDGE",
			},
			Type: "DOCKER",
		},
		CPUs:      0.5,
		Env:       &env,
		Instances: &instance,
		Mem:       &mem,
		Labels:    &h.marathonLabels,
	}

	if _, err := h.marathonClient.CreateApplication(application); err != nil {
		return "", err
	}

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

	deployments, err := h.marathonClient.ApplicationDeployments(application.ID)
	if err != nil {
		ticker.Stop()
		return "", fmt.Errorf("spawnMarathonDockerWorker> %s failed to list deployments: %s", logJob, err.Error())
	}

	if len(deployments) == 0 {
		ticker.Stop()
		return "", nil
	}

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
		return workerName, nil
	}

	return "", fmt.Errorf("spawnMarathonDockerWorker> %s error while deploying worker", logJob)
}

func (h *HatcheryMarathon) listApplications(idPrefix string) ([]string, error) {
	values := url.Values{}
	values.Set("embed", "apps.counts")
	values.Set("id", h.Config.MarathonIDPrefix)
	return h.marathonClient.ListApplications(values)
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryMarathon) WorkersStarted() int {
	apps, err := h.listApplications(h.Config.MarathonIDPrefix)
	if err != nil {
		log.Warning("WorkersStarted> error on list applications err:%s", err)
		return 0
	}
	return len(apps)
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

// Init only starts killing routine of worker not registered
func (h *HatcheryMarathon) Init() error {
	if err := hatchery.Register(h); err != nil {
		return fmt.Errorf("Cannot register: %s", err)
	}

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
	workers, err := h.CDSClient().WorkerList()
	if err != nil {
		return err
	}

	apps, err := h.listApplications(h.Config.MarathonIDPrefix)
	if err != nil {
		return err
	}

	for _, w := range workers {
		if w.Status != sdk.StatusDisabled {
			continue
		}

		// check that there is a worker matching
		for _, app := range apps {
			if strings.HasSuffix(app, w.Name) {
				log.Info("killing disabled worker %s", app)
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
	log.Debug("killAwolWorkers>")
	workers, err := h.CDSClient().WorkerList()
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

	log.Debug("killAwolWorkers> check %d apps", len(apps.Apps))

	var found bool
	// then for each RUNNING marathon application
	for _, app := range apps.Apps {
		log.Debug("killAwolWorkers> check app %s", app.ID)
		// Worker is deploying, leave him alone
		if app.TasksRunning == 0 {
			log.Debug("killAwolWorkers> app %s is deploying, do nothing", app.ID)
			continue
		}
		t, err := time.Parse(time.RFC3339, app.Version)
		if err != nil {
			log.Warning("killAwolWorkers> app %s - Cannot parse last update: %s", app.ID, err)
			break
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
		if !found && time.Since(t) > 1*time.Minute {
			log.Debug("killAwolWorkers> killing awol worker %s", app.ID)
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
