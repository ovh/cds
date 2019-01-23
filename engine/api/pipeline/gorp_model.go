package pipeline

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// PipelineAudit is a gorp wrapper around sdk.PipelineAudit
type PipelineAudit sdk.PipelineAudit

func init() {
	gorpmapping.Register(
		gorpmapping.New(PipelineAudit{}, "pipeline_audit", true, "id"),
	)
}
