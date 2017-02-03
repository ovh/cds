package database

import "github.com/go-gorp/gorp"

//PreInsert implement the PreInsert hook
func (p *RepositoryPoller) PreInsert(s gorp.SqlExecutor) error {
	p.ApplicationID = p.Application.ID
	p.PipelineID = p.Pipeline.ID
	return nil
}

//PreUpdate implement the PreUpdate hook
func (p *RepositoryPoller) PreUpdate(s gorp.SqlExecutor) error {
	p.ApplicationID = p.Application.ID
	p.PipelineID = p.Pipeline.ID
	return nil
}
