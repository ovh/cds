package sdk

//PipelineScheduler is a cron scheduler
type PipelineScheduler struct {
	ID            int64       `json:"id" db:"id"`
	ApplicationID int64       `json:"-" db:"application_id"`
	PipelineID    int64       `json:"-" db:"pipeline_id"`
	EnvironmentID int64       `json:"-" db:"environment_id"`
	Args          []Parameter `json:"args,omitempty" db:"-"`
	Crontab       string      `json:"crontab,omitempty" db:"crontab"`
}
