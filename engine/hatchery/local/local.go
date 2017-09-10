package local

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// HatcheryConfiguration is the configuration for local hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration
	Basedir string `default:"/tmp"`
}

// New instanciates a new hatchery local
func New() *HatcheryLocal {
	return new(HatcheryLocal)
}

func (h *HatcheryLocal) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	return nil
}

func (h *HatcheryLocal) CheckConfiguration(cfg interface{}) error {
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

	if hconfig.Basedir == "" {
		return fmt.Errorf("Invalid basedir directory")
	}

	if ok, err := api.DirectoryExists(hconfig.Basedir); !ok {
		return fmt.Errorf("Basedir doesn't exist")
	} else if err != nil {
		return fmt.Errorf("Invalid basedir: %v", err)
	}
	return nil
}

func (h *HatcheryLocal) Serve(ctx context.Context) error {
	hatchery.Create(h)
	return nil
}

// HatcheryLocal implements HatcheryMode interface for local usage
type HatcheryLocal struct {
	Config HatcheryConfiguration
	sync.Mutex
	hatch   *sdk.Hatchery
	workers map[string]*exec.Cmd
	client  cdsclient.Interface
	os      string
	arch    string
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

//Client returns cdsclient instance
func (h *HatcheryLocal) Client() cdsclient.Interface {
	return h.client
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryLocal) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryLocal) ModelType() string {
	return sdk.HostProcess
}

// CanSpawn return wether or not hatchery can spawn model.
// requirements are not supported
func (h *HatcheryLocal) CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	if h.Hatchery() == nil {
		log.Debug("CanSpawn false Hatchery nil")
		return false
	}
	if model.ID != h.Hatchery().Model.ID {
		log.Debug("CanSpawn false ID different model.ID:%d h.workerModelID:%d ", model.ID, h.Hatchery().Model.ID)
		return false
	}
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			return false
		}
	}
	log.Debug("CanSpawn true for job %d", jobID)
	return true
}

// killWorker kill a local process
func (h *HatcheryLocal) killWorker(worker sdk.Worker) error {
	for name, cmd := range h.workers {
		if worker.Name == name {
			log.Info("KillLocalWorker> Killing %s", worker.Name)
			return cmd.Process.Kill()
		}
	}

	return fmt.Errorf("Worker not found")
}

// SpawnWorker starts a new worker process
func (h *HatcheryLocal) SpawnWorker(wm *sdk.Model, jobID int64, requirements []sdk.Requirement, registerOnly bool, logInfo string) (string, error) {
	var err error

	if len(h.workers) >= h.Config.Provision.MaxWorker {
		return "", fmt.Errorf("Max capacity reached (%d)", h.Config.Provision.MaxWorker)
	}

	if jobID > 0 {
		log.Info("spawnWorker> spawning worker %s (%s) for job %d - %s", wm.Name, wm.Image, jobID, logInfo)
	} else {
		log.Info("spawnWorker> spawning worker %s (%s) - %s", wm.Name, wm.Image, logInfo)
	}

	wName := fmt.Sprintf("%s-%s", h.hatch.Name, namesgenerator.GetRandomName(0))
	if registerOnly {
		wName = "register-" + wName
	}

	var args []string
	args = append(args, fmt.Sprintf("--api=%s", h.Client().APIURL()))
	args = append(args, fmt.Sprintf("--token=%s", h.Config.API.Token))
	args = append(args, fmt.Sprintf("--basedir=%s", h.Config.Basedir))
	args = append(args, fmt.Sprintf("--model=%d", h.Hatchery().Model.ID))
	args = append(args, fmt.Sprintf("--name=%s", wName))
	args = append(args, fmt.Sprintf("--hatchery=%d", h.hatch.ID))
	args = append(args, fmt.Sprintf("--hatchery-name=%s", h.hatch.Name))

	if h.Config.Provision.WorkerLogsOptions.Graylog.Host != "" {
		args = append(args, fmt.Sprintf("--graylog-host=%s", h.Config.Provision.WorkerLogsOptions.Graylog.Host))
	}
	if h.Config.Provision.WorkerLogsOptions.Graylog.Port != 0 {
		args = append(args, fmt.Sprintf("--graylog-port=%d", h.Config.Provision.WorkerLogsOptions.Graylog.Port))
	}
	if h.Config.Provision.WorkerLogsOptions.Graylog.ExtraKey != "" {
		args = append(args, fmt.Sprintf("--graylog-extra-key=%s", h.Config.Provision.WorkerLogsOptions.Graylog.ExtraKey))
	}
	if h.Config.Provision.WorkerLogsOptions.Graylog.Extravalue != "" {
		args = append(args, fmt.Sprintf("--graylog-extra-value=%s", h.Config.Provision.WorkerLogsOptions.Graylog.Extravalue))
	}
	if h.Config.API.GRPC.URL != "" && wm.Communication == sdk.GRPC {
		args = append(args, fmt.Sprintf("--grpc-api=%s", h.Config.API.GRPC.URL))
		args = append(args, fmt.Sprintf("--grpc-insecure=%t", h.Config.API.GRPC.Insecure))
	}

	args = append(args, "--single-use")

	if jobID > 0 {
		args = append(args, fmt.Sprintf("--booked-job-id=%d", jobID))
	}

	if registerOnly {
		args = append(args, "register")
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
		return "", err
	}
	h.Lock()
	h.workers[wName] = cmd
	h.Unlock()

	// Wait in a goroutine so that when process exits, Wait() update cmd.ProcessState
	go func() {
		cmd.Wait()
	}()
	return wName, nil
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

	h.client = cdsclient.NewHatchery(
		h.Configuration().API.HTTP.URL,
		h.Configuration().API.Token,
		h.Configuration().Provision.RegisterFrequency,
		h.Configuration().API.HTTP.Insecure,
	)

	req, err := h.Client().Requirements()
	if err != nil {
		return fmt.Errorf("Cannot fetch requirements: %s", err)
	}

	capa, err := checkCapabilities(req)
	if err != nil {
		return fmt.Errorf("Cannot check local capabilities: %s", err)
	}

	genname := hatchery.GenerateName("local", h.Configuration().Name)

	h.hatch = &sdk.Hatchery{
		Name: genname,
		Model: sdk.Model{
			Name:         genname,
			Image:        genname,
			Capabilities: capa,
		},
		Version: sdk.VERSION,
	}

	if err := hatchery.Register(h); err != nil {
		return fmt.Errorf("Cannot register: %s", err)
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

	apiworkers, err := h.Client().WorkerList()
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
			log.Info("Killing AWOL worker %s", w.Name)
			if err := h.killWorker(w); err != nil {
				log.Warning("Error killing worker %s :%s", name, err)
			}
			killedWorkers = append(killedWorkers, name)
			break
		}
		// Worker is disabled. kill it
		if w.Status == sdk.StatusDisabled {
			log.Info("Killing disabled worker %s", w.Name)

			if err := h.killWorker(w); err != nil {
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

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryLocal) NeedRegistration(m *sdk.Model) bool {
	if m.NeedRegistration || m.LastRegistration.Unix() < m.UserLastModified.Unix() {
		return true
	}
	return false
}
