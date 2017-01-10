package scheduler

import (
	"github.com/gorhill/cronexpr"
	"github.com/ovh/cds/sdk"
)

//New instanciates a new pipeline scheduler
func New(app *sdk.Application, pip *sdk.Pipeline, env *sdk.Environment, cron string, args ...sdk.Parameter) (*sdk.PipelineScheduler, error) {
	_, err := cronexpr.Parse(cron)
	if err != nil {
		return nil, err
	}

	if env == nil {
		env = &sdk.DefaultEnv
	}

	return &sdk.PipelineScheduler{
		ApplicationID: app.ID,
		PipelineID:    pip.ID,
		EnvironmentID: env.ID,
		Crontab:       cron,
		Disabled:      false,
		Args:          args,
	}, nil
}
