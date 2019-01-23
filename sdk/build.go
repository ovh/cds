package sdk

import (
	"time"
)

// SpawnInfo contains an information about spawning
type SpawnInfo struct {
	APITime    time.Time `json:"api_time,omitempty" db:"-" mapstructure:"-"`
	RemoteTime time.Time `json:"remote_time,omitempty" db:"-" mapstructure:"-"`
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
	WorkerID   string       `json:"worker_id" db:"-"`
}

// ExecutedJobSummary is a light representation of ExecutedJob for CDS event
type ExecutedJobSummary struct {
	StepStatusSummary []StepStatusSummary `json:"step_status"`
	Reason            string              `json:"reason"`
	WorkerName        string              `json:"worker_name"`
	WorkerID          string              `json:"worker_id"`
	JobName           string              `json:"job_name"`
	PipelineActionID  int64               `json:"pipeline_action_id"`
	PipelineStageID   int64               `json:"pipeline_stage_id"`
	Steps             []ActionSummary     `json:"steps"`
}

// ToSummary transforms an ExecutedJob to an ExecutedJobSummary
func (j ExecutedJob) ToSummary() ExecutedJobSummary {
	sum := ExecutedJobSummary{
		JobName:          j.Action.Name,
		Reason:           j.Reason,
		WorkerName:       j.WorkerName,
		PipelineActionID: j.PipelineActionID,
		PipelineStageID:  j.PipelineStageID,
	}
	sum.StepStatusSummary = make([]StepStatusSummary, len(j.StepStatus))
	for i := range j.StepStatus {
		sum.StepStatusSummary[i] = j.StepStatus[i].ToSummary()
	}

	sum.Steps = make([]ActionSummary, len(j.Action.Actions))
	for i := range j.Action.Actions {
		sum.Steps[i] = j.Action.Actions[i].ToSummary()
	}

	return sum
}

// StepStatus Represent a step and his status
type StepStatus struct {
	StepOrder int       `json:"step_order" db:"-"`
	Status    string    `json:"status" db:"-"`
	Start     time.Time `json:"start" db:"-"`
	Done      time.Time `json:"done" db:"-"`
}

// StepStatusSummary Represent a step and his status for CDS event
type StepStatusSummary struct {
	StepOrder int    `json:"step_order" db:"-"`
	Status    string `json:"status" db:"-"`
	Start     int64  `json:"start" db:"-"`
	Done      int64  `json:"done" db:"-"`
}

// ToSummary transform a StepStatus into a StepStatusSummary
func (ss StepStatus) ToSummary() StepStatusSummary {
	return StepStatusSummary{
		Start:     ss.Start.Unix(),
		StepOrder: ss.StepOrder,
		Status:    ss.Status,
		Done:      ss.Done.Unix(),
	}
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
	case StatusStopped.String():
		return StatusStopped
	case StatusWorkerPending.String():
		return StatusWorkerPending
	case StatusWorkerRegistering.String():
		return StatusWorkerRegistering
	default:
		return StatusUnknown
	}
}

func (t Status) String() string { return string(t) }

// StatusToStrings returns []string from given []status.
func StatusToStrings(ss []Status) []string {
	res := make([]string, len(ss))
	for i, s := range ss {
		res[i] = s.String()
	}
	return res
}

// Action status in queue
const (
	StatusWaiting           Status = "Waiting"
	StatusChecking          Status = "Checking" // DEPRECATED, to remove when removing pipelineBuild
	StatusBuilding          Status = "Building"
	StatusSuccess           Status = "Success"
	StatusFail              Status = "Fail"
	StatusDisabled          Status = "Disabled"
	StatusNeverBuilt        Status = "Never Built"
	StatusUnknown           Status = "Unknown"
	StatusSkipped           Status = "Skipped"
	StatusStopped           Status = "Stopped"
	StatusWorkerPending     Status = "Pending"
	StatusWorkerRegistering Status = "Registering"
)

// StatusIsTerminated returns if status is terminated (nothing related to building or waiting, ...)
func StatusIsTerminated(status string) bool {
	switch status {
	case StatusBuilding.String(), StatusWaiting.String():
		return false
	default:
		return true
	}
}

// StatusValidate returns if given strings are valid status.
func StatusValidate(status ...string) bool {
	for _, s := range status {
		if StatusFromString(s) == StatusUnknown {
			return false
		}
	}
	return true
}
