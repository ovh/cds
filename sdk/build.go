package sdk

import (
	"fmt"
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
	ID   string        `json:"id,omitempty" db:"-"`
	Args []interface{} `json:"args,omitempty" db:"-"`
	Type string        `json:"type,omitempty" db:"-"`
}

func SpawnMsgNew(msg Message, args ...interface{}) SpawnMsg {
	for i := range args {
		args[i] = fmt.Sprintf("%v", args[i])
	}
	return SpawnMsg{
		ID:   msg.ID,
		Type: msg.Type,
		Args: args,
	}
}

func (s SpawnMsg) DefaultUserMessage() string {
	if _, ok := Messages[s.ID]; ok {
		m := Messages[s.ID]
		return fmt.Sprintf(m.Format[EN], s.Args...)
	}
	return ""
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

// Action status in queue
const (
	StatusPending           = "Pending"
	StatusWaiting           = "Waiting"
	StatusChecking          = "Checking" // DEPRECATED, to remove when removing pipelineBuild
	StatusBuilding          = "Building"
	StatusSuccess           = "Success"
	StatusFail              = "Fail"
	StatusDisabled          = "Disabled"
	StatusNeverBuilt        = "Never Built"
	StatusUnknown           = "Unknown"
	StatusSkipped           = "Skipped"
	StatusStopped           = "Stopped"
	StatusWorkerPending     = "Pending"
	StatusWorkerRegistering = "Registering"

	StatusCrafting = "Crafting"
)

var (
	StatusTerminated    = True
	StatusNotTerminated = False
)

// StatusIsTerminated returns if status is terminated (nothing related to building or waiting, ...)
func StatusIsTerminated(status string) bool {
	switch status {
	case StatusPending, StatusBuilding, StatusWaiting, "": // A stage does not have status when he's waiting a previous stage
		return false
	default:
		return true
	}
}

// StatusValidate returns if given strings are valid status.
func StatusValidate(status ...string) bool {
	for _, s := range status {
		if s == StatusUnknown {
			return false
		}
	}
	return true
}
