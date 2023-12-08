package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.opencensus.io/stats"
)

type AuthConsumerHatcherySigninRequest struct {
	Token        string        `json:"token"`
	Name         string        `json:"name"`
	HatcheryType string        `json:"type"`
	HTTPURL      string        `json:"http_url"`
	Config       ServiceConfig `json:"config" db:"config" cli:"-" mapstructure:"config"`
	PublicKey    []byte        `json:"public_key"`
	Version      string        `json:"version"`
}

type AuthConsumerHatcherySigninResponse struct {
	Uptodate bool     `json:"up_to_date"`
	APIURL   string   `json:"api_url"`
	Token    string   `json:"token"`
	Hatchery Hatchery `json:"hatchery"`
	Region   string   `json:"region"`
}

type HatcheryStatus struct {
	ID         int64            `json:"id" db:"id" cli:"id,key"`
	HatcheryID string           `json:"hatchery_id" db:"hatchery_id" cli:"hatchery_id"`
	SessionID  string           `json:"session_id" db:"session_id" cli:"session_id"`
	Status     MonitoringStatus `json:"monitoring_status" db:"monitoring_status"`
}

type Hatchery struct {
	ID            string        `json:"id" db:"id" cli:"id,key"`
	Name          string        `json:"name" db:"name" cli:"name"`
	ModelType     string        `json:"model_type" db:"model_type" cli:"model_type"`
	Config        ServiceConfig `json:"config" db:"config"`
	LastHeartbeat time.Time     `json:"last_heartbeat,omitempty" db:"last_heartbeat" cli:"last_heartbeat"`
	PublicKey     []byte        `json:"public_key" db:"public_key"`
	HTTPURL       string        `json:"http_url" db:"http_url"`

	// On signup / regen
	Token string `json:"token,omitempty" db:"-" cli:"token,omitempty"`
}

type HatcheryMetrics struct {
	Jobs                          *stats.Int64Measure
	JobsWebsocket                 *stats.Int64Measure
	JobsProcessed                 *stats.Int64Measure
	SpawningWorkers               *stats.Int64Measure
	SpawnedWorkers                *stats.Int64Measure
	SpawningWorkersErrors         *stats.Int64Measure
	JobReceivedInQueuePollingWSv1 *stats.Int64Measure
	JobReceivedInQueuePollingWSv2 *stats.Int64Measure
	ChanV1JobAdd                  *stats.Int64Measure
	ChanV2JobAdd                  *stats.Int64Measure
	ChanWorkerStarterPop          *stats.Int64Measure
	PendingWorkers                *stats.Int64Measure
	RegisteringWorkers            *stats.Int64Measure
	CheckingWorkers               *stats.Int64Measure
	WaitingWorkers                *stats.Int64Measure
	BuildingWorkers               *stats.Int64Measure
	DisabledWorkers               *stats.Int64Measure
}

type HatcheryPendingWorkerCreation struct {
	mapSpawnJobRequest      map[string]struct{}
	mapSpawnJobRequestMutex *sync.Mutex
}

func (c *HatcheryPendingWorkerCreation) Init() {
	c.mapSpawnJobRequest = make(map[string]struct{})
	c.mapSpawnJobRequestMutex = new(sync.Mutex)
}

func (c *HatcheryPendingWorkerCreation) SetJobInPendingWorkerCreation(id string) {
	c.mapSpawnJobRequestMutex.Lock()
	c.mapSpawnJobRequest[id] = struct{}{}
	c.mapSpawnJobRequestMutex.Unlock()
}

func (c *HatcheryPendingWorkerCreation) RemoveJobFromPendingWorkerCreation(id string) {
	c.mapSpawnJobRequestMutex.Lock()
	delete(c.mapSpawnJobRequest, id)
	c.mapSpawnJobRequestMutex.Unlock()
}

func (c *HatcheryPendingWorkerCreation) IsJobAlreadyPendingWorkerCreation(id string) bool {
	c.mapSpawnJobRequestMutex.Lock()
	_, has := c.mapSpawnJobRequest[id]
	c.mapSpawnJobRequestMutex.Unlock()
	return has
}

type HatcheryConfig map[string]interface{}

func (hc HatcheryConfig) Value() (driver.Value, error) {
	j, err := json.Marshal(hc)
	return j, WrapError(err, "cannot marshal HatcheryConfig")
}

func (hc *HatcheryConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, hc), "cannot unmarshal HatcheryConfig")
}

type WorkerStarterWorkerModel struct {
	ModelV1 *Model

	// Worker model v2
	ModelV2       *V2WorkerModel
	PreCmd        string
	Cmd           string
	Shell         string
	PostCmd       string
	DockerSpec    V2WorkerModelDockerSpec
	OpenstackSpec V2WorkerModelOpenstackSpec
	VSphereSpec   V2WorkerModelVSphereSpec
	Commit        string
}

func (w WorkerStarterWorkerModel) GetName() string {
	if w.ModelV1 != nil {
		return w.ModelV1.Name
	} else {
		return w.ModelV2.Name
	}
}

func (w WorkerStarterWorkerModel) GetFlavor(reqs RequirementList, defaultFlavor string) string {
	switch {
	case w.ModelV1 != nil:
		if w.ModelV1.ModelVirtualMachine.Flavor != "" {
			return w.ModelV1.ModelVirtualMachine.Flavor
		}
	case w.ModelV2 != nil:
		for _, r := range reqs {
			if r.Type == FlavorRequirement && r.Value != "" {
				return r.Value
			}
		}
	}
	return defaultFlavor
}

func (w WorkerStarterWorkerModel) GetOpenstackImage() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelVirtualMachine.Image
	case w.ModelV2 != nil:
		return w.OpenstackSpec.Image
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetDockerImage() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelDocker.Image
	case w.ModelV2 != nil:
		return w.DockerSpec.Image
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetShell() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelDocker.Shell
	case w.ModelV2 != nil:
		return w.Shell
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetCmd() string {
	switch {
	case w.ModelV1 != nil:
		if w.ModelV1.ModelVirtualMachine.Cmd != "" {
			return w.ModelV1.ModelVirtualMachine.Cmd
		}
		return w.ModelV1.ModelDocker.Cmd
	case w.ModelV2 != nil:
		return w.Cmd
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetPreCmd() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelVirtualMachine.PreCmd
	case w.ModelV2 != nil:
		return w.PreCmd
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetPostCmd() string {
	switch {
	case w.ModelV1 != nil:
		if w.ModelV1.ModelVirtualMachine.PostCmd != "" {
			return w.ModelV1.ModelVirtualMachine.PostCmd
		}
	case w.ModelV2 != nil:
		return w.PostCmd
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetDockerEnvs() map[string]string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelDocker.Envs
	case w.ModelV2 != nil:
		return w.DockerSpec.Envs
	}
	return nil
}

func (w WorkerStarterWorkerModel) GetDockerUsername() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelDocker.Username
	case w.ModelV2 != nil:
		return w.DockerSpec.Username
	}
	return ""
}

func (w WorkerStarterWorkerModel) IsPrivate() bool {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelDocker.Private
	case w.ModelV2 != nil:
		return false
	}
	return false
}

func (w WorkerStarterWorkerModel) GetDockerPassword() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelDocker.Password
	case w.ModelV2 != nil:
		return w.DockerSpec.Password
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetPath() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.Path()
	case w.ModelV2 != nil:
		return w.ModelV2.Name
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetFullPath() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.Group.Name + "/" + w.ModelV1.Name
	case w.ModelV2 != nil:
		return w.ModelV2.Name
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetVSphereImage() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelVirtualMachine.Image
	case w.ModelV2 != nil:
		return w.VSphereSpec.Image
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetVSphereUsername() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelVirtualMachine.User
	case w.ModelV2 != nil:
		return w.VSphereSpec.Username
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetVSpherePassword() string {
	switch {
	case w.ModelV1 != nil:
		return w.ModelV1.ModelVirtualMachine.Password
	case w.ModelV2 != nil:
		return w.VSphereSpec.Password
	}
	return ""
}

func (w WorkerStarterWorkerModel) GetLastModified() string {
	switch {
	case w.ModelV1 != nil:
		return fmt.Sprintf("%d", w.ModelV1.UserLastModified.Unix())
	case w.ModelV2 != nil:
		return w.Commit
	}
	return ""
}

func IsJobIDForRegister(jobID string) bool {
	if IsValidUUID(jobID) {
		return false
	}
	jobIDint, _ := strconv.Atoi(jobID)
	return jobIDint == 0 || jobIDint < 0
}
