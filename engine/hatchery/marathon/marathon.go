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

	"github.com/dgrijalva/jwt-go"
	"github.com/gambol99/go-marathon"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

// New instanciates a new Hatchery Marathon
func New() *HatcheryMarathon {
	s := new(HatcheryMarathon)
	s.GoRoutines = sdk.NewGoRoutines()
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

var _ hatchery.InterfaceWithModels = new(HatcheryMarathon)

// GetLogger return the logger
func (h *HatcheryMarathon) GetLogger() *logrus.Logger {
	return h.ServiceLogger
}

// Init cdsclient config.
func (h *HatcheryMarathon) Init(config interface{}) (cdsclient.ServiceConfig, error) {
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
func (h *HatcheryMarathon) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := h.NewMonitoringStatus()
	m.AddLine(sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(h.WorkersStarted(ctx)), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})
	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryMarathon) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid hatchery marathon configuration")
	}

	if err := hconfig.Check(); err != nil {
		return fmt.Errorf("Invalid marathon configuration: %v", err)
	}

	if hconfig.MarathonURL == "" {
		return fmt.Errorf("Marathon URL is mandatory")
	}

	if hconfig.MarathonIDPrefix == "" {
		return fmt.Errorf("Marathon ID Prefix is mandatory")
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

// Serve start the hatchery server
func (h *HatcheryMarathon) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryMarathon) Configuration() service.HatcheryCommonConfiguration {
	return h.Config.HatcheryCommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryMarathon) ModelType() string {
	return sdk.Docker
}

// WorkerModelsEnabled returns Worker model enabled.
func (h *HatcheryMarathon) WorkerModelsEnabled() ([]sdk.Model, error) {
	return h.CDSClient().WorkerModelEnabledList()
}

// WorkerModelSecretList returns secret for given model.
func (h *HatcheryMarathon) WorkerModelSecretList(m sdk.Model) (sdk.WorkerModelSecrets, error) {
	return h.CDSClient().WorkerModelSecretList(m.Group.Name, m.Name)
}

// CanSpawn return wether or not hatchery can spawn model
// requirements services are not supported
func (h *HatcheryMarathon) CanSpawn(ctx context.Context, model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	// Service and Hostname requirement are not supported
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement {
			log.Debug("CanSpawn> Job %d has a service requirement. Marathon can't spawn a worker for this job", jobID)
			return false
		} else if r.Type == sdk.HostnameRequirement {
			log.Debug("CanSpawn> Job %d has a hostname requirement. Marathon can't spawn a worker for this job", jobID)
			return false
		}
	}

	deployments, errd := h.marathonClient.Deployments()
	if errd != nil {
		log.Info(ctx, "CanSpawn> Error on h.marathonClient.Deployments() : %s", errd)
		return false
	}
	// Do not DOS marathon, if deployment queue is longer than MaxConcurrentProvisioning (default 10)
	maxProvisionning := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProvisionning == 0 {
		maxProvisionning = 10
	}

	if len(deployments) >= maxProvisionning {
		log.Info(ctx, "CanSpawn> %d item in deployment queue, waiting", len(deployments))
		return false
	}

	apps, err := h.listApplications(h.Config.MarathonIDPrefix)
	if err != nil {
		log.Info(ctx, "CanSpawn> Error on m.listApplications() : %s", errd)
		return false
	}
	if len(apps) >= h.Configuration().Provision.MaxWorker {
		log.Info(ctx, "CanSpawn> max number of containers reached, aborting. Current: %d. Max: %d", len(apps), h.Configuration().Provision.MaxWorker)
		return false
	}

	return true
}

// SpawnWorker creates an application on mesos via marathon
// requirements services are not supported
func (h *HatcheryMarathon) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	ctx, end := telemetry.Span(ctx, "hatcheryMarathon.SpawnWorker")
	defer end()

	if spawnArgs.JobID > 0 {
		log.Debug("spawnWorker> spawning worker %s (%s) for job %d", spawnArgs.Model.Name, spawnArgs.Model.ModelDocker.Image, spawnArgs.JobID)
	} else {
		log.Debug("spawnWorker> spawning worker %s (%s)", spawnArgs.Model.Name, spawnArgs.Model.ModelDocker.Image)
	}

	if spawnArgs.JobID == 0 && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

	// Estimate needed memory, we will set 110% of required memory
	memory := int64(h.Config.DefaultMemory)

	instance := 1
	forcePull := strings.HasSuffix(spawnArgs.Model.ModelDocker.Image, ":latest")

	udataParam := sdk.WorkerArgs{
		API:               h.Configuration().API.HTTP.URL,
		Token:             spawnArgs.WorkerToken,
		HTTPInsecure:      h.Config.API.HTTP.Insecure,
		Name:              spawnArgs.WorkerName,
		TTL:               h.Config.WorkerTTL,
		Model:             spawnArgs.Model.Path(),
		HatcheryName:      h.Name(),
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
		WorkflowJobID:     spawnArgs.JobID,
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

	cmd += "; sleep 120" // sleep 2min, to let marathon hatchery remove the container

	//Check if there is a memory requirement
	//if there is a service requirement: exit
	if spawnArgs.JobID > 0 {
		for _, r := range spawnArgs.Requirements {
			if r.Type == sdk.MemoryRequirement {
				var err error
				memory, err = strconv.ParseInt(r.Value, 10, 64)
				if err != nil {
					log.Warning(ctx, "spawnMarathonDockerWorker> unable to parse memory requirement %d: %v", memory, err)
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
		ID:  fmt.Sprintf("%s/%s", h.Config.MarathonIDPrefix, spawnArgs.WorkerName),
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
		CPUs:      h.Config.DefaultCPUs,
		Instances: &instance,
		Mem:       &mem,
		Labels:    &h.marathonLabels,
	}

	_, next := telemetry.Span(ctx, "marathonClient.CreateApplication")
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
				log.Debug("spawnMarathonDockerWorker> worker %s spawning in progress [%d seconds] please wait...", application.ID, int(delta))
			case <-stop:
				return
			}
		}
	}()

	log.Debug("spawnMarathonDockerWorker> worker %s spawning in progress, please wait...", application.ID)
	_, next = telemetry.Span(ctx, "marathonClient.ApplicationDeployments")
	deployments, err := h.marathonClient.ApplicationDeployments(application.ID)
	next()
	if err != nil {
		ticker.Stop()
		return fmt.Errorf("spawnMarathonDockerWorker> failed to list deployments: %s", err.Error())
	}

	if len(deployments) == 0 {
		ticker.Stop()
		return nil
	}

	_, next = telemetry.Span(ctx, "waitDeployment")
	wg := &sync.WaitGroup{}
	var errorsChan = make(chan error, len(deployments))

	for _, deploy := range deployments {
		wg.Add(1)
		go func(id string) {
			goCtx, cncl := context.WithTimeout(ctx, (time.Duration(h.Config.WorkerSpawnTimeout)+1)*time.Second)
			tick := time.NewTicker(500 * time.Millisecond)
			defer func() {
				close(errorsChan)
				cncl()
				tick.Stop()
				wg.Done()

			}()
			for {
				select {
				case _ = <-goCtx.Done():
					if _, err := h.marathonClient.DeleteDeployment(id, true); err != nil {
						errorsChan <- fmt.Errorf("error on delete timeouted deployment %s: %v", id, err.Error())
					} else {
						errorsChan <- fmt.Errorf("deployment for %s timeout", id)
					}
					return
				case _ = <-tick.C:
					found, err := h.marathonClient.HasDeployment(id)
					if err != nil {
						errorsChan <- fmt.Errorf("error on deployment %s: %s", id, err.Error())
					}
					if !found {
						log.Debug("spawnMarathonDockerWorker> deployment %s succeeded", id)
						errorsChan <- nil
						return
					}
				}
			}
		}(deploy.DeploymentID)
	}

	wg.Wait()
	next()

	errors := sdk.MultiError{}
	for b := range errorsChan {
		if b != nil {
			errors.Append(b)
		}
	}
	if errors.IsEmpty() {
		return nil
	}

	return fmt.Errorf("spawnMarathonDockerWorker> %s", errors.Error())
}

func (h *HatcheryMarathon) listApplications(idPrefix string) ([]string, error) {
	values := url.Values{}
	values.Set("embed", "apps.counts")
	values.Set("id", h.Config.MarathonIDPrefix)
	return h.marathonClient.ListApplications(values)
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryMarathon) WorkersStarted(ctx context.Context) []string {
	apps, err := h.listApplications(h.Config.MarathonIDPrefix)
	if err != nil {
		log.Warning(ctx, "WorkersStarted> error on list applications err:%s", err)
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
func (h *HatcheryMarathon) WorkersStartedByModel(ctx context.Context, model *sdk.Model) int {
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
func (h *HatcheryMarathon) InitHatchery(ctx context.Context) error {
	if err := h.RefreshServiceLogger(ctx); err != nil {
		log.Error(ctx, "Hatchery> marathon> Cannot get cdn configuration : %v", err)
	}
	h.GoRoutines.Run(ctx, "marathon-routines", func(ctx context.Context) {
		h.routines(ctx)
	})
	return nil
}

func (h *HatcheryMarathon) routines(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.GoRoutines.Exec(ctx, "marathon-killDisabledWorker", func(ctx context.Context) {
				if err := h.killDisabledWorkers(); err != nil {
					log.Warning(context.Background(), "Cannot kill disabled workers: %s", err)
				}
			})
			h.GoRoutines.Exec(ctx, "marathon-killAwolWorkers", func(ctx context.Context) {
				if err := h.killAwolWorkers(); err != nil {
					log.Warning(context.Background(), "Cannot kill awol workers: %s", err)
				}
			})
			h.GoRoutines.Exec(ctx, "marathon-refreshCDNConfiguration", func(ctx context.Context) {
				if err := h.RefreshServiceLogger(ctx); err != nil {
					log.Error(ctx, "Hatchery> marathon> Cannot get cdn configuration : %v", err)
				}
			})
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Hatchery> marathon> Exiting routines")
			}
			return
		}
	}

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
				log.Info(ctx, "killing disabled worker %s id:%s wk:%d ak:%d", app, w.ID, wk, ak)
				if _, err := h.marathonClient.DeleteApplication(app, true); err != nil {
					log.Warning(ctx, "killDisabledWorkers> Error while delete app %s err:%s", app, err)
				}
				break
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
			log.Warning(ctx, "killAwolWorkers> app %s - Cannot parse last update: %s", app.ID, err)
			break
		}

		// We let 2 minutes to worker to start and 5 minutes to a worker to register
		var maxDeploymentDuration = time.Duration(2) * time.Minute
		if strings.HasPrefix(app.ID, "register-") {
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
			if strings.HasPrefix(app.ID, "register-") && app.Env != nil {
				model := (*app.Env)["CDS_MODEL_PATH"]
				if err := hatchery.CheckWorkerModelRegister(h, model); err != nil {
					var spawnErr = sdk.SpawnErrorForm{
						Error: err.Error(),
					}
					tuple := strings.SplitN(model, "/", 2)
					if err := h.CDSClient().WorkerModelSpawnError(tuple[0], tuple[1], spawnErr); err != nil {
						log.Error(ctx, "killAndRemove> error on call client.WorkerModelSpawnError on worker model %s for register: %s", model, err)
					}
				}
			}
			if _, err := h.marathonClient.DeleteApplication(app.ID, true); err != nil {
				log.Warning(ctx, "killAwolWorkers> Error while delete app %s err:%s", app.ID, err)
				// continue to next app
			}
		}
	}

	return nil
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryMarathon) NeedRegistration(ctx context.Context, wm *sdk.Model) bool {
	if wm.NeedRegistration || wm.LastRegistration.Unix() < wm.UserLastModified.Unix() {
		return true
	}
	return false
}
