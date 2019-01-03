package pipeline

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// Log is a gorp wrapper around sdk.Log
type Log sdk.Log

// PipelineAudit is a gorp wrapper around sdk.PipelineAudit
type PipelineAudit sdk.PipelineAudit

func init() {
	gorpmapping.Register(
		gorpmapping.New(Log{}, "pipeline_build_log", true, "id"),
		gorpmapping.New(PipelineAudit{}, "pipeline_audit", true, "id"),
	)
}
