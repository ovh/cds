package scheduler

import "github.com/ovh/cds/sdk"

//New instanciates a new scheduler for an application/pipeline/env
func New(app *sdk.Application, pip *sdk.Pipeline, env *sdk.Environment, params []sdk.Parameter) {

}

type pipelineSchedulerCron struct {
}

func (s *pipelineSchedulerCron) Run() {
}
