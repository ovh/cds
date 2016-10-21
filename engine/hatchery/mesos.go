package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"

	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var (
	marathonHost     string
	marathonID       string
	marathonVHOST    string
	marathonUser     string
	marathonPassword string
)

type marathonPOSTAppParams struct {
	DockerImage   string
	APIEndpoint   string
	WorkerKey     string
	WorkerName    string
	WorkerModelID int64
	HatcheryID    int64

	MarathonID    string
	MarathonVHOST string
	Memory        int
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
		"mem": {{.Memory}}
}
`

// HatcheryMesos implements HatcheryMode interface for mesos mode
type HatcheryMesos struct {
	hatch *hatchery.Hatchery
}

// ID must returns hatchery id
func (m *HatcheryMesos) ID() int64 {
	if m.hatch == nil {
		return 0
	}
	return m.hatch.ID
}

// SetWorkerModelID set the workerModelIDon each heartbeat
func (m *HatcheryMesos) SetWorkerModelID(id int64) {}

// Mode must returns hatchery mode
func (m *HatcheryMesos) Mode() string {
	if m == nil {
		return ""
	}
	return MesosMode
}

//Hatchery returns hatchery instance
func (m *HatcheryMesos) Hatchery() *hatchery.Hatchery {
	return m.hatch
}

// KillWorker deletes an application on mesos via marathon
func (m *HatcheryMesos) KillWorker(worker sdk.Worker) error {
	appID := path.Join(marathonID, worker.Name)
	log.Notice("killMesosWorker> Killing %s\n", appID)
	return deleteApp(marathonHost, marathonUser, marathonPassword, appID)
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
	log.Notice("Spawning worker %s (%s)\n", model.Name, model.Image)
	var err error

	// Do not DOS marathon, if deployment queue is longer than 10, wait
	deployments, err := getDeployments(marathonHost, marathonUser, marathonPassword, "/cds/")
	if err != nil {
		return err
	}
	if len(deployments) >= 10 {
		log.Notice("%d item in deployment queue, waiting\n", len(deployments))
		time.Sleep(2 * time.Second)
		return nil
	}

	apps, err := getApps(marathonHost, marathonUser, marathonPassword, marathonID)
	if err != nil {
		return err
	}
	if len(apps) >= maxWorker {
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

	switch model.Type {
	case sdk.Docker:
		return spawnMesosDockerWorker(model, m.hatch.ID)
	}

	return fmt.Errorf("Model not handled\n")
}

// WorkerStarted returns the number of instances of given model started but
// not necessarily register on CDS yet
func (m *HatcheryMesos) WorkerStarted(model *sdk.Model) int {
	apps, err := getApps(marathonHost, marathonUser, marathonPassword, marathonID)
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

// ParseConfig for mesos mode
func (m *HatcheryMesos) ParseConfig() {
	marathonHost = os.Getenv("MARATHON_HOST")
	if marathonHost == "" {
		sdk.Exit("marathon-host not provided, aborting\n")
	}

	marathonID = os.Getenv("MARATHON_ID")
	if marathonID == "" {
		sdk.Exit("marathon-id not provided, aborting\n")
	}

	marathonVHOST = os.Getenv("MARATHON_VHOST")
	if marathonVHOST == "" {
		sdk.Exit("marathon-vhost not provided, aborting\n")
	}

	marathonUser = os.Getenv("MARATHON_USER")
	if marathonUser == "" {
		sdk.Exit("marathon-user not provided, aborting\n")
	}

	marathonPassword = os.Getenv("MARATHON_PASSWORD")
	if marathonPassword == "" {
		sdk.Exit("marathon-password not provided, aborting\n")
	}

}

// Init only starts killing routine of worker not registered
func (m *HatcheryMesos) Init() error {

	// Register without declaring model
	name, err := os.Hostname()
	if err != nil {
		log.Warning("Cannot retrieve hostname: %s\n", err)
		name = "cds-hatchery-mesos"
	}
	m.hatch = &hatchery.Hatchery{
		Name: name,
		UID:  uk,
	}
	err = register(m.hatch)
	if err != nil {
		log.Warning("Cannot register hatchery: %s\n", err)
	}

	// Start cleaning routines
	startKillAwolWorkerRoutine()
	return nil
}

func spawnMesosDockerWorker(model *sdk.Model, hatcheryID int64) error {
	tmpl, err := template.New("marathonPOST").Parse(marathonPOSTAppTemplate)
	if err != nil {
		return err
	}

	// Estimate needed memory
	memory := 1024
	for _, c := range model.Capabilities {
		if c.Value == "java" {
			memory = 4096
		}
		if c.Value == "go" && memory < 3072 {
			memory = 3072
		}
		if c.Value == "npm" && memory < 2048 {
			memory = 2048
		}
		if c.Value == "python" && memory < 2048 {
			memory = 2048
		}
	}

	for {
		params := marathonPOSTAppParams{
			DockerImage:   model.Image,
			APIEndpoint:   sdk.Host,
			WorkerKey:     uk,
			WorkerName:    fmt.Sprintf("%s-%s", strings.ToLower(model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)),
			WorkerModelID: model.ID,
			HatcheryID:    hatcheryID,
			MarathonID:    marathonID,
			MarathonVHOST: marathonVHOST,
			Memory:        memory,
		}

		var buffer bytes.Buffer
		err = tmpl.Execute(&buffer, params)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("POST", marathonHost+"/v2/apps", &buffer)
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "CDS-HATCHERY/1.0")
		req.SetBasicAuth(marathonUser, marathonPassword)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode < 300 {
			resp.Body.Close()
			//time.Sleep(1 * time.Second) // Given time to mesos to start the worker
			return nil
		}

		log.Warning("STATUS: %s\n", resp.Status)

		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return fmt.Errorf("%s", resp.Status)
		}
		resp.Body.Close()
	}

}

func startKillAwolWorkerRoutine() {
	if hatcheryMode != "mesos" {
		return
	}

	go func() {
		for {
			time.Sleep(10 * time.Second)

			err := killDisabledWorkers()
			if err != nil {
				log.Warning("Cannot kill awol workers: %s\n", err)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)

			err := killAwolWorkers()
			if err != nil {
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

	apps, err := getApps(marathonHost, marathonUser, marathonPassword, marathonID)
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
				err := deleteApp(marathonHost, marathonUser, marathonPassword, app.ID)
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

	apps, err := getApps(marathonHost, marathonUser, marathonPassword, marathonID)
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
			}
		}

		// then if it's not found, kill it !
		if !found && time.Since(t) > 1*time.Minute {
			log.Notice("killing awol worker %s\n", apps[i].ID)
			err := deleteApp(marathonHost, marathonUser, marathonPassword, apps[i].ID)
			if err != nil {
				return err
			}
		}

	}

	return nil
}
