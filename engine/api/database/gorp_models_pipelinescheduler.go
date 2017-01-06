package database

import (
	"encoding/json"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

//PostInsert is a DB Hook on PipelineScheduler to store params as JSON in DB
func (p *PipelineScheduler) PostInsert(s gorp.SqlExecutor) error {
	btes, err := json.Marshal(p.Args)
	if err != nil {
		return err
	}

	query := "update pipeline_scheduler set args = $2 where id = $1"
	if _, err := s.Exec(query, p.ID, btes); err != nil {
		return err
	}

	p.EnvironmentName, err = s.SelectStr("select name from environment where environment.id = $1", p.EnvironmentID)
	if err != nil {
		return err
	}

	return nil
}

//PostUpdate is a DB Hook on PipelineScheduler to store params as JSON in DB
func (p *PipelineScheduler) PostUpdate(s gorp.SqlExecutor) error {
	btes, err := json.Marshal(p.Args)
	if err != nil {
		return err
	}

	query := "update pipeline_scheduler set args = $2 where id = $1"
	if _, err := s.Exec(query, p.ID, btes); err != nil {
		return err
	}

	p.EnvironmentName, err = s.SelectStr("select name from environment where environment.id = $1", p.EnvironmentID)
	if err != nil {
		return err
	}

	return nil
}

//PreDelete is a DB Hook on PipelineScheduler to store params as JSON in DB
func (p *PipelineScheduler) PreDelete(s gorp.SqlExecutor) error {
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

	var err error
	p.EnvironmentName, err = s.SelectStr("select name from environment where environment.id = $1", p.EnvironmentID)
	if err != nil {
		return err
	}

	return nil
}
