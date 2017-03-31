package local

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/spf13/viper"
)

var hatcheryLocal *HatcheryLocal

// HatcheryLocal implements HatcheryMode interface for local usage
type HatcheryLocal struct {
	sync.Mutex
	hatch   *sdk.Hatchery
	basedir string
	workers map[string]*exec.Cmd
}

// ID must returns hatchery id
func (h *HatcheryLocal) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

//Hatchery returns hatchery instance
func (h *HatcheryLocal) Hatchery() *sdk.Hatchery {
	return h.hatch
}

// ModelType returns type of hatchery
func (*HatcheryLocal) ModelType() string {
	return sdk.HostProcess
}

// CanSpawn return wether or not hatchery can spawn model.
// requirements are not supported
func (h *HatcheryLocal) CanSpawn(model *sdk.Model, job *sdk.PipelineBuildJob) bool {
	if h.Hatchery() == nil {
		log.Debug("CanSpawn false Hatchery nil")
		return false
	}
	if model.ID != h.Hatchery().Model.ID {
		log.Debug("CanSpawn false ID different model.ID:%d h.workerModelID:%d ", model.ID, h.Hatchery().Model.ID)
		return false
	}
	for _, r := range job.Job.Action.Requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			return false
		}
	}
	log.Debug("CanSpawn true for job %d", job.ID)
	return true
}

// KillWorker kill a local process
func (h *HatcheryLocal) KillWorker(worker sdk.Worker) error {
	for name, cmd := range h.workers {
		if worker.Name == name {
			log.Notice("KillLocalWorker> Killing %s", worker.Name)
			return cmd.Process.Kill()
		}
	}

	return fmt.Errorf("Worker not found")
}

// SpawnWorker starts a new worker process
func (h *HatcheryLocal) SpawnWorker(wm *sdk.Model, job *sdk.PipelineBuildJob) error {
	var err error

	if len(h.workers) >= viper.GetInt("max-worker") {
		return fmt.Errorf("Max capacity reached (%d)", viper.GetInt("max-worker"))
	}

	wName := fmt.Sprintf("%s-%s", h.hatch.Name, namesgenerator.GetRandomName(0))

	var args []string
	args = append(args, fmt.Sprintf("--api=%s", sdk.Host))
	args = append(args, fmt.Sprintf("--key=%s", viper.GetString("token")))
	args = append(args, fmt.Sprintf("--basedir=%s", h.basedir))
	args = append(args, fmt.Sprintf("--model=%d", h.Hatchery().Model.ID))
	args = append(args, fmt.Sprintf("--name=%s", wName))
	args = append(args, fmt.Sprintf("--hatchery=%d", h.hatch.ID))
	args = append(args, "--single-use")

	if job != nil {
		args = append(args, fmt.Sprintf("--booked-job-id=%d", job.ID))
	}

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

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryLocal) WorkersStarted() int {
	return len(h.workers)
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryLocal) WorkersStartedByModel(model *sdk.Model) int {
	h.localWorkerIndexCleanup()
	var x int
	for name := range h.workers {
		if strings.Contains(name, model.Name) {
			x++
		}
	}

	return x
}

// checkCapabilities checks all requirements, foreach type binary, check if binary is on current host
// returns an error "Exit status X" if current host misses one requirement
func checkCapabilities(req []sdk.Requirement) ([]sdk.Requirement, error) {
	var capa []sdk.Requirement
	var tmp map[string]sdk.Requirement

	tmp = make(map[string]sdk.Requirement)
	for _, r := range req {
		ok, err := hatchery.CheckRequirement(r)
		if err != nil {
			return nil, err
		}

		if ok {
			tmp[r.Name] = r
		}
	}

	for _, r := range tmp {
		capa = append(capa, r)
	}

	return capa, nil
}

// Init register local hatchery with its worker model
func (h *HatcheryLocal) Init() error {
	h.workers = make(map[string]*exec.Cmd)

	req, err := sdk.GetRequirements()
	if err != nil {
		return fmt.Errorf("Cannot fetch requirements: %s", err)
	}

	capa, err := checkCapabilities(req)
	if err != nil {
		return fmt.Errorf("Cannot check local capabilities: %s", err)
	}

	name := hatchery.GenerateName("local", viper.GetString("name"))

	h.hatch = &sdk.Hatchery{
		Name: name,
		UID:  viper.GetString("token"),
		Model: sdk.Model{
			Name:         name,
			Image:        name,
			Capabilities: capa,
		},
	}

	log.Notice("Call hatchery.Register Init()")

	if err := hatchery.Register(h.hatch, viper.GetString("token")); err != nil {
		log.Warning("Cannot register hatchery: %s", err)
		return err
	}

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
			log.Warning("Cannot kill awol workers: %s", err)
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
			log.Notice("Killing AWOL worker %s", w.Name)
			if err := h.KillWorker(w); err != nil {
				log.Warning("Error killing worker %s :%s", name, err)
			}
			killedWorkers = append(killedWorkers, name)
			break
		}
		// Worker is disabled. kill it
		if w.Status == sdk.StatusDisabled {
			log.Notice("Killing disabled worker %s", w.Name)

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
