package pipeline

import (
	"time"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// PipelineAudit is a gorp wrapper around sdk.PipelineAudit
type PipelineAudit sdk.PipelineAudit

type pipelineAction struct {
	ID              int64     `db:"id"`
	PipelineStageID int64     `db:"pipeline_stage_id"`
	ActionID        int64     `db:"action_id"`
	Args            string    `db:"args"`
	Enabled         bool      `db:"enabled"`
	LastModified    time.Time `db:"last_modified"`
}

func pipelineActionsToIDs(pas []pipelineAction) []int64 {
	ids := make([]int64, 0, len(pas))
	for i := range pas {
		ids[i] = pas[i].ID
	}
	return ids
}

func pipelineActionsToActionIDs(pas []pipelineAction) []int64 {
	ids := make([]int64, 0, len(pas))
	for i := range pas {
		ids[i] = pas[i].ActionID
	}
	return ids
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(PipelineAudit{}, "pipeline_audit", true, "id"),
		gorpmapping.New(pipelineAction{}, "pipeline_action", true, "id"),
	)
}
