package hatchery

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// CommonConfiguration is the base configuration for all hatcheries
type CommonConfiguration struct {
	Name string `toml:"name" default:"" comment:"Name of Hatchery"`
	API  struct {
		HTTP struct {
			URL      string `toml:"url" default:"http://localhost:8081" commented:"true" comment:"CDS API URL"`
			Insecure bool   `toml:"insecure" default:"false" commented:"true" comment:"sslInsecureSkipVerify, set to true if you use a self-signed SSL on CDS API"`
		} `toml:"http"`
		GRPC struct {
			URL      string `toml:"url" default:"http://localhost:8082" commented:"true"`
			Insecure bool   `toml:"insecure" default:"false" commented:"true" comment:"sslInsecureSkipVerify, set to true if you use a self-signed SSL on CDS API"`
		} `toml:"grpc"`
		Token                string `toml:"token" default:"" comment:"CDS Token to reach CDS API. See https://ovh.github.io/cds/advanced/advanced.worker.token/ "`
		RequestTimeout       int    `toml:"requestTimeout" default:"10" comment:"Request CDS API: timeout in seconds"`
		MaxHeartbeatFailures int    `toml:"maxHeartbeatFailures" default:"10" comment:"Maximum allowed consecutives failures on heatbeat routine"`
	} `toml:"api"`
	Provision struct {
		Disabled          bool `toml:"disabled" default:"false" comment:"Disabled provisionning. Format:true or false"`
		Frequency         int  `toml:"frequency" default:"30" comment:"Check provisioning each n Seconds"`
		MaxWorker         int  `toml:"maxWorker" default:"10" comment:"Maximum allowed simultaneous workers"`
		GraceTimeQueued   int  `toml:"graceTimeQueued" default:"4" comment:"if worker is queued less than this value (seconds), hatchery does not take care of it"`
		RegisterFrequency int  `toml:"registerFrequency" default:"60" comment:"Check if some worker model have to be registered each n Seconds"`
		WorkerLogsOptions struct {
			Graylog struct {
				Host       string `toml:"host"`
				Port       int    `toml:"port"`
				Protocol   string `toml:"protocol"`
				ExtraKey   string `toml:"extraKey"`
				ExtraValue string `toml:"extraValue"`
			} `toml:"graylog"`
		} `toml:"workerLogsOptions" comment:"Worker Log Configuration"`
	} `toml:"provision"`
	LogOptions struct {
		SpawnOptions struct {
			ThresholdCritical int `toml:"thresholdCritical" default:"480" comment:"log critical if spawn take more than this value (in seconds)"`
			ThresholdWarning  int `toml:"thresholdWarning" default:"360" comment:"log warning if spawn take more than this value (in seconds)"`
		} `toml:"spawnOptions"`
	} `toml:"logOptions" comment:"Hatchery Log Configuration"`
}

// SpawnArguments contains arguments to func SpawnWorker
type SpawnArguments struct {
	Model         sdk.Model
	IsWorkflowJob bool
	JobID         int64
	Requirements  []sdk.Requirement
	RegisterOnly  bool
	LogInfo       string
}

// Interface describe an interface for each hatchery mode
// Init create new clients for different api
// SpawnWorker creates a new vm instance
// CanSpawn return wether or not hatchery can spawn model
// WorkersStartedByModel returns the number of instances of given model started but not necessarily register on CDS yet
// WorkersStarted returns the number of instances started but not necessarily register on CDS yet
// Hatchery returns hatchery instance
// Client returns cdsclient instance
// ModelType returns type of hatchery
// NeedRegistration return true if worker model need regsitration
// ID returns hatchery id
type Interface interface {
	Init() error
	SpawnWorker(spawnArgs SpawnArguments) (string, error)
	CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool
	WorkersStartedByModel(model *sdk.Model) int
	WorkersStarted() int
	Hatchery() *sdk.Hatchery
	Client() cdsclient.Interface
	Configuration() CommonConfiguration
	ModelType() string
	NeedRegistration(model *sdk.Model) bool
	ID() int64
	Serve(ctx context.Context) error
}

var (
	// Client is a CDS Client
	Client sdk.HTTPClient
)

// CheckRequirement checks binary requirement in path
func CheckRequirement(r sdk.Requirement) (bool, error) {
	switch r.Type {
	case sdk.BinaryRequirement:
		if _, err := exec.LookPath(r.Value); err != nil {
			// Return nil because the error contains 'Exit status X', that's what we wanted
			return false, nil
		}
		return true, nil
	default:
		return false, nil
	}
}

func receiveJob(h Interface, isWorkflowJob bool, execGroups []sdk.Group, jobID int64, jobQueuedSeconds int64, jobBookedBy sdk.Hatchery, requirements []sdk.Requirement, models []sdk.Model, nRoutines *int64, spawnIDs *cache.Cache, hostname string) bool {
	if jobID == 0 {
		return false
	}

	n := atomic.LoadInt64(nRoutines)
	if n > 10 {
		log.Info("too many routines in same time %d", n)
		return false
	}

	if _, exist := spawnIDs.Get(string(jobID)); exist {
		log.Debug("job %d already spawned in previous routine", jobID)
		return false
	}

	if jobQueuedSeconds < int64(h.Configuration().Provision.GraceTimeQueued) {
		log.Debug("job %d is too fresh, queued since %d seconds, let existing waiting worker check it", jobID, jobQueuedSeconds)
		return false
	}

	log.Debug("work on job %d queued since %d seconds", jobID, jobQueuedSeconds)
	if jobBookedBy.ID != 0 {
		t := "current hatchery"
		if jobBookedBy.ID != h.Hatchery().ID {
			t = "another hatchery"
		}
		log.Debug("job %d already booked by %s %s (%d)", jobID, t, jobBookedBy.Name, jobBookedBy.ID)
		return false
	}

	atomic.AddInt64(nRoutines, 1)
	defer atomic.AddInt64(nRoutines, -1)
	isSpawned, errR := routine(h, isWorkflowJob, models, execGroups, jobID, requirements, hostname, time.Now().Unix())
	if errR != nil {
		log.Warning("Error on routine: %s", errR)
		return false
	}
	return isSpawned
}

func routine(h Interface, isWorkflowJob bool, models []sdk.Model, execGroups []sdk.Group, jobID int64, requirements []sdk.Requirement, hostname string, timestamp int64) (bool, error) {
	defer logTime(h, fmt.Sprintf("routine> %d", timestamp), time.Now())
	log.Debug("routine> %d enter", timestamp)

	if h.Hatchery() == nil || h.Hatchery().ID == 0 {
		log.Debug("Create> continue")
		return false, nil
	}

	if len(models) == 0 {
		return false, fmt.Errorf("routine> %d - No model returned by CDS api", timestamp)
	}
	log.Debug("routine> %d - models received: %d", timestamp, len(models))

	for _, model := range models {
		if canRunJob(h, timestamp, execGroups, jobID, requirements, &model, hostname) {
			if err := h.Client().QueueJobBook(isWorkflowJob, jobID); err != nil {
				// perhaps already booked by another hatchery
				log.Debug("routine> %d - cannot book job %d %s: %s", timestamp, jobID, model.Name, err)
				break // go to next job
			}
			log.Debug("routine> %d - send book job %d %s by hatchery %d isWorkflowJob:%t", timestamp, jobID, model.Name, h.Hatchery().ID, isWorkflowJob)

			start := time.Now()
			infos := []sdk.SpawnInfo{
				{
					RemoteTime: start,
					Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStarts.ID, Args: []interface{}{fmt.Sprintf("%s", h.Hatchery().Name), fmt.Sprintf("%d", h.Hatchery().ID), model.Name}},
				},
			}
			workerName, errSpawn := h.SpawnWorker(SpawnArguments{Model: model, IsWorkflowJob: isWorkflowJob, JobID: jobID, Requirements: requirements, LogInfo: "spawn for job"})
			if errSpawn != nil {
				log.Warning("routine> %d - cannot spawn worker %s for job %d: %s", timestamp, model.Name, jobID, errSpawn)
				infos = append(infos, sdk.SpawnInfo{
					RemoteTime: time.Now(),
					Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryErrorSpawn.ID, Args: []interface{}{fmt.Sprintf("%s", h.Hatchery().Name), fmt.Sprintf("%d", h.Hatchery().ID), model.Name, sdk.Round(time.Since(start), time.Second).String(), errSpawn.Error()}},
				})
				if err := h.Client().QueueJobSendSpawnInfo(isWorkflowJob, jobID, infos); err != nil {
					log.Warning("routine> %d - cannot client.QueueJobSendSpawnInfo for job (err spawn)%d: %s", timestamp, jobID, err)
				}
				if err := h.Client().WorkerModelSpawnError(model.ID, fmt.Sprintf("routine> cannot spawn worker %s for job %d: %s", model.Name, jobID, errSpawn)); err != nil {
					log.Error("routine> error on call client.WorkerModelSpawnError on worker model %s for register: %s", model.Name, errSpawn)
				}
				continue // try another model
			}

			infos = append(infos, sdk.SpawnInfo{
				RemoteTime: time.Now(),
				Message: sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStartsSuccessfully.ID,
					Args: []interface{}{
						fmt.Sprintf("%s", h.Hatchery().Name),
						fmt.Sprintf("%d", h.Hatchery().ID),
						fmt.Sprintf("%s", workerName),
						sdk.Round(time.Since(start), time.Second).String()},
				},
			})

			if err := h.Client().QueueJobSendSpawnInfo(isWorkflowJob, jobID, infos); err != nil {
				log.Warning("routine> %d - cannot client.QueueJobSendSpawnInfo for job %d: %s", timestamp, jobID, err)
			}
			return true, nil // ok for this job
		}
	}

	return false, nil
}

func provisioning(h Interface, provisionDisabled bool, models []sdk.Model) {
	if provisionDisabled {
		log.Debug("provisioning> disabled on this hatchery")
		return
	}

	for k := range models {
		if models[k].Type == h.ModelType() {
			existing := h.WorkersStartedByModel(&models[k])
			for i := existing; i < int(models[k].Provision); i++ {
				go func(m sdk.Model) {
					if name, errSpawn := h.SpawnWorker(SpawnArguments{Model: m, IsWorkflowJob: false, JobID: 0, Requirements: nil, LogInfo: "spawn for provision"}); errSpawn != nil {
						log.Warning("provisioning> cannot spawn worker %s with model %s for provisioning: %s", name, m.Name, errSpawn)
						if err := h.Client().WorkerModelSpawnError(m.ID, fmt.Sprintf("routine> cannot spawn worker %s for provisioning: %s", m.Name, errSpawn)); err != nil {
							log.Error("provisioning> cannot client.WorkerModelSpawnError for worker %s with model %s for provisioning: %s", name, m.Name, errSpawn)
						}
					}
				}(models[k])
			}
		}
	}
}

func canRunJob(h Interface, timestamp int64, execGroups []sdk.Group, jobID int64, requirements []sdk.Requirement, model *sdk.Model, hostname string) bool {
	if model.Type != h.ModelType() {
		return false
	}

	// if current hatchery is in same group than worker model -> do not avoid spawn, even if worker model is in error
	if model.NbSpawnErr > 5 && h.Hatchery().GroupID != model.ID {
		log.Warning("canRunJob> Too many errors on spawn with model %s, please check this worker model", model.Name)
		return false
	}

	if execGroups != nil && len(execGroups) > 0 {
		checkGroup := false
		for _, g := range execGroups {
			if g.ID == model.GroupID {
				checkGroup = true
				break
			}
		}
		if !checkGroup {
			log.Debug("canRunJob> %d - job %d - model %s attached to group %d can't run this job", timestamp, jobID, model.Name, model.GroupID)
			return false
		}
	}

	var containsModelRequirement, containsHostnameRequirement bool
	for _, r := range requirements {
		switch r.Type {
		case sdk.ModelRequirement:
			containsModelRequirement = true
		case sdk.HostnameRequirement:
			containsHostnameRequirement = true
		}
	}
	// Common check
	for _, r := range requirements {
		// If requirement is a Model requirement, it's easy. It's either can or can't run
		// r.Value could be: theModelName --port=8888:9999, so we take strings.Split(r.Value, " ")[0] to compare
		// only modelName
		if r.Type == sdk.ModelRequirement && strings.Split(r.Value, " ")[0] != model.Name {
			log.Debug("canRunJob> %d - job %d - model requirement r.Value(%s) != model.Name(%s)", timestamp, jobID, strings.Split(r.Value, " ")[0], model.Name)
			return false
		}

		// If requirement is an hostname requirement, it's for a specific worker
		if r.Type == sdk.HostnameRequirement && r.Value != hostname {
			log.Debug("canRunJob> %d - job %d - hostname requirement r.Value(%s) != hostname(%s)", timestamp, jobID, r.Value, hostname)
			return false
		}

		// service and memory requirements are only supported by docker model
		if model.Type != sdk.Docker && (r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement) {
			log.Debug("canRunJob> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", timestamp, jobID, model.Type)
			return false
		}

		// Skip network access requirement as we can't check it
		if r.Type == sdk.NetworkAccessRequirement || r.Type == sdk.PluginRequirement || r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			log.Debug("canRunJob> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", timestamp, jobID, model.Type)
			continue
		}

		if !containsModelRequirement && !containsHostnameRequirement {
			if r.Type == sdk.BinaryRequirement {
				found := false
				// Check binary requirement against worker model capabilities
				for _, c := range model.Capabilities {
					if r.Value == c.Value || r.Value == c.Name {
						found = true
						break
					}
				}

				if !found {
					log.Debug("canRunJob> %d - job %d - model(%s) does not have binary %s(%s) for this job.", timestamp, jobID, model.Name, r.Name, r.Value)
					return false
				}
			}
		}
	}

	return h.CanSpawn(model, jobID, requirements)
}

func logTime(h Interface, name string, then time.Time) {
	d := time.Since(then)
	if d > time.Duration(h.Configuration().LogOptions.SpawnOptions.ThresholdCritical)*time.Second {
		log.Error("%s took %s to execute", name, d)
		return
	}

	if d > time.Duration(h.Configuration().LogOptions.SpawnOptions.ThresholdWarning)*time.Second {
		log.Warning("%s took %s to execute", name, d)
		return
	}

	log.Debug("%s took %s to execute", name, d)
}
