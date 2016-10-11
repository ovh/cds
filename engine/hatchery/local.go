package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"

	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// HatcheryLocal implements HatcheryMode interface for local usage
type HatcheryLocal struct {
	sync.Mutex
	hatch         *hatchery.Hatchery
	basedir       string
	workers       map[string]*exec.Cmd
	workerModelID int64
}

// SetWorkerModelID set the workerModelIDon each heartbeat
func (h *HatcheryLocal) SetWorkerModelID(id int64) {
	h.workerModelID = id
}

// Mode must returns hatchery mode
func (h *HatcheryLocal) Mode() string {
	if h == nil {
		return ""
	}
	return LocalMode
}

// ID must returns hatchery id
func (h *HatcheryLocal) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

//Hatchery returns hatchery instance
func (h *HatcheryLocal) Hatchery() *hatchery.Hatchery {
	return h.hatch
}

// CanSpawn return wether or not hatchery can spawn model.
// requirements are not supported
func (h *HatcheryLocal) CanSpawn(model *sdk.Model, req []sdk.Requirement) bool {
	if model.ID != h.workerModelID {
		return false
	}
	if len(req) > 0 {
		return false
	}
	return true
}

// Refresh retrieves worker models status from API
// and tries to act if needed
func (h *HatcheryLocal) Refresh() error {
	return nil
}

// KillWorker kill a local process
func (h *HatcheryLocal) KillWorker(worker sdk.Worker) error {
	for name, cmd := range h.workers {
		if worker.Name == name {
			log.Notice("KillLocalWorker> Killing %s\n", worker.Name)
			return cmd.Process.Kill()
		}
	}

	return fmt.Errorf("Worker not found")
}

// SpawnWorker starts a new worker process
func (h *HatcheryLocal) SpawnWorker(wm *sdk.Model, req []sdk.Requirement) error {
	var err error
	uk, err = sdk.GenerateWorkerKey(sdk.FirstUseExpire)
	if err != nil {
		return fmt.Errorf("cannot generate worker key: %s", err)
	}

	if len(h.workers) >= maxWorker {
		return fmt.Errorf("Max capacity reached (%d)\n", maxWorker)
	}

	wName := fmt.Sprintf("%s-%s", h.hatch.Name, namesgenerator.GetRandomName(0))

	var args []string
	args = append(args, fmt.Sprintf("--api=%s", sdk.Host))
	args = append(args, fmt.Sprintf("--key=%s", uk))
	args = append(args, fmt.Sprintf("--basedir=%s", h.basedir))
	args = append(args, fmt.Sprintf("--model=%d", h.workerModelID))
	args = append(args, fmt.Sprintf("--name=%s", wName))
	args = append(args, fmt.Sprintf("--hatchery=%d", h.hatch.ID))
	args = append(args, "--single-use")

	cmd := exec.Command("worker", args...)

	// Clearenv
	cmd.Env = []string{}
	env := os.Environ()
	for _, e := range env {
		if !strings.HasPrefix(e, "CDS") && !strings.HasPrefix(e, "HATCHERY") {
			cmd.Env = append(cmd.Env, e)
		}
	}

	if err = cmd.Start(); err != nil {
		return err
	}
	h.Lock()
	h.workers[wName] = cmd
	h.Unlock()

	// Wait in a goroutine so that when process exits, Wait() update cmd.ProcessState
	go func() {
		cmd.Wait()
	}()
	return nil
}

// WorkerStarted returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryLocal) WorkerStarted(model *sdk.Model) int {
	h.localWorkerIndexCleanup()
	var x int
	for name := range h.workers {
		if strings.Contains(name, model.Name) {
			x++
		}
	}

	return x
}

// ParseConfig for local mode
func (h *HatcheryLocal) ParseConfig() {
	h.basedir = os.Getenv("BASEDIR")
	if h.basedir == "" {
		sdk.Exit("basedir not provided, aborting\n")
	}
}

// Init register local hatchery with its worker model
func (h *HatcheryLocal) Init() error {
	h.workers = make(map[string]*exec.Cmd)

	name, err := os.Hostname()
	if err != nil {
		log.Warning("Cannot retrieve hostname: %s\n", err)
	}

	req, err := sdk.GetRequirements()
	if err != nil {
		log.Warning("Cannot fetch requirements: %s\n", err)
	}

	capa, err := checkCapabilities(req)
	if err != nil {
		log.Warning("Cannot check local capabilities: %s\n", err)
	}

	h.hatch = &hatchery.Hatchery{
		Name: name,
		Model: sdk.Model{
			Name:         name,
			Image:        name,
			Capabilities: capa,
		},
	}

	h.workerModelID = h.hatch.Model.ID

	go h.startKillAwolWorkerRoutine()
	return nil
}

func (h *HatcheryLocal) localWorkerIndexCleanup() {
	h.Lock()
	defer h.Unlock()

	needToDeleteWorkers := []string{}
	for name, cmd := range h.workers {
		// check if worker is still alive
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			needToDeleteWorkers = append(needToDeleteWorkers, name)
		}
	}

	for _, name := range needToDeleteWorkers {
		delete(h.workers, name)
	}

}

func (h *HatcheryLocal) startKillAwolWorkerRoutine() {
	for {
		time.Sleep(30 * time.Second)
		err := h.killAwolWorkers()
		if err != nil {
			log.Warning("Cannot kill awol workers: %s\n", err)
		}
	}
}

func (h *HatcheryLocal) killAwolWorkers() error {
	h.localWorkerIndexCleanup()

	h.Lock()
	defer h.Unlock()

	apiworkers, err := sdk.GetWorkers()
	if err != nil {
		return err
	}

	killedWorkers := []string{}
	for name := range h.workers {
		// look for worker in apiworkers
		var w sdk.Worker
		for i := range apiworkers {
			if apiworkers[i].Name == name {
				w = apiworkers[i]
				break
			}
		}
		// Worker not found on api side. kill it
		if w.Name == "" {
			w.Name = name
			log.Notice("Killing AWOL worker %s\n", w.Name)
			if err := h.KillWorker(w); err != nil {
				log.Warning("Error killing worker %s :%s", name, err)
			}
			killedWorkers = append(killedWorkers, name)
			break
		}
		// Worker is disabled. kill it
		if w.Status == sdk.StatusDisabled {
			log.Notice("Killing disabled worker %s\n", w.Name)

			if err := h.KillWorker(w); err != nil {
				log.Warning("Error killing worker %s :%s", name, err)
			}
			killedWorkers = append(killedWorkers, name)
			break
		}
	}

	for _, name := range killedWorkers {
		delete(h.workers, name)
	}

	return nil
}
