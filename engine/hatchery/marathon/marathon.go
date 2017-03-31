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

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

var hatcheryMarathon *HatcheryMarathon

// HatcheryMarathon implements HatcheryMode interface for mesos mode
type HatcheryMarathon struct {
	hatch *sdk.Hatchery
	token string

	client marathon.Marathon

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
func (m *HatcheryMarathon) ID() int64 {
	if m.hatch == nil {
		return 0
	}
	return m.hatch.ID
}

//Hatchery returns hatchery instance
func (m *HatcheryMarathon) Hatchery() *sdk.Hatchery {
	return m.hatch
}

// KillWorker deletes an application on mesos via marathon
func (m *HatcheryMarathon) KillWorker(worker sdk.Worker) error {
	appID := path.Join(hatcheryMarathon.marathonID, worker.Name)
	log.Notice("KillWorker> Killing %s", appID)

	_, err := m.client.DeleteApplication(appID, true)
	return err
}

// ModelType returns type of hatchery
func (*HatcheryMarathon) ModelType() string {
	return sdk.Docker
}

// CanSpawn return wether or not hatchery can spawn model
// requirements services are not supported
func (m *HatcheryMarathon) CanSpawn(model *sdk.Model, job *sdk.PipelineBuildJob) bool {
	//Service requirement are not supported
	for _, r := range job.Job.Action.Requirements {
		if r.Type == sdk.ServiceRequirement {
			log.Notice("CanSpawn> Job %d has a service requirement. Marathon can't spawn a worker for this job", job.ID)
			return false
		}
	}

	deployments, errd := m.client.Deployments()
	if errd != nil {
		log.Notice("CanSpawn> Error on m.client.Deployments() : %s", errd)
		return false
	}
	// Do not DOS marathon, if deployment queue is longer than 10
	if len(deployments) >= 10 {
		log.Notice("CanSpawn> %d item in deployment queue, waiting", len(deployments))
		return false
	}

	apps, err := m.listApplications(m.marathonID)
	if err != nil {
		log.Notice("CanSpawn> Error on m.listApplications() : %s", errd)
		return false
	}
	if len(apps) >= viper.GetInt("max-worker") {
		log.Notice("CanSpawn> max number of containers reached, aborting. Current: %d. Max: %d", len(apps), viper.GetInt("max-worker"))
		return false
	}

	return true
}

// SpawnWorker creates an application on mesos via marathon
// requirements services are not supported
func (m *HatcheryMarathon) SpawnWorker(model *sdk.Model, job *sdk.PipelineBuildJob) error {
	if job != nil {
		log.Notice("spawnWorker> spawning worker %s (%s) for job %d", model.Name, model.Image, job.ID)
	} else {
		log.Notice("spawnWorker> spawning worker %s (%s)", model.Name, model.Image)
	}

	var logJob string

	// Estimate needed memory, we will set 110% of required memory
	memory := m.defaultMemory

	cmd := "rm -f worker && curl ${CDS_API}/download/worker/$(uname -m) -o worker &&  chmod +x worker && exec ./worker"
	instance := 1
	workerName := fmt.Sprintf("%s-%s", strings.ToLower(model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1))
	forcePull := strings.HasSuffix(model.Image, ":latest")

	env := map[string]string{
		"CDS_API":        sdk.Host,
		"CDS_KEY":        m.token,
		"CDS_NAME":       workerName,
		"CDS_MODEL":      fmt.Sprintf("%d", model.ID),
		"CDS_HATCHERY":   fmt.Sprintf("%d", m.hatch.ID),
		"CDS_SINGLE_USE": "1",
		"CDS_TTL":        fmt.Sprintf("%d", m.workerTTL),
	}

	//Check if there is a memory requirement
	//if there is a service requirement: exit
	if job != nil {
		logJob = fmt.Sprintf("for job %d,", job.ID)
		env["CDS_BOOKED_JOB_ID"] = fmt.Sprintf("%d", job.ID)

		for _, r := range job.Job.Action.Requirements {
			if r.Name == sdk.ServiceRequirement {
				return fmt.Errorf("spawnMarathonDockerWorker> %s service requirement not supported", logJob)
			}

			if r.Type == sdk.MemoryRequirement {
				var err error
				memory, err = strconv.Atoi(r.Value)
				if err != nil {
					log.Warning("spawnMarathonDockerWorker> %s unable to parse memory requirement %s:%s", logJob, memory, err)
					return err
				}
			}
		}
	}

	mem := float64(memory * 110 / 100)

	application := &marathon.Application{
		ID:  fmt.Sprintf("%s/%s", m.marathonID, workerName),
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

	if _, err := m.client.CreateApplication(application); err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second * 5)
	go func() {
		t0 := time.Now()
		for t := range ticker.C {
			delta := math.Floor(t.Sub(t0).Seconds())
			log.Debug("spawnMarathonDockerWorker> %s worker %s spawning in progress [%d seconds] please wait...", logJob, application.ID, int(delta))
		}
	}()

	log.Debug("spawnMarathonDockerWorker> %s worker %s spawning in progress, please wait...", logJob, application.ID)

	deployments, err := m.client.ApplicationDeployments(application.ID)
	if err != nil {
		ticker.Stop()
		return fmt.Errorf("spawnMarathonDockerWorker> %s failed to list deployments: %s", logJob, err.Error())
	}

	if len(deployments) == 0 {
		ticker.Stop()
		return nil
	}

	wg := &sync.WaitGroup{}
	var done bool
	var successChan = make(chan bool, len(deployments))
	for _, deploy := range deployments {
		wg.Add(1)
		go func(id string) {
			go func() {
				time.Sleep((time.Duration(m.workerSpawnTimeout) + 1) * time.Second)
				if done {
					return
				}
				// try to delete deployment
				log.Debug("spawnMarathonDockerWorker> %s timeout (%d) on deployment %s", logJob, m.workerSpawnTimeout, id)
				if _, err := m.client.DeleteDeployment(id, true); err != nil {
					log.Warning("spawnMarathonDockerWorker> %s error on delete timeouted deployment %s: %s", logJob, id, err.Error())
				}
				ticker.Stop()
				successChan <- false
				wg.Done()
			}()

			if err := m.client.WaitOnDeployment(id, time.Duration(m.workerSpawnTimeout)*time.Second); err != nil {
				log.Warning("spawnMarathonDockerWorker> %s error on deployment %s: %s", logJob, id, err.Error())
				ticker.Stop()
				successChan <- false
				wg.Done()
				return
			}

			log.Debug("spawnMarathonDockerWorker> %s deployment %s succeeded", logJob, id)
			ticker.Stop()
			successChan <- true
			wg.Done()
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
	ticker.Stop()
	close(successChan)
	done = true

	if success {
		return nil
	}

	return fmt.Errorf("spawnMarathonDockerWorker> %s error while deploying worker", logJob)
}

func (m *HatcheryMarathon) listApplications(idPrefix string) ([]string, error) {
	values := url.Values{}
	values.Set("embed", "apps.counts")
	values.Set("id", hatcheryMarathon.marathonID)
	return m.client.ListApplications(values)
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (m *HatcheryMarathon) WorkersStarted() int {
	apps, err := m.listApplications(hatcheryMarathon.marathonID)
	if err != nil {
		log.Warning("WorkersStarted> error on list applications err:%s", err)
		return 0
	}
	return len(apps)
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (m *HatcheryMarathon) WorkersStartedByModel(model *sdk.Model) int {
	apps, err := m.listApplications(hatcheryMarathon.marathonID)
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
func (m *HatcheryMarathon) Init() error {
	// Register without declaring model
	m.hatch = &sdk.Hatchery{
		Name: hatchery.GenerateName("marathon", viper.GetString("name")),
		UID:  viper.GetString("token"),
	}

	if err := hatchery.Register(m.hatch, viper.GetString("token")); err != nil {
		log.Warning("Cannot register hatchery: %s", err)
	}

	// Start cleaning routines
	m.startKillAwolWorkerRoutine()
	return nil
}

func (m *HatcheryMarathon) startKillAwolWorkerRoutine() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			if err := m.killDisabledWorkers(); err != nil {
				log.Warning("Cannot kill disabled workers: %s", err)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			if err := m.killAwolWorkers(); err != nil {
				log.Warning("Cannot kill awol workers: %s", err)
			}
		}
	}()
}

func (m *HatcheryMarathon) killDisabledWorkers() error {
	workers, err := sdk.GetWorkers()
	if err != nil {
		return err
	}

	apps, err := m.listApplications(hatcheryMarathon.marathonID)
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
				log.Notice("killing disabled worker %s", app)
				if _, err := m.client.DeleteApplication(app, true); err != nil {
					log.Warning("killDisabledWorkers> Error while delete app %s err:%s", app, err)
					// continue to next app
				}
			}
		}
	}

	return nil
}

func (m *HatcheryMarathon) killAwolWorkers() error {
	log.Debug("killAwolWorkers>")
	workers, err := sdk.GetWorkers()
	if err != nil {
		return err
	}

	values := url.Values{}
	values.Set("embed", "apps.counts")
	values.Set("id", hatcheryMarathon.marathonID)

	apps, err := m.client.Applications(values)
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
			log.Notice("killAwolWorkers> killing awol worker %s", app.ID)
			if _, err := m.client.DeleteApplication(app.ID, true); err != nil {
				log.Warning("killAwolWorkers> Error while delete app %s err:%s", app.ID, err)
				// continue to next app
			}
		}
	}

	return nil
}
