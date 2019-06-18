package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type sqlNodeContextData struct {
	ID                        int64          `db:"id"`
	NodeID                    int64          `db:"node_id"`
	PipelineID                sql.NullInt64  `db:"pipeline_id"`
	ApplicationID             sql.NullInt64  `db:"application_id"`
	EnvironmentID             sql.NullInt64  `db:"environment_id"`
	ProjectIntegrationID      sql.NullInt64  `db:"project_integration_id"`
	DefaultPayload            sql.NullString `db:"default_payload"`
	DefaultPipelineParameters sql.NullString `db:"default_pipeline_parameters"`
	Conditions                sql.NullString `db:"conditions"`
	Mutex                     bool           `db:"mutex"`
}

func insertNodeContextData(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node) error {
	if n.Context == nil {
		n.Context = &sdk.NodeContext{}
	}
	n.Context.ID = 0

	tempContext := sqlNodeContextData{}
	tempContext.NodeID = n.ID
	if n.Context.PipelineID != 0 {
		tempContext.PipelineID = sql.NullInt64{Valid: true, Int64: n.Context.PipelineID}
	}
	if n.Context.ApplicationID != 0 {
		tempContext.ApplicationID = sql.NullInt64{Valid: true, Int64: n.Context.ApplicationID}
	}
	if n.Context.EnvironmentID != 0 {
		tempContext.EnvironmentID = sql.NullInt64{Valid: true, Int64: n.Context.EnvironmentID}
	}
	if n.Context.ProjectIntegrationID != 0 {
		tempContext.ProjectIntegrationID = sql.NullInt64{Valid: true, Int64: n.Context.ProjectIntegrationID}
	}

	var errDP error
	tempContext.DefaultPayload, errDP = gorpmapping.JSONToNullString(n.Context.DefaultPayload)
	if errDP != nil {
		return sdk.WrapError(errDP, "insertNodeContextData> Cannot stringify default payload")
	}

	for _, cond := range n.Context.Conditions.PlainConditions {
		if _, ok := sdk.WorkflowConditionsOperators[cond.Operator]; !ok {
			return sdk.ErrWorkflowConditionBadOperator
		}
	}

	var errC error
	tempContext.Conditions, errC = gorpmapping.JSONToNullString(n.Context.Conditions)
	if errC != nil {
		return sdk.WrapError(errC, "insertNodeContextData> Cannot stringify default pipeline parameters")
	}

	tempContext.Mutex = n.Context.Mutex

	if n.Context.PipelineID != 0 {
		//Checks pipeline parameters
		if len(n.Context.DefaultPipelineParameters) > 0 {
			defaultPipParams := make([]sdk.Parameter, 0, len(n.Context.DefaultPipelineParameters))
			for i := range n.Context.DefaultPipelineParameters {
				var paramFound bool
				param := &n.Context.DefaultPipelineParameters[i]
				for _, pipParam := range w.Pipelines[n.Context.PipelineID].Parameter {
					if pipParam.Name == param.Name {
						param.Type = pipParam.Type
						paramFound = true
						break
					}
				}
				if paramFound {
					defaultPipParams = append(defaultPipParams, *param)
				}
			}
			n.Context.DefaultPipelineParameters = defaultPipParams

		}
	}
	var errDPP error
	tempContext.DefaultPipelineParameters, errDPP = gorpmapping.JSONToNullString(n.Context.DefaultPipelineParameters)
	if errDPP != nil {
		return sdk.WrapError(errDPP, "insertNodeContextData> Cannot stringify default pipeline parameters")
	}

	//Insert new node
	dbContext := dbNodeContextData(tempContext)
	if err := db.Insert(&dbContext); err != nil {
		return sdk.WrapError(err, "insertNodeContextData> Unable to insert workflow node context")
	}
	n.Context.ID = dbContext.ID
	n.Context.NodeID = n.ID
	return nil
}
