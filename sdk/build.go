package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// PipelineBuildJob represents an action to be run
type PipelineBuildJob struct {
	ID              int64       `json:"id" db:"id"`
	Job             ExecutedJob `json:"job" db:"-"`
	Parameters      []Parameter `json:"parameters,omitempty" db:"-"`
	Status          string      `json:"status"  db:"status"`
	Queued          time.Time   `json:"queued,omitempty" db:"queued"`
	QueuedSeconds   int64       `json:"queued_seconds,omitempty" db:"-"`
	Start           time.Time   `json:"start,omitempty" db:"start"`
	Done            time.Time   `json:"done,omitempty" db:"done"`
	Model           string      `json:"model,omitempty" db:"model"`
	PipelineBuildID int64       `json:"pipeline_build_id,omitempty" db:"pipeline_build_id"`
	BookedBy        Hatchery    `json:"bookedby" db:"-"`
	SpawnInfos      []SpawnInfo `json:"spawninfos" db:"-"`
}

// SpawnInfo contains an information about spawning
type SpawnInfo struct {
	APITime    time.Time `json:"api_time,omitempty" db:"-"`
	RemoteTime time.Time `json:"remote_time,omitempty" db:"-"`
	Message    SpawnMsg  `json:"message,omitempty" db:"-"`
	// UserMessage contains msg translated for end user
	UserMessage string `json:"user_message,omitempty" db:"-"`
}

// SpawnMsg represents a msg for spawnInfo
type SpawnMsg struct {
	ID   string        `json:"id" db:"-"`
	Args []interface{} `json:"args" db:"-"`
}

// ExecutedJob represents a running job
type ExecutedJob struct {
	Job
	StepStatus []StepStatus `json:"step_status" db:"-"`
	Reason     string       `json:"reason" db:"-"`
	WorkerName string       `json:"worker_name" db:"-"`
}

// StepStatus Represent a step and his status
type StepStatus struct {
	StepOrder int    `json:"step_order" db:"-"`
	Status    string `json:"status" db:"-"`
}

// BuildState define struct returned when looking for build state informations
type BuildState struct {
	Stages   []Stage `json:"stages"`
	Logs     []Log   `json:"logs"`
	StepLogs Log     `json:"step_logs"`
	Status   Status  `json:"status"`
}

// Status reprensents a Build Action or Build Pipeline Status
type Status string

// StatusFromString returns a Status from a given string
func StatusFromString(in string) Status {
	switch in {
	case StatusWaiting.String():
		return StatusWaiting
	case StatusBuilding.String():
		return StatusBuilding
	case StatusChecking.String():
		return StatusChecking
	case StatusSuccess.String():
		return StatusSuccess
	case StatusNeverBuilt.String():
		return StatusNeverBuilt
	case StatusFail.String():
		return StatusFail
	case StatusDisabled.String():
		return StatusDisabled
	case StatusSkipped.String():
		return StatusSkipped
	default:
		return StatusUnknown
	}
}

func (t Status) String() string {
	return string(t)
}

// Action status in queue
const (
	StatusWaiting    Status = "Waiting"
	StatusChecking   Status = "Checking"
	StatusBuilding   Status = "Building"
	StatusSuccess    Status = "Success"
	StatusFail       Status = "Fail"
	StatusDisabled   Status = "Disabled"
	StatusNeverBuilt Status = "Never Built"
	StatusUnknown    Status = "Unknown"
	StatusSkipped    Status = "Skipped"
)

// GetBuildQueue retrieves current CDS build in queue
func GetBuildQueue() ([]PipelineBuildJob, error) {
	var q []PipelineBuildJob

	path := "/queue?status=all"

	data, code, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	if err = json.Unmarshal(data, &q); err != nil {
		return nil, err
	}

	return q, nil
}

// GetBuildState Get the state of given build
func GetBuildState(projectKey, appName, pipelineName, env, buildID string) (PipelineBuild, error) {
	var buildState PipelineBuild

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s?envName=%s", projectKey, appName, pipelineName, buildID, env)
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return buildState, err
	}
	if code >= 300 {
		return buildState, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, &buildState)
	if err != nil {
		return buildState, err
	}
	return buildState, nil
}

// GetBuildActionLog Get the log of the given action for the given build
func GetBuildActionLog(projectKey, appName, pipelineName, buildID, pipelineActionID string) (BuildState, error) {
	var buildState BuildState

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s/action/%s/log", projectKey, appName, pipelineName, buildID, pipelineActionID)
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return buildState, err
	}
	if code >= 300 {
		return buildState, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, &buildState)
	if err != nil {
		return buildState, err
	}
	return buildState, nil
}

// Translate translates messages in pipelineBuildJob
func (p *PipelineBuildJob) Translate(lang string) {
	for ki, info := range p.SpawnInfos {
		m := NewMessage(Messages[info.Message.ID], info.Message.Args...)
		p.SpawnInfos[ki].UserMessage = m.String(lang)
	}

}
