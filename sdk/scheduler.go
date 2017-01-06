package sdk

import (
	"time"
)

//PipelineScheduler is a cron scheduler
type PipelineScheduler struct {
	ID              int64                       `json:"id" db:"id"`
	ApplicationID   int64                       `json:"-" db:"application_id"`
	PipelineID      int64                       `json:"-" db:"pipeline_id"`
	EnvironmentID   int64                       `json:"-" db:"environment_id"`
	EnvironmentName string                      `json:"environment_name" db:"-"`
	Args            []Parameter                 `json:"args,omitempty" db:"-"`
	Crontab         string                      `json:"crontab,omitempty" db:"crontab"`
	Disabled        bool                        `json:"disable" db:"disable"`
	LastExecution   *PipelineSchedulerExecution `json:"last_execution" db:"-"`
	NextExecution   *PipelineSchedulerExecution `json:"next_execution" db:"-"`
}

//PipelineSchedulerExecution is a cron scheduler execution
type PipelineSchedulerExecution struct {
	ID                   int64      `json:"id" db:"id"`
	PipelineSchedulerID  int64      `json:"-" db:"pipeline_scheduler_id"`
	ExecutionPlannedDate time.Time  `json:"execution_planned_date,omitempty" db:"execution_planned_date"`
	ExecutionDate        *time.Time `json:"execution_date" db:"execution_date"`
	Executed             bool       `json:"executed" db:"executed"`
	PipelineBuildVersion int64      `json:"pipeline_build_version" db:"pipeline_build_version"`
}
