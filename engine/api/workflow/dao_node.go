package workflow

import (
	"database/sql"

	"github.com/ovh/cds/sdk"
)

var nodeNamePattern = sdk.NamePatternRegex

type sqlContext struct {
	ID                        int64          `db:"id"`
	WorkflowNodeID            int64          `db:"workflow_node_id"`
	AppID                     sql.NullInt64  `db:"application_id"`
	EnvID                     sql.NullInt64  `db:"environment_id"`
	ProjectIntegrationID      sql.NullInt64  `db:"project_integration_id"`
	DefaultPayload            sql.NullString `db:"default_payload"`
	DefaultPipelineParameters sql.NullString `db:"default_pipeline_parameters"`
	Conditions                sql.NullString `db:"conditions"`
	Mutex                     sql.NullBool   `db:"mutex"`
}
