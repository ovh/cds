package sdk

import (
	"time"
)

// Log struct holds a single line of build log
type Log struct {
	ID                 int64     `json:"id" db:"id"`
	PipelineBuildJobID int64     `json:"pipeline_build_job_id" db:"pipeline_build_job_id"`
	PipelineBuildID    int64     `json:"pipeline_build_id" db:"pipeline_build_id"`
	Start              time.Time `json:"start" db:"start"`
	LastModified       time.Time `json:"last_modified" db:"last_modified"`
	Done               time.Time `json:"done" db:"done"`
	StepOrder          int       `json:"step_order" db:"step_order"`
	Value              string    `json:"value" db:"value"`
}

// NewLog returns a log struct
func NewLog(pipJobID int64, value string, pipelineBuildID int64, stepOrder int) *Log {
	l := &Log{
		PipelineBuildJobID: pipJobID,
		PipelineBuildID:    pipelineBuildID,
		Start:              time.Now(),
		StepOrder:          stepOrder,
		Value:              value,
		LastModified:       time.Now(),
	}

	return l
}
