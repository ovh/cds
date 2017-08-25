package marathon

import (
	"fmt"
	"math"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/gambol99/go-marathon"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

var hatcheryMarathon *HatcheryMarathon

// HatcheryMarathon implements HatcheryMode interface for mesos mode
type HatcheryMarathon struct {
	hatch *sdk.Hatchery
	token string

	marathonClient marathon.Marathon
	client         cdsclient.Interface

	marathonHost     string
	marathonUser     string
	marathonPassword string

	marathonID           string
	marathonLabelsString string
	marathonLabels       map[string]string

	defaultMemory      int
	workerTTL          int
	workerSpawnTimeout int
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

//Client returns cdsclient instance
func (h *HatcheryMarathon) Client() cdsclient.Interface {
	return h.client
}

// KillWorker deletes an application on mesos via marathon
func (h *HatcheryMarathon) KillWorker(worker sdk.Worker) error {
	appID := path.Join(hatcheryMarathon.marathonID, worker.Name)
	log.Info("KillWorker> Killing %s", appID)

	_, err := h.marathonClient.DeleteApplication(appID, true)
	return err
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

	apps, err := h.listApplications(h.marathonID)
	if err != nil {
		log.Info("CanSpawn> Error on m.listApplications() : %s", errd)
		return false
	}
	if len(apps) >= viper.GetInt("max-worker") {
		log.Info("CanSpawn> max number of containers reached, aborting. Current: %d. Max: %d", len(apps), viper.GetInt("max-worker"))
		return false
	}

	return true
}

// SpawnWorker creates an application on mesos via marathon
// requirements services are not supported
func (h *HatcheryMarathon) SpawnWorker(model *sdk.Model, jobID int64, requirements []sdk.Requirement, registerOnly bool, logInfo string) (string, error) {
	if jobID > 0 {
		log.Info("spawnWorker> spawning worker %s (%s) for job %d - %s", model.Name, model.Image, jobID, logInfo)
	} else {
		log.Info("spawnWorker> spawning worker %s (%s) - %s", model.Name, model.Image, logInfo)
	}

	var logJob string

	// Estimate needed memory, we will set 110% of required memory
	memory := h.defaultMemory

	cmd := "rm -f worker && curl ${CDS_API}/download/worker/$(uname -m) -o worker &&  chmod +x worker && exec ./worker"
	if registerOnly {
		cmd += " register"
	}
	instance := 1
	workerName := fmt.Sprintf("%s-%s", strings.ToLower(model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1))
	if registerOnly {
		workerName = "register-" + workerName
	}
	forcePull := strings.HasSuffix(model.Image, ":latest")

	env := map[string]string{
		"CDS_API":           sdk.Host,
		"CDS_TOKEN":         h.token,
		"CDS_NAME":          workerName,
		"CDS_MODEL":         fmt.Sprintf("%d", model.ID),
		"CDS_HATCHERY":      fmt.Sprintf("%d", h.hatch.ID),
		"CDS_HATCHERY_NAME": fmt.Sprintf("%s", h.hatch.Name),
		"CDS_SINGLE_USE":    "1",
		"CDS_TTL":           fmt.Sprintf("%d", h.workerTTL),
	}

	if viper.GetString("worker_graylog_host") != "" {
		env["CDS_GRAYLOG_HOST"] = viper.GetString("worker_graylog_host")
	}
	if viper.GetString("worker_graylog_port") != "" {
		env["CDS_GRAYLOG_PORT"] = viper.GetString("worker_graylog_port")
	}
	if viper.GetString("worker_graylog_extra_key") != "" {
		env["CDS_GRAYLOG_EXTRA_KEY"] = viper.GetString("worker_graylog_extra_key")
	}
	if viper.GetString("worker_graylog_extra_value") != "" {
		env["CDS_GRAYLOG_EXTRA_VALUE"] = viper.GetString("worker_graylog_extra_value")
	}
	if viper.GetString("grpc_api") != "" && model.Communication == sdk.GRPC {
		env["CDS_GRPC_API"] = viper.GetString("grpc_api")
		env["CDS_GRPC_INSECURE"] = strconv.FormatBool(viper.GetBool("grpc_insecure"))
	}

	//Check if there is a memory requirement
	//if there is a service requirement: exit
	if jobID > 0 {
		logJob = fmt.Sprintf("for job %d,", jobID)
		env["CDS_BOOKED_JOB_ID"] = fmt.Sprintf("%d", jobID)

		for _, r := range requirements {
			if r.Type == sdk.MemoryRequirement {
				var err error
				memory, err = strconv.Atoi(r.Value)
				if err != nil {
					log.Warning("spawnMarathonDockerWorker> %s unable to parse memory requirement %s:%s", logJob, memory, err)
					return "", err
				}
			}
		}
	}

	mem := float64(memory * 110 / 100)

	application := &marathon.Application{
		ID:  fmt.Sprintf("%s/%s", h.marathonID, workerName),
		Cmd: &cmd,
		Container: &marathon.Container{
			Docker: &marathon.Docker{
				ForcePullImage: &forcePull,
				Image:          model.Image,
				Network:        "BRIDGE",
			},
			Type: "DOCKER",
		},
		CPUs:      0.5,
		Env:       &env,
		Instances: &instance,
		Mem:       &mem,
		Labels:    &hatcheryMarathon.marathonLabels,
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
				time.Sleep((time.Duration(h.workerSpawnTimeout) + 1) * time.Second)
				if done {
					return
				}
				// try to delete deployment
				log.Debug("spawnMarathonDockerWorker> %s timeout (%d) on deployment %s", logJob, h.workerSpawnTimeout, id)
				if _, err := h.marathonClient.DeleteDeployment(id, true); err != nil {
					log.Warning("spawnMarathonDockerWorker> %s error on delete timeouted deployment %s: %s", logJob, id, err.Error())
				}
				successChan <- false
				wg.Done()
			}()

			if err := h.marathonClient.WaitOnDeployment(id, time.Duration(h.workerSpawnTimeout)*time.Second); err != nil {
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
	values.Set("id", hatcheryMarathon.marathonID)
	return h.marathonClient.ListApplications(values)
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryMarathon) WorkersStarted() int {
	apps, err := h.listApplications(hatcheryMarathon.marathonID)
	if err != nil {
		log.Warning("WorkersStarted> error on list applications err:%s", err)
		return 0
	}
	return len(apps)
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryMarathon) WorkersStartedByModel(model *sdk.Model) int {
	apps, err := h.listApplications(hatcheryMarathon.marathonID)
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
func (h *HatcheryMarathon) Init(name, api, token string, requestSecondsTimeout int, insecureSkipVerifyTLS bool) error {
	sdk.Options(api, "", "", token)

	h.hatch = &sdk.Hatchery{
		Name: hatchery.GenerateName("marathon", name),
	}

	h.client = cdsclient.NewHatchery(api, token, requestSecondsTimeout, insecureSkipVerifyTLS)
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
	workers, err := sdk.GetWorkers()
	if err != nil {
		return err
	}

	apps, err := h.listApplications(hatcheryMarathon.marathonID)
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
	workers, err := sdk.GetWorkers()
	if err != nil {
		return err
	}

	values := url.Values{}
	values.Set("embed", "apps.counts")
	values.Set("id", hatcheryMarathon.marathonID)

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
			log.Info("killAwolWorkers> killing awol worker %s", app.ID)
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
