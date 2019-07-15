package local

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// New instanciates a new hatchery local
func New() *HatcheryLocal {
	s := new(HatcheryLocal)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	s.LocalWorkerRunner = new(localWorkerRunner)
	return s
}

func (h *HatcheryLocal) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(HatcheryConfiguration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid local hatchery configuration"))
	}

	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
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
	h.Name = genname
	h.HTTPURL = h.Config.URL

	h.Type = services.TypeHatchery
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	h.Common.Common.ServiceName = "cds-hatchery-local"

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryLocal) Status() sdk.MonitoringStatus {
	m := h.CommonMonitoring()
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{
		Component: "Workers",
		Value:     fmt.Sprintf("%d/%d", len(h.WorkersStarted()), h.Config.Provision.MaxWorker),
		Status:    sdk.MonitoringStatusOK,
	})

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

	if ok, err := sdk.DirectoryExists(hconfig.Basedir); !ok {
		return fmt.Errorf("Basedir doesn't exist")
	} else if err != nil {
		return fmt.Errorf("Invalid basedir: %v", err)
	}
	return nil
}

//Service returns service instance
func (h *HatcheryLocal) Service() *sdk.Service {
	return h.Common.Common.ServiceInstance
}

//Hatchery returns hatchery instance
func (h *HatcheryLocal) Hatchery() *sdk.Hatchery {
	return h.hatch
}

// Serve start the hatchery server
func (h *HatcheryLocal) Serve(ctx context.Context) error {
	h.hatch = &sdk.Hatchery{}
	return h.CommonServe(ctx, h)
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryLocal) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
}

// CanSpawn return wether or not hatchery can spawn model.
// requirements are not supported
func (h *HatcheryLocal) CanSpawn(_ *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	if h.Hatchery() == nil {
		log.Debug("CanSpawn false Hatchery nil")
		return false
	}

	for _, r := range requirements {
		ok, err := h.checkRequirement(r)
		if err != nil || !ok {
			log.Debug("CanSpawn false hatchery.checkRequirement ok:%v err:%v r:%v", ok, err, r)
			return false
		}
	}

	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			log.Debug("CanSpawn false service or memory")
			return false
		}

		if r.Type == sdk.OSArchRequirement && r.Value != (runtime.GOOS+"/"+runtime.GOARCH) {
			log.Debug("CanSpawn> job %d cannot spawn on this OSArch.", jobID)
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

type localWorkerRunner struct{}

func (localWorkerRunner) NewCmd(command string, args ...string) *exec.Cmd {
	var cmd = exec.Command(command, args...)
	cmd.Env = []string{}
	return cmd
}

const workerCmdTmpl = "worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --graylog-extra-key={{.GraylogExtraKey}} --graylog-extra-value={{.GraylogExtraValue}} --graylog-host={{.GraylogHost}} --graylog-port={{.GraylogPort}} --booked-workflow-job-id={{.WorkflowJobID}} --single-use"

// SpawnWorker starts a new worker process
func (h *HatcheryLocal) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	log.Debug("HatcheryLocal.SpawnWorker> %s want to spawn a worker named %s (jobID = %d)", spawnArgs.HatcheryName, spawnArgs.WorkerName, spawnArgs.JobID)

	// Generate a random string 16 chars length
	bs := make([]byte, 16)
	if _, err := rand.Read(bs); err != nil {
		return err
	}
	rndstr := hex.EncodeToString(bs)[0:16]
	basedir := path.Join(h.Config.Basedir, rndstr)
	// Create the directory
	if err := os.MkdirAll(basedir, os.FileMode(0755)); err != nil {
		return err
	}

	udataParam := sdk.WorkerArgs{
		API:               h.Configuration().API.HTTP.URL,
		Token:             h.Configuration().API.Token,
		BaseDir:           basedir,
		HTTPInsecure:      h.Config.API.HTTP.Insecure,
		Name:              spawnArgs.WorkerName,
		Model:             spawnArgs.ModelName(),
		HatcheryName:      h.Name,
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
	}

	udataParam.WorkflowJobID = spawnArgs.JobID

	tmpl, errt := template.New("cmd").Parse(workerCmdTmpl)
	if errt != nil {
		return errt
	}
	var buffer bytes.Buffer
	if errTmpl := tmpl.Execute(&buffer, udataParam); errTmpl != nil {
		return errTmpl
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
		cmd = h.LocalWorkerRunner.NewCmd(binCmd, cmdSplitted...)
	} else {
		cmd = h.LocalWorkerRunner.NewCmd(binCmd, cmdSplitted[1:]...)
	}

	// Clearenv
	env := os.Environ()
	for _, e := range env {
		if !strings.HasPrefix(e, "CDS") && !strings.HasPrefix(e, "HATCHERY") {
			cmd.Env = append(cmd.Env, e)
		}
	}

	if err := cmd.Start(); err != nil {
		log.Error("hatchery> local> %v", err)
		return err
	}

	log.Debug("worker %s has been spawned by %s", spawnArgs.WorkerName, h.Name)

	h.Lock()
	h.workers[spawnArgs.WorkerName] = workerCmd{cmd: cmd, created: time.Now()}
	h.Unlock()
	// Wait in a goroutine so that when process exits, Wait() update cmd.ProcessState
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Error("hatchery> local> %v", err)
		}
	}()

	return nil
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

// InitHatchery register local hatchery with its worker model
func (h *HatcheryLocal) InitHatchery() error {
	h.workers = make(map[string]workerCmd)
	sdk.GoRoutine(context.Background(), "startKillAwolWorkerRoutine", h.startKillAwolWorkerRoutine)
	return nil
}

func (h *HatcheryLocal) localWorkerIndexCleanup() {
	h.Lock()
	defer h.Unlock()

	needToDeleteWorkers := []string{}
	for name, workerCmd := range h.workers {
		// check if worker is still alive
		if workerCmd.cmd.ProcessState != nil && workerCmd.cmd.ProcessState.Exited() {
			log.Debug("process %s has been removed", name)
			needToDeleteWorkers = append(needToDeleteWorkers, name)
		}
	}

	for _, name := range needToDeleteWorkers {
		delete(h.workers, name)
	}
}

func (h *HatcheryLocal) startKillAwolWorkerRoutine(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	for range t.C {
		if err := h.killAwolWorkers(); err != nil {
			log.Warning("Cannot kill awol workers: %s", err)
		}
	}
}

func (h *HatcheryLocal) killAwolWorkers() error {
	log.Debug("hatchery> local> killAwolWorkers")
	h.localWorkerIndexCleanup()

	h.Lock()
	defer h.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	apiWorkers, err := h.CDSClient().WorkerList(ctx)
	if err != nil {
		return err
	}

	mAPIWorkers := make(map[string]sdk.Worker, len(apiWorkers))
	for _, w := range apiWorkers {
		mAPIWorkers[w.Name] = w
	}

	killedWorkers := []string{}
	for name, workerCmd := range h.workers {
		var kill bool
		// if worker not found on api side or disabled, kill it
		if w, ok := mAPIWorkers[name]; !ok {
			// if no name on api, and worker create less than 10 seconds, don't kill it
			if time.Now().Unix()-10 < workerCmd.created.Unix() {
				log.Debug("killAwolWorkers> Avoid killing baby worker %s born at %s", name, workerCmd.created)
				continue
			}
			log.Info("Killing AWOL worker %s", name)
			kill = true
		} else if w.Status == sdk.StatusDisabled {
			log.Info("Killing disabled worker %s", w.Name)
			kill = true
		}

		if kill {
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

// checkRequirement checks binary requirement in path
func (h *HatcheryLocal) checkRequirement(r sdk.Requirement) (bool, error) {
	switch r.Type {
	case sdk.BinaryRequirement:
		if _, err := exec.LookPath(r.Value); err != nil {
			log.Debug("checkRequirement> %v not in path", r.Value)
			// Return nil because the error contains 'Exit status X', that's what we wanted
			return false, nil
		}
		return true, nil
	case sdk.PluginRequirement:
		return true, nil
	case sdk.OSArchRequirement:
		osarch := strings.Split(r.Value, "/")
		if len(osarch) != 2 {
			return false, fmt.Errorf("invalid requirement %s", r.Value)
		}
		return osarch[0] == strings.ToLower(sdk.GOOS) && osarch[1] == strings.ToLower(sdk.GOARCH), nil
	default:
		return false, nil
	}
}
