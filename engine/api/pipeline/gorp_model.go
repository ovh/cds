package pipeline

import (
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// PipelineAudit is a gorp wrapper around sdk.PipelineAudit
type PipelineAudit sdk.PipelineAudit

// Pipeline is a gorp wrapper around sdk.Pipeline
type Pipeline sdk.Pipeline

type pipelineAction struct {
	ID              int64     `db:"id"`
	PipelineStageID int64     `db:"pipeline_stage_id"`
	ActionID        int64     `db:"action_id"`
	Args            *string   `db:"args"`
	Enabled         bool      `db:"enabled"`
	LastModified    time.Time `db:"last_modified"`
}

func pipelineActionsToIDs(pas []pipelineAction) []int64 {
	ids := make([]int64, 0, len(pas))
	for i := range pas {
		ids = append(ids, pas[i].ID)
	}
	return ids
}

func pipelineActionsToActionIDs(pas []pipelineAction) []int64 {
	ids := make([]int64, 0, len(pas))
	for i := range pas {
		ids = append(ids, pas[i].ActionID)
	}
	return ids
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(Pipeline{}, "pipeline", true, "id"),
		gorpmapping.New(PipelineAudit{}, "pipeline_audit", true, "id"),
		gorpmapping.New(pipelineAction{}, "pipeline_action", true, "id"),
	)
}

func (pip *Pipeline) PostGet(db gorp.SqlExecutor) error {
	projectKey, err := db.SelectStr("SELECT project.projectkey FROM project WHERE id = $1", pip.ProjectID)
	if err != nil {
		return sdk.WrapError(err, "cannot fetch project key for project id %d", pip.ProjectID)
	}
	pip.ProjectKey = projectKey

	return nil
}
