package docker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// New instanciates a new hatchery docker
func New() *HatcheryDocker {
	return new(HatcheryDocker)
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryDocker) ApplyConfiguration(cfg interface{}) error {
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

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryDocker) CheckConfiguration(cfg interface{}) error {
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
	return nil
}

// Serve start the HatcheryDocker server
func (h *HatcheryDocker) Serve(ctx context.Context) error {
	hatchery.Create(h)
	return nil
}

// ID must returns hatchery id
func (h *HatcheryDocker) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

//Hatchery returns hatchery instance
func (h *HatcheryDocker) Hatchery() *sdk.Hatchery {
	return h.hatch
}

//Client returns cdsclient instance
func (h *HatcheryDocker) Client() cdsclient.Interface {
	return h.client
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryDocker) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryDocker) ModelType() string {
	return sdk.Docker
}

// CanSpawn return wether or not hatchery can spawn model
// requirement are not supported
func (h *HatcheryDocker) CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			return false
		}
	}
	return true
}

// Init starts cleaning routine
// and check hatchery can run in docker mode with given configuration
func (h *HatcheryDocker) Init() error {
	h.workers = make(map[string]*exec.Cmd)

	h.hatch = &sdk.Hatchery{
		Name:    hatchery.GenerateName("docker", h.Configuration().Name),
		Version: sdk.VERSION,
	}

	h.client = cdsclient.NewHatchery(
		h.Configuration().API.HTTP.URL,
		h.Configuration().API.Token,
		h.Configuration().Provision.RegisterFrequency,
		h.Configuration().API.HTTP.Insecure,
	)
	if err := hatchery.Register(h); err != nil {
		return fmt.Errorf("Cannot register: %s", err)
	}

	ok, err := hatchery.CheckRequirement(sdk.Requirement{Type: sdk.BinaryRequirement, Value: "docker"})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Docker not found on this host")
	}

	go h.workerIndexCleanupRoutine()
	go h.killAwolWorkerRoutine()
	return nil
}

func (h *HatcheryDocker) workerIndexCleanupRoutine() {
	for {
		time.Sleep(1 * time.Second)
		h.Lock()

		for name, cmd := range h.workers {
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				log.Debug("HatcheryDocker.IndexCleanup: removing exited %s", name)
				delete(h.workers, name)
				break
			}
		}
		h.Unlock()
	}
}

func (h *HatcheryDocker) killAwolWorkerRoutine() {
	for {
		time.Sleep(5 * time.Second)
		h.killAwolWorker()
	}
}

func (h *HatcheryDocker) killAwolWorker() {
	apiworkers, err := h.Client().WorkerList()
	if err != nil {
		log.Warning("Cannot get workers: %s", err)
		return
	}

	h.Lock()
	defer h.Unlock()
	log.Debug("Hatchery has %d processes in index", len(h.workers))

	for name, cmd := range h.workers {
		for _, n := range apiworkers {
			// If worker is disabled, kill it
			if n.Name == name && n.Status == sdk.StatusDisabled {
				log.Debug("Worker %s is disabled. Kill it with fire !", name)

				// if process not killed, kill it
				if cmd.ProcessState == nil || (cmd.ProcessState != nil && !cmd.ProcessState.Exited()) {
					err = cmd.Process.Kill()
					if err != nil {
						log.Warning("HatcheryDocker.killAwolWorker: cannot kill %s: %s", name, err)
					}
				}

				// Remove container
				go func() {
					cmd := exec.Command("docker", "rm", "-f", name)
					err = cmd.Run()
					if err != nil {
						log.Warning("HatcheryDocker.killAwolWorker: cannot rm container %s: %s", name, err)
					}
				}()

				delete(h.workers, name)
				log.Info("HatcheryDocker.killAwolWorker> Killed disabled worker %s", name)
				return
			}
		}
	}
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryDocker) WorkersStarted() int {
	return len(h.workers)
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryDocker) WorkersStartedByModel(model *sdk.Model) int {
	var x int
	for name := range h.workers {
		if strings.Contains(name, model.Name) {
			x++
		}
	}
	return x
}

// SpawnWorker starts a new worker in a docker container locally
func (h *HatcheryDocker) SpawnWorker(wm *sdk.Model, jobID int64, requirements []sdk.Requirement, registerOnly bool, logInfo string) (string, error) {
	if wm.Type != sdk.Docker {
		return "", fmt.Errorf("cannot handle %s worker model", wm.Type)
	}

	if len(h.workers) >= viper.GetInt("max-worker") {
		return "", fmt.Errorf("Max capacity reached (%d)", viper.GetInt("max-worker"))
	}

	if jobID > 0 {
		log.Info("spawnWorker> spawning worker %s (%s) for job %d - %s", wm.Name, wm.Image, jobID, logInfo)
	} else {
		log.Info("spawnWorker> spawning worker %s (%s) - %s", wm.Name, wm.Image, logInfo)
	}

	name, errs := randSeq(16)
	if errs != nil {
		return "", fmt.Errorf("cannot create worker name: %s", errs)
	}
	name = wm.Name + "-" + name
	if registerOnly {
		name = "register-" + name
	}

	var args []string
	args = append(args, "run", "--rm", "-a", "STDOUT", "-a", "STDERR")
	args = append(args, fmt.Sprintf("--name=%s", name))
	args = append(args, "-e", "CDS_SINGLE_USE=1")
	args = append(args, "-e", fmt.Sprintf("CDS_API=%s", h.Client().APIURL()))
	args = append(args, "-e", fmt.Sprintf("CDS_NAME=%s", name))
	args = append(args, "-e", fmt.Sprintf("CDS_TOKEN=%s", viper.GetString("token")))
	args = append(args, "-e", fmt.Sprintf("CDS_MODEL=%d", wm.ID))
	args = append(args, "-e", fmt.Sprintf("CDS_HATCHERY=%d", h.hatch.ID))
	args = append(args, "-e", fmt.Sprintf("CDS_HATCHERY_NAME=%s", h.hatch.Name))

	if viper.GetString("worker_graylog_host") != "" {
		args = append(args, "-e", fmt.Sprintf("CDS_GRAYLOG_HOST=%s", viper.GetString("worker_graylog_host")))
	}
	if viper.GetString("worker_graylog_port") != "" {
		args = append(args, "-e", fmt.Sprintf("CDS_GRAYLOG_PORT=%s", viper.GetString("worker_graylog_port")))
	}
	if viper.GetString("worker_graylog_extra_key") != "" {
		args = append(args, "-e", fmt.Sprintf("CDS_GRAYLOG_EXTRA_KEY=%s", viper.GetString("worker_graylog_extra_key")))
	}
	if viper.GetString("worker_graylog_extra_value") != "" {
		args = append(args, "-e", fmt.Sprintf("CDS_GRAYLOG_EXTRA_VALUE=%s", viper.GetString("worker_graylog_extra_value")))
	}
	if viper.GetString("grpc_api") != "" && wm.Communication == sdk.GRPC {
		args = append(args, "-e", fmt.Sprintf("CDS_GRPC_API=%s", viper.GetString("grpc_api")))
		args = append(args, "-e", fmt.Sprintf("CDS_GRPC_INSECURE=%t", viper.GetBool("grpc_insecure")))
	}

	if jobID > 0 {
		args = append(args, "-e", fmt.Sprintf("CDS_BOOKED_JOB_ID=%d", jobID))
	}

	if h.addhost != "" {
		args = append(args, fmt.Sprintf("--add-host=%s", h.addhost))
	}
	args = append(args, wm.Image)
	args = append(args, "sh", "-c", fmt.Sprintf("rm -f worker && echo 'Download worker' && curl %s/download/worker/`uname -m` -o worker && echo 'chmod worker' && chmod +x worker && echo 'starting worker' && ./worker", h.Client().APIURL()))

	if registerOnly {
		args = append(args, "register")
	}
	cmd := exec.Command("docker", args...)
	log.Debug("Running %s", cmd.Args)

	if err := cmd.Start(); err != nil {
		return "", err
	}
	h.Lock()
	h.workers[name] = cmd
	h.Unlock()

	// Wait in a goroutine so that when process exits, Wait() update cmd.ProcessState
	// ProcessState is then checked in nextAvailableLocalID
	go func() {
		cmd.Wait()
	}()

	// Do not spam docker daemon
	time.Sleep(2 * time.Second)
	return name, nil
}

func randSeq(n int) (string, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	ex := hex.EncodeToString(b)
	sized := []byte(ex)[0:n]
	return string(sized), nil
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryDocker) NeedRegistration(m *sdk.Model) bool {
	if m.NeedRegistration || m.LastRegistration.Unix() < m.UserLastModified.Unix() {
		return true
	}
	return false
}
