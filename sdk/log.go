package sdk

import (
	"time"
)

// Log struct holds a single line of build log
type Log struct {
	ID              int64     `json:"id"`
	ActionBuildID   int64     `json:"action_build_id"`
	PipelineBuildID int64     `json:"pipeline_build_id"`
	Timestamp       time.Time `json:"timestamp"`
	Step            string    `json:"step"`
	Value           string    `json:"value"`
	StepOrder       int       `json:"step_order"`
}

// NewLog returns a log struct
func NewLog(buildid int64, step string, value string, pipelineBuildID int64, stepOrder int) *Log {
	l := &Log{
		ActionBuildID:   buildid,
		Step:            step,
		Value:           value,
		PipelineBuildID: pipelineBuildID,
		StepOrder:       stepOrder,
	}

	return l
}
