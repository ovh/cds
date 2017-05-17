package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertOrUpdateNode insert or update a node for the workflow
func insertOrUpdateNode(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.WorkflowNode, u *sdk.User, skipDependencies bool) error {
	log.Debug("insertOrUpdateNode> insert or update node %d (%s) on %s(%#v)", n.ID, n.Ref, n.Pipeline.Name, n.Context)

	n.WorkflowID = w.ID

	if n.PipelineID == 0 {
		n.PipelineID = n.Pipeline.ID
	}

	if n.Context == nil {
		n.Context = &sdk.WorkflowNodeContext{}
	}

	//Loading the pipeline to check pipeline parameters
	pip, errPip := pipeline.LoadPipelineByID(db, n.PipelineID, true)
	if errPip != nil {
		return sdk.WrapError(errPip, "InsertOrUpdateNode> Unable to load workflow node pipeline %d", n.PipelineID)
	}

	if len(pip.Parameter) != len(n.Context.DefaultPipelineParameters) {
		return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Pipeline parameters don't match"))
	}

	//Checking parameters
	for _, p := range pip.Parameter {
		if sdk.ParameterFind(n.Context.DefaultPipelineParameters, p.Name) == nil {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Pipeline parameter %s doens't match", p.Name))
		}
	}

	if n.Context.ApplicationID == 0 && n.Context.Application != nil {
		n.Context.ApplicationID = n.Context.Application.ID
	}
	if n.Context.EnvironmentID == 0 && n.Context.Environment != nil {
		n.Context.EnvironmentID = n.Context.Environment.ID
	}

	var oldNode *sdk.WorkflowNode

	//If the node got an ID; check it in database
	if n.ID != 0 {
		var err error
		oldNode, err = loadNode(db, w, n.ID, u)
		if err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to load workflow node")
		}
		n.Context.WorkflowNodeID = oldNode.ID
	}

	if oldNode != nil {
		//Update the node
		dbwn := Node(*n)
		if _, err := db.Update(&dbwn); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to update workflow root node")
		}
		if err := DeleteNodeDependencies(db, w, n, u); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to delete workflow root node dependencies")
		}
	} else {
		//Insert new node
		dbwn := Node(*n)
		if err := db.Insert(&dbwn); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node")
		}
		n.ID = dbwn.ID
	}

	if skipDependencies {
		return nil
	}

	//Insert context
	n.Context.WorkflowNodeID = n.ID
	if err := insertNodeContext(db, n.Context); err != nil {
		return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node %d context", n.ID)
	}

	//Insert hooks
	for i := range n.Hooks {
		h := &n.Hooks[i]
		if err := insertOrUpdateHook(db, n, h); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node trigger")
		}
	}

	//Insert triggers
	for i := range n.Triggers {
		t := &n.Triggers[i]
		if err := insertOrUpdateTrigger(db, w, n, t, u); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node trigger")
		}
	}

	return nil
}

type sqlContext struct {
	ID                        int64          `db:"id"`
	WorkflowNodeID            int64          `db:"workflow_node_id"`
	AppID                     sql.NullInt64  `db:"application_id"`
	EnvID                     sql.NullInt64  `db:"environment_id"`
	DefaultPayload            sql.NullString `db:"default_payload"`
	DefaultPipelineParameters sql.NullString `db:"default_pipeline_parameters"`
}

func insertNodeContext(db gorp.SqlExecutor, c *sdk.WorkflowNodeContext) error {
	if err := db.QueryRow("INSERT INTO workflow_node_context (workflow_node_id) VALUES ($1) RETURNING id", c.WorkflowNodeID).Scan(&c.ID); err != nil {
		return sdk.WrapError(err, "insertNodeContext> Unable to insert workflow node context")
	}

	var sqlContext = sqlContext{}
	sqlContext.ID = c.ID
	sqlContext.WorkflowNodeID = c.WorkflowNodeID

	// Set ApplicationID in context
	if c.ApplicationID != 0 {
		sqlContext.AppID = sql.NullInt64{Int64: c.ApplicationID, Valid: true}
	}

	// Set EnvironmentID in context
	if c.EnvironmentID != 0 {
		sqlContext.EnvID = sql.NullInt64{Int64: c.EnvironmentID, Valid: true}
	}

	// Set DefaultPayload in context
	if c.DefaultPayload != nil {
		b, errM := json.Marshal(c.DefaultPayload)
		if errM != nil {
			return sdk.WrapError(errM, "InsertOrUpdateNode> Unable to marshall workflow node context(%d) default payload", c.ID)
		}
		sqlContext.DefaultPayload = sql.NullString{String: string(b), Valid: true}
	}

	// Set PipelineParameters in context
	if c.DefaultPayload != nil {
		b, errM := json.Marshal(c.DefaultPipelineParameters)
		if errM != nil {
			return sdk.WrapError(errM, "InsertOrUpdateNode> Unable to marshall workflow node context(%d) default pipeline parameters", c.ID)
		}
		sqlContext.DefaultPipelineParameters = sql.NullString{String: string(b), Valid: true}
	}

	if _, err := db.Update(&sqlContext); err != nil {
		return sdk.WrapError(err, "InsertOrUpdateNode> Unable to update workflow node context(%d)", c.ID)
	}

	return nil
}

// loadNode loads a node in a workflow
func loadNode(db gorp.SqlExecutor, w *sdk.Workflow, id int64, u *sdk.User) (*sdk.WorkflowNode, error) {
	dbwn := Node{}
	if err := db.SelectOne(&dbwn, "select * from workflow_node where workflow_id = $1 and id = $2", w.ID, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNodeNotFound
		}
		return nil, err
	}

	wn := sdk.WorkflowNode(dbwn)
	wn.WorkflowID = w.ID
	wn.Ref = fmt.Sprintf("%d", dbwn.ID)

	//Load triggers
	triggers, errTrig := loadTriggers(db, w, &wn, u)
	if errTrig != nil {
		return nil, sdk.WrapError(errTrig, "LoadNode> Unable to load triggers of %d", id)
	}
	wn.Triggers = triggers

	//TODO: Check user permission

	//Load context
	ctx, errCtx := loadNodeContext(db, &wn, u)
	if errCtx != nil {
		return nil, sdk.WrapError(errCtx, "LoadNode> Unable to load context of %d", id)
	}
	wn.Context = ctx

	//Load pipeline
	pip, err := pipeline.LoadPipelineByID(db, wn.PipelineID, false)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNode> Unable to load pipeline of %d", id)
	}
	wn.Pipeline = *pip

	return &wn, nil
}

func loadNodeContext(db gorp.SqlExecutor, wn *sdk.WorkflowNode, u *sdk.User) (*sdk.WorkflowNodeContext, error) {
	dbnc := NodeContext{}
	if err := db.SelectOne(&dbnc, "select id from workflow_node_context where workflow_node_id = $1", wn.ID); err != nil {
		return nil, sdk.WrapError(err, "loadNodeContext> Unable to load node context %d", wn.ID)
	}
	ctx := sdk.WorkflowNodeContext(dbnc)
	ctx.WorkflowNodeID = wn.ID

	var sqlContext = sqlContext{}
	if err := db.SelectOne(&sqlContext,
		"select application_id, environment_id, default_payload, default_pipeline_parameters from workflow_node_context where id = $1", ctx.ID); err != nil {
		return nil, err
	}
	if sqlContext.AppID.Valid {
		ctx.ApplicationID = sqlContext.AppID.Int64
	}
	if sqlContext.EnvID.Valid {
		ctx.EnvironmentID = sqlContext.EnvID.Int64
	}

	//Unmarshal payload
	if sqlContext.DefaultPayload.Valid {
		if err := json.Unmarshal([]byte(sqlContext.DefaultPayload.String), &ctx.DefaultPayload); err != nil {
			return nil, sdk.WrapError(err, "loadNodeContext> Unable to unmarshall context %d default payload %d", ctx.ID)
		}
	}

	//Unmarshal pipeline parameters
	if sqlContext.DefaultPipelineParameters.Valid {
		if err := json.Unmarshal([]byte(sqlContext.DefaultPipelineParameters.String), &ctx.DefaultPipelineParameters); err != nil {
			return nil, sdk.WrapError(err, "loadNodeContext> Unable to unmarshall context %d default pipeline parameters %d", ctx.ID)
		}
	}

	//Load the application in the context
	if ctx.ApplicationID != 0 {
		app, err := application.LoadByID(db, ctx.ApplicationID, u)
		if err != nil {
			return nil, sdk.WrapError(err, "loadNodeContext> Unable to load application %d", ctx.ApplicationID)
		}
		ctx.Application = app
	}

	//Load the env in the context
	if ctx.EnvironmentID != 0 {
		env, err := environment.LoadEnvironmentByID(db, ctx.EnvironmentID)
		if err != nil {
			return nil, sdk.WrapError(err, "loadNodeContext> Unable to load env %d", ctx.EnvironmentID)
		}
		ctx.Environment = env
	}
	return &ctx, nil
}

//deleteNode deletes nodes and all its children
func deleteNode(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, u *sdk.User) error {
	log.Debug("deleteNode> Delete node %d", node.ID)
	if err := DeleteNodeDependencies(db, w, node, u); err != nil {
		return sdk.WrapError(err, "DeleteNode> Unable to delete node dependencies %d", node.ID)
	}

	dbwn := Node(*node)
	if _, err := db.Delete(&dbwn); err != nil {
		return sdk.WrapError(err, "DeleteNode> Unable to delete node %d", dbwn.ID)
	}

	node.ID = 0
	return nil
}

//DeleteNodeDependencies delete triggers, hooks, context for the node
func DeleteNodeDependencies(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, u *sdk.User) error {
	log.Debug("DeleteNodeDependencies> Delete node %d", node.ID)
	for i := range node.Triggers {
		t := &node.Triggers[i]
		if err := deleteTrigger(db, w, t, u); err != nil {
			return sdk.WrapError(err, "DeleteNodeDependencies> Unable to delete trigger %d", t.ID)
		}
	}

	for _, h := range node.Hooks {
		if err := deleteHook(db, &h); err != nil {
			return sdk.WrapError(err, "DeleteNodeDependencies> Unable to delete hook %d", h.ID)
		}
	}

	if node.Context != nil {
		dbnc := NodeContext(*node.Context)
		if _, err := db.Delete(&dbnc); err != nil {
			return sdk.WrapError(err, "DeleteNodeDependencies> Unable to delete context %d", dbnc.ID)
		}
		node.Context.ID = 0
	}

	query := "select workflow_node_join_id from workflow_node_join_source where workflow_node_id = $1"
	joinIDs := []int64{}
	if _, err := db.Select(&joinIDs, query, node.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "DeleteNodeDependencies> Unable to load joins for node %d", node.ID)
	}

	for _, jid := range joinIDs {
		j, err := loadJoin(db, w, jid, u)
		if err != nil {
			return sdk.WrapError(err, "DeleteNodeDependencies> Unable to load join %d for node %d", jid, node.ID)
		}
		if err := deleteJoin(db, w, j, u); err != nil {
			return sdk.WrapError(err, "DeleteNodeDependencies> Unable to delete join %d for node %d", jid, node.ID)
		}
	}

	return nil
}
