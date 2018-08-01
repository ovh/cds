package local

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
)

// New instanciates a new hatchery local
func New() *HatcheryLocal {
	s := new(HatcheryLocal)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryLocal) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	genname := h.Configuration().Name
	h.Client = cdsclient.NewHatchery(
		h.Configuration().API.HTTP.URL,
		h.Configuration().API.Token,
		h.Configuration().Provision.RegisterFrequency,
		h.Configuration().API.HTTP.Insecure,
		genname,
	)

	h.API = h.Config.API.HTTP.URL
	h.Name = h.Config.Name
	h.HTTPURL = h.Config.URL
	h.Token = h.Config.API.Token
	h.Type = services.TypeHatchery
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	h.ServiceName = "cds-hatchery-local"

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryLocal) Status() sdk.MonitoringStatus {
	m := h.CommonMonitoring()
	if h.IsInitialized() {
		m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(h.WorkersStarted()), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})
	}
	return m
}

// CheckConfiguration checks the validity of the configuration object
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

	if hconfig.Name == "" {
		return fmt.Errorf("please enter a name in your local hatchery configuration")
	}

	if ok, err := api.DirectoryExists(hconfig.Basedir); !ok {
		return fmt.Errorf("Basedir doesn't exist")
	} else if err != nil {
		return fmt.Errorf("Invalid basedir: %v", err)
	}
	return nil
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

// Serve start the hatchery server
func (h *HatcheryLocal) Serve(ctx context.Context) error {
	req, err := h.CDSClient().Requirements()
	if err != nil {
		return fmt.Errorf("Cannot fetch requirements: %s", err)
	}

	capa, err := checkCapabilities(req)
	if err != nil {
		return fmt.Errorf("Cannot check local capabilities: %s", err)
	}

	h.hatch = &sdk.Hatchery{
		Name: h.Name,
		Model: sdk.Model{
			Name: h.Name,
			Type: sdk.HostProcess,
			ModelVirtualMachine: sdk.ModelVirtualMachine{
				Image: h.Name,
			},
			RegisteredCapabilities: capa,
			Provision:              int64(h.Config.NbProvision),
		},
		Version: sdk.VERSION,
	}

	return h.CommonServe(ctx, h)
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
func (h *HatcheryLocal) killWorker(name string, workerCmd workerCmd) error {
	log.Info("KillLocalWorker> Killing %s", name)
	return workerCmd.cmd.Process.Kill()
}

// SpawnWorker starts a new worker process
func (h *HatcheryLocal) SpawnWorker(spawnArgs hatchery.SpawnArguments) (string, error) {
	wName := fmt.Sprintf("%s-%s", h.hatch.Name, namesgenerator.GetRandomName(0))
	if spawnArgs.RegisterOnly {
		wName = "register-" + wName
	}

	if spawnArgs.JobID > 0 {
		log.Debug("spawnWorker> spawning worker %s (%s) for job %d - %s", wName, spawnArgs.Model.ModelVirtualMachine.Image, spawnArgs.JobID, spawnArgs.LogInfo)
	} else {
		log.Debug("spawnWorker> spawning worker %s (%s) - %s", wName, spawnArgs.Model.ModelVirtualMachine.Image, spawnArgs.LogInfo)
	}

	udataParam := sdk.WorkerArgs{
		API:               h.Configuration().API.HTTP.URL,
		Token:             h.Configuration().API.Token,
		BaseDir:           h.Config.Basedir,
		HTTPInsecure:      h.Config.API.HTTP.Insecure,
		Name:              wName,
		Model:             spawnArgs.Model.ID,
		Hatchery:          h.hatch.ID,
		HatcheryName:      h.hatch.Name,
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
		GrpcAPI:           h.Configuration().API.GRPC.URL,
		GrpcInsecure:      h.Configuration().API.GRPC.Insecure,
	}

	if spawnArgs.JobID > 0 {
		if spawnArgs.IsWorkflowJob {
			udataParam.WorkflowJobID = spawnArgs.JobID
		} else {
			udataParam.PipelineBuildJobID = spawnArgs.JobID
		}
	}

	if spawnArgs.IsWorkflowJob {
		udataParam.WorkflowJobID = spawnArgs.JobID
	} else {
		udataParam.PipelineBuildJobID = spawnArgs.JobID
	}

	if spawnArgs.Model.ModelVirtualMachine.Cmd == "" {
		return "", fmt.Errorf("hatchery local> Cannot launch main worker command because it's empty")
	}

	tmpl, errt := template.New("cmd").Parse(spawnArgs.Model.ModelVirtualMachine.Cmd)
	if errt != nil {
		return "", errt
	}
	var buffer bytes.Buffer
	if errTmpl := tmpl.Execute(&buffer, udataParam); errTmpl != nil {
		return "", errTmpl
	}

	cmdSplitted := strings.Split(buffer.String(), " -")
	for i := range cmdSplitted[1:] {
		cmdSplitted[i+1] = "-" + strings.Trim(cmdSplitted[i+1], " ")
	}

	binCmd := cmdSplitted[0]
	log.Debug("Command exec: %v", cmdSplitted)
	var cmd *exec.Cmd
	if spawnArgs.RegisterOnly {
		cmdSplitted[0] = "register"
		cmd = exec.Command(binCmd, cmdSplitted...)
	} else {
		cmd = exec.Command(binCmd, cmdSplitted[1:]...)
	}

	// Clearenv
	cmd.Env = []string{}
	env := os.Environ()
	for _, e := range env {
		if !strings.HasPrefix(e, "CDS") && !strings.HasPrefix(e, "HATCHERY") {
			cmd.Env = append(cmd.Env, e)
		}
	}

	if err := cmd.Start(); err != nil {
		log.Error("hatchery> local> %v", err)
		return "", err
	}

	log.Debug("worker %s has been spawned by %s", wName, h.Name)

	h.Lock()
	h.workers[wName] = workerCmd{cmd: cmd, created: time.Now()}
	h.Unlock()
	// Wait in a goroutine so that when process exits, Wait() update cmd.ProcessState
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Error("hatchery> local> %v", err)
		}
	}()

	return wName, nil
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryLocal) WorkersStarted() []string {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()
	workers := make([]string, len(h.workers))
	var i int
	for n := range h.workers {
		workers[i] = n
		i++
	}
	return workers
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryLocal) WorkersStartedByModel(model *sdk.Model) int {
	h.localWorkerIndexCleanup()
	var x int

	h.Mutex.Lock()
	defer h.Mutex.Unlock()
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
	h.workers = make(map[string]workerCmd)

	if err := hatchery.Register(h); err != nil {
		return fmt.Errorf("Cannot register: %s", err)
	}

	sdk.GoRoutine("startKillAwolWorkerRoutine", h.startKillAwolWorkerRoutine)
	return nil
}

func (h *HatcheryLocal) localWorkerIndexCleanup() {
	h.Lock()
	defer h.Unlock()

	needToDeleteWorkers := []string{}
	for name, workerCmd := range h.workers {
		// check if worker is still alive
		if workerCmd.cmd.ProcessState != nil && workerCmd.cmd.ProcessState.Exited() {
			needToDeleteWorkers = append(needToDeleteWorkers, name)
		}
	}

	for _, name := range needToDeleteWorkers {
		delete(h.workers, name)
	}
}

func (h *HatcheryLocal) startKillAwolWorkerRoutine() {
	for {
		time.Sleep(5 * time.Second)
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

	apiworkers, err := h.CDSClient().WorkerList()
	if err != nil {
		return err
	}

	killedWorkers := []string{}
	for name, workerCmd := range h.workers {
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
			// if no name on api, and worker create less than 10 seconds, don't kill it
			if time.Now().Unix()-10 < workerCmd.created.Unix() {
				log.Debug("killAwolWorkers> Avoid killing baby worker %s born at %s", name, workerCmd.created)
				continue
			}
			w.Name = name
			log.Info("Killing AWOL worker %s", w.Name)
			if err := h.killWorker(name, workerCmd); err != nil {
				log.Warning("Error killing worker %s :%s", name, err)
			}
			killedWorkers = append(killedWorkers, name)
		} else if w.Status == sdk.StatusDisabled {
			// Worker is disabled. kill it
			log.Info("Killing disabled worker %s", w.Name)

			if err := h.killWorker(name, workerCmd); err != nil {
				log.Warning("Error killing worker %s :%s", name, err)
			}
			killedWorkers = append(killedWorkers, name)
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
