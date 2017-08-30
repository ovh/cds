package scheduler

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

//PipelineScheduler is a gorp wrapper around sdk.PipelineScheduler
type PipelineScheduler sdk.PipelineScheduler

//PipelineSchedulerExecution is a gorp wrapper around sdk.PipelineSchedulerExecution
type PipelineSchedulerExecution sdk.PipelineSchedulerExecution

//PostInsert is a DB Hook on PipelineScheduler to store params as JSON in DB
func (p *PipelineScheduler) PostInsert(s gorp.SqlExecutor) error {
	btes, errj := json.Marshal(p.Args)
	if errj != nil {
		return errj
	}

	query := "update pipeline_scheduler set args = $2 where id = $1"
	if _, err := s.Exec(query, p.ID, btes); err != nil {
		return err
	}

	var errs error
	p.EnvironmentName, errs = s.SelectStr("select name from environment where environment.id = $1", p.EnvironmentID)
	if errs != nil {
		return errs
	}

	return nil
}

//PostUpdate is a DB Hook on PipelineScheduler to store params as JSON in DB
func (p *PipelineScheduler) PostUpdate(s gorp.SqlExecutor) error {
	btes, errj := json.Marshal(p.Args)
	if errj != nil {
		return errj
	}

	query := "update pipeline_scheduler set args = $2 where id = $1"
	if _, err := s.Exec(query, p.ID, btes); err != nil {
		return err
	}

	var errs error
	p.EnvironmentName, errs = s.SelectStr("select name from environment where environment.id = $1", p.EnvironmentID)
	if errs != nil {
		return errs
	}

	return nil
}

//PreDelete is a DB Hook on PipelineScheduler to store params as JSON in DB
func (p *PipelineScheduler) PreDelete(s gorp.SqlExecutor) error {
	if _, err := s.Exec("delete from pipeline_scheduler_execution where pipeline_scheduler_id = $1", p.ID); err != nil {
		return err
	}
	return nil
}

//PostGet is a DB Hook to get all data from DB
func (p *PipelineScheduler) PostGet(s gorp.SqlExecutor) error {
	//Load created_by
	p.Args = []sdk.Parameter{}
	str, errSelect := s.SelectNullStr("select args from pipeline_scheduler where id = $1", &p.ID)
	if errSelect != nil {
		return errSelect
	}
	if !str.Valid || str.String == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(str.String), &p.Args); err != nil {
		return err
	}

	var errs error
	p.EnvironmentName, errs = s.SelectStr("select name from environment where environment.id = $1", p.EnvironmentID)
	if errs != nil {
		return errs
	}

	return nil
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(PipelineScheduler{}, "pipeline_scheduler", true, "id"),
		gorpmapping.New(PipelineSchedulerExecution{}, "pipeline_scheduler_execution", true, "id"),
	)
}
