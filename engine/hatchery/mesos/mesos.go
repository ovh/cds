package mesos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

var hatcheryMesos *HatcheryMesos

type marathonPOSTAppParams struct {
	DockerImage    string
	APIEndpoint    string
	WorkerKey      string
	WorkerName     string
	WorkerModelID  int64
	HatcheryID     int64
	MarathonID     string
	MarathonVHOST  string
	MarathonLabels string
	Memory         int
}

const marathonPOSTAppTemplate = `
{
    "container": {
        "docker": {
            "forcePullImage": false,
            "image": "{{.DockerImage}}",
            "network": "BRIDGE",
					  "portMapping": []
				},
        "type": "DOCKER"
    },
		"cmd": "rm -f worker && curl ${CDS_API}/download/worker/$(uname -m) -o worker &&  chmod +x worker && exec ./worker",
		"cpus": 0.5,
    "env": {
        "CDS_API": "{{.APIEndpoint}}",
        "CDS_KEY": "{{.WorkerKey}}",
        "CDS_NAME": "{{.WorkerName}}",
        "CDS_MODEL": "{{.WorkerModelID}}",
        "CDS_HATCHERY": "{{.HatcheryID}}",
        "CDS_SINGLE_USE": "1"
    },
    "id": "{{.MarathonID}}/{{.WorkerName}}",
    "instances": 1,
	"ports": [],
	"mem": {{.Memory}},
	"labels": {{.MarathonLabels}}
}
`

// HatcheryMesos implements HatcheryMode interface for mesos mode
type HatcheryMesos struct {
	hatch *sdk.Hatchery

	token string

	marathonHost         string
	marathonID           string
	marathonVHOST        string
	marathonUser         string
	marathonPassword     string
	marathonLabelsString string
	marathonLabels       map[string]string

	defaultMemory int
}

// ID must returns hatchery id
func (m *HatcheryMesos) ID() int64 {
	if m.hatch == nil {
		return 0
	}
	return m.hatch.ID
}

//Hatchery returns hatchery instance
func (m *HatcheryMesos) Hatchery() *sdk.Hatchery {
	return m.hatch
}

// KillWorker deletes an application on mesos via marathon
func (m *HatcheryMesos) KillWorker(worker sdk.Worker) error {
	appID := path.Join(hatcheryMesos.marathonID, worker.Name)
	log.Notice("killMesosWorker> Killing %s\n", appID)
	return deleteApp(hatcheryMesos.marathonHost, hatcheryMesos.marathonUser, hatcheryMesos.marathonPassword, appID)
}

// CanSpawn return wether or not hatchery can spawn model
// requirements are not supported
func (m *HatcheryMesos) CanSpawn(model *sdk.Model, req []sdk.Requirement) bool {
	if model.Type != sdk.Docker {
		return false
	}
	if len(req) > 0 {
		return false
	}
	return true
}

// SpawnWorker creates an application on mesos via marathon
// requirements are not supported
func (m *HatcheryMesos) SpawnWorker(model *sdk.Model, req []sdk.Requirement) error {
	if model.Type != sdk.Docker {
		return fmt.Errorf("Model not handled")
	}

	log.Notice("Spawning worker %s (%s)\n", model.Name, model.Image)

	// Do not DOS marathon, if deployment queue is longer than 10, wait
	deployments, err := getDeployments(hatcheryMesos.marathonHost, hatcheryMesos.marathonUser, hatcheryMesos.marathonPassword)
	if err != nil {
		return err
	}
	if len(deployments) >= 10 {
		log.Notice("%d item in deployment queue, waiting\n", len(deployments))
		time.Sleep(2 * time.Second)
		return nil
	}

	apps, err := getApps(hatcheryMesos.marathonHost, hatcheryMesos.marathonUser, hatcheryMesos.marathonPassword, hatcheryMesos.marathonID)
	if err != nil {
		return err
	}
	if len(apps) >= viper.GetInt("max-worker") {
		return fmt.Errorf("max number of containers reached, aborting")
	}

	mss, err := sdk.GetWorkerModelStatus()
	if err != nil {
		return err
	}
	for _, ms := range mss {
		if ms.ModelName == model.Name {
			// Security against deficient worker model with worker not connecting
			// TODO: Should validate worker before running them at scale
			if int(ms.CurrentCount) > countOf(model.Name, apps)+10 {
				return fmt.Errorf("Over 20 %s workers started on mesos but 0 connected, something is wrong\n", model.Name)
			}
			break
		}
	}

	return m.spawnMesosDockerWorker(model, m.hatch.ID, req)
}

// WorkerStarted returns the number of instances of given model started but
// not necessarily register on CDS yet
func (m *HatcheryMesos) WorkerStarted(model *sdk.Model) int {
	apps, err := getApps(hatcheryMesos.marathonHost, hatcheryMesos.marathonUser, hatcheryMesos.marathonPassword, hatcheryMesos.marathonID)
	if err != nil {
		return 0
	}

	var x int
	for _, app := range apps {
		if strings.Contains(app.ID, strings.ToLower(model.Name)) {
			x++
		}
	}

	return x
}

// Init only starts killing routine of worker not registered
func (m *HatcheryMesos) Init() error {
	// Register without declaring model
	name, err := os.Hostname()
	if err != nil {
		log.Warning("Cannot retrieve hostname: %s\n", err)
		name = "cds-hatchery-mesos"
	}
	m.hatch = &sdk.Hatchery{
		Name: name,
		UID:  viper.GetString("token"),
	}

	if err = hatchery.Register(m.hatch, viper.GetString("token")); err != nil {
		log.Warning("Cannot register hatchery: %s\n", err)
	}

	// Start cleaning routines
	startKillAwolWorkerRoutine()
	return nil
}

func (m *HatcheryMesos) marathonConfig(model *sdk.Model, hatcheryID int64, memory int) (io.Reader, error) {
	tmpl, err := template.New("marathonPOST").Parse(marathonPOSTAppTemplate)
	if err != nil {
		return nil, err
	}

	m.marathonLabels["hatchery"] = fmt.Sprintf("%d", hatcheryID)

	labels, err := json.Marshal(m.marathonLabels)
	if err != nil {
		log.Critical("spawnMesosDockerWorker> Invalid labels : %s", err)
		return nil, err
	}

	params := marathonPOSTAppParams{
		DockerImage:    model.Image,
		APIEndpoint:    sdk.Host,
		WorkerKey:      m.token,
		WorkerName:     fmt.Sprintf("%s-%s", strings.ToLower(model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)),
		WorkerModelID:  model.ID,
		HatcheryID:     hatcheryID,
		MarathonID:     m.marathonID,
		MarathonVHOST:  m.marathonVHOST,
		Memory:         memory * 110 / 100,
		MarathonLabels: string(labels),
	}

	buffer := &bytes.Buffer{}
	if err := tmpl.Execute(buffer, params); err != nil {
		log.Critical("Unable to execute marathon template : %s", err)
		return nil, err
	}

	return buffer, nil
}

func (m *HatcheryMesos) spawnMesosDockerWorker(model *sdk.Model, hatcheryID int64, req []sdk.Requirement) error {
	// Estimate needed memory, we will set 110% of required memory
	memory := m.defaultMemory
	//Check if there is a memory requirement
	//if there is a service requirement: exit
	for _, r := range req {
		if r.Name == sdk.ServiceRequirement {
			return fmt.Errorf("Service requirement not supported")
		}

		if r.Type == sdk.MemoryRequirement {
			var err error
			memory, err = strconv.Atoi(r.Value)
			if err != nil {
				log.Warning("spawnMesosDockerWorker>Unable to parse memory requirement %s :s\n", memory, err)
				return err
			}
		}
	}

	buffer, err := m.marathonConfig(model, hatcheryID, memory)
	r, err := http.NewRequest("POST", hatcheryMesos.marathonHost+"/v2/apps", buffer)
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", "CDS-HATCHERY/1.0")
	r.SetBasicAuth(hatcheryMesos.marathonUser, hatcheryMesos.marathonPassword)

	resp, err := hatchery.Client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return fmt.Errorf("%s", resp.Status)
	}

	return nil
}

func startKillAwolWorkerRoutine() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			if err := killDisabledWorkers(); err != nil {
				log.Warning("Cannot kill awol workers: %s\n", err)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)

			if err := killAwolWorkers(); err != nil {
				log.Warning("Cannot kill awol workers: %s\n", err)
			}
		}
	}()
}

func killDisabledWorkers() error {
	workers, err := sdk.GetWorkers()
	if err != nil {
		return err
	}

	apps, err := getApps(hatcheryMesos.marathonHost, hatcheryMesos.marathonUser, hatcheryMesos.marathonPassword, hatcheryMesos.marathonID)
	if err != nil {
		return err
	}

	for _, w := range workers {
		if w.Status != sdk.StatusDisabled {
			continue
		}

		// check that there is a worker matching
		for _, app := range apps {
			if strings.HasSuffix(app.ID, w.Name) {
				log.Notice("killing disabled worker %s\n", app.ID)
				err := deleteApp(hatcheryMesos.marathonHost, hatcheryMesos.marathonUser, hatcheryMesos.marathonPassword, app.ID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func killAwolWorkers() error {
	workers, err := sdk.GetWorkers()
	if err != nil {
		return err
	}

	apps, err := getApps(hatcheryMesos.marathonHost, hatcheryMesos.marathonUser, hatcheryMesos.marathonPassword, hatcheryMesos.marathonID)
	if err != nil {
		return err
	}

	var found bool
	// then for each RUNNING marathon application
	for i := range apps {
		// Worker is deploying, leave him alone
		if apps[i].TasksRunning == 0 {
			continue
		}
		t, err := time.Parse(time.RFC3339, apps[i].Version)
		if err != nil {
			log.Warning("Cannot parse last update: %s\n", err)
			break
		}

		// check that there is a worker matching
		found = false
		for _, w := range workers {
			if strings.HasSuffix(apps[i].ID, w.Name) && w.Status != sdk.StatusDisabled {
				found = true
				break
			}
		}

		// then if it's not found, kill it !
		if !found && time.Since(t) > 1*time.Minute {
			log.Notice("killing awol worker %s\n", apps[i].ID)

			if err := deleteApp(hatcheryMesos.marathonHost, hatcheryMesos.marathonUser, hatcheryMesos.marathonPassword, apps[i].ID); err != nil {
				return err
			}
		}
	}

	return nil
}
