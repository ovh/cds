package poller

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
)

// RepositoryPoller is a gorp wrapper around sdk.RepositoryPoller
type RepositoryPoller sdk.RepositoryPoller

// RepositoryPollerExecution is a gorp wrapper around sdk.RepositoryPollerExecution
type RepositoryPollerExecution sdk.RepositoryPollerExecution

// PreInsert implement the PreInsert hook
func (p *RepositoryPoller) PreInsert(s gorp.SqlExecutor) error {
	if p.ApplicationID == 0 {
		p.ApplicationID = p.Application.ID
	}
	if p.PipelineID == 0 {
		p.PipelineID = p.Pipeline.ID
	}
	return nil
}

// PreUpdate implement the PreUpdate hook
func (p *RepositoryPoller) PreUpdate(s gorp.SqlExecutor) error {
	p.ApplicationID = p.Application.ID
	p.PipelineID = p.Pipeline.ID
	return nil
}

// PreDelete is a DB Hook
func (p *RepositoryPoller) PreDelete(s gorp.SqlExecutor) error {
	if _, err := s.Exec("delete from poller_execution where application_id = $1 and pipeline_id = $2", p.ApplicationID, p.PipelineID); err != nil {
		return err
	}
	return nil
}

// PostGet is a DB Hook
func (p *RepositoryPoller) PostGet(db gorp.SqlExecutor) error {
	app, err := application.LoadByID(db, nil, p.ApplicationID, nil, application.LoadOptions.Default)
	if err != nil {
		return sdk.WrapError(err, "PostGet> error loading application %d", p.ApplicationID)
	}
	pip, err := pipeline.LoadPipelineByID(db, p.PipelineID, true)
	if err != nil {
		return sdk.WrapError(err, "PostGet> error loading pipeline %d", p.PipelineID)
	}
	p.Application = *app
	p.Pipeline = *pip

	next, errN := LoadNextExecution(db, app.ID, pip.ID)
	if errN != nil {
		return sdk.WrapError(errN, "PostGet> Cannot load nextexecution")
	}
	if next != nil {
		p.NextExecution = next
	}
	return nil
}

//PostInsert is a DB Hook
func (p *RepositoryPollerExecution) PostInsert(s gorp.SqlExecutor) error {
	eBtes, err := json.Marshal(p.PushEvents)
	if err != nil {
		return err
	}

	pBtes, err := json.Marshal(p.PipelineBuildVersions)
	if err != nil {
		return err
	}

	query := "update poller_execution set push_events = $2, pipeline_build_versions = $3 where id = $1"
	if _, err := s.Exec(query, p.ID, eBtes, pBtes); err != nil {
		return err
	}
	return nil
}

//PostUpdate is a DB Hook
func (p *RepositoryPollerExecution) PostUpdate(s gorp.SqlExecutor) error {
	return p.PostInsert(s)
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(RepositoryPoller{}, "poller", false, "application_id", "pipeline_id"),
		gorpmapping.New(RepositoryPollerExecution{}, "poller_execution", true, "id"),
	)
}
