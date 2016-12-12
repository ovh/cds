package database

import (
	"github.com/go-gorp/gorp"
)

//PostInsert is a DB Hook on PipelineScheduler to store params as JSON in DB
func (p *PipelineScheduler) PostInsert(s gorp.SqlExecutor) error {
	return nil
}

//PostUpdate is a DB Hook on PipelineScheduler to store params as JSON in DB
func (p *PipelineScheduler) PostUpdate(s gorp.SqlExecutor) error {
	return nil
}

//PreDelete is a DB Hook on PipelineScheduler to store params as JSON in DB
func (p *PipelineScheduler) PreDelete(s gorp.SqlExecutor) error {
	return nil
}

//PostSelect is a DB Hook to get all data from DB
func (p *PipelineScheduler) PostSelect(s gorp.SqlExecutor) error {
	return nil
}
