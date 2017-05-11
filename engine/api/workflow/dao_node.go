package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertOrUpdateNode insert or update a node for the workflow
func insertOrUpdateNode(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.WorkflowNode, u *sdk.User, skipDependencies bool) error {
	defer func() {
		log.Debug("insertOrUpdateNode> insert or update node %d (%s) on %s", n.ID, n.Ref, n.Pipeline.Name)
	}()
	n.WorkflowID = w.ID

	if n.PipelineID == 0 {
		n.PipelineID = n.Pipeline.ID
	}

	if n.Context == nil {
		n.Context = &sdk.WorkflowNodeContext{}
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
		log.Debug("InsertOrUpdateNode> Updating root node %d", oldNode.ID)
		dbwn := Node(*n)
		if _, err := db.Update(&dbwn); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to update workflow root node")
		}
		if err := DeleteNodeDependencies(db, n); err != nil {
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
	if err := db.QueryRow("INSERT INTO workflow_node_context (workflow_node_id) VALUES ($1) RETURNING id", n.Context.WorkflowNodeID).Scan(&n.Context.ID); err != nil {
		return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node context")
	}

	// Set ApplicationID in context
	if n.Context.ApplicationID != 0 {
		if _, err := db.Exec("UPDATE workflow_node_context SET application_id=$1 where id=$2", n.Context.ApplicationID, n.Context.ID); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node context(%d) for application %d", n.Context.ID, n.Context.ApplicationID)
		}
	}

	// Set EnvironmentID in context
	if n.Context.EnvironmentID != 0 {
		if _, err := db.Exec("UPDATE workflow_node_context SET environment_id=$1 where id=$2", n.Context.EnvironmentID, n.Context.ID); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node context(%d) for env %d", n.Context.ID, n.Context.EnvironmentID)
		}
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

	//Load triggers
	triggers, err := loadTriggers(db, w, &wn, u)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNode> Unable to load triggers of %d", id)
	}
	wn.Triggers = triggers

	//TODO: Check user permission

	//Load context
	dbnc := NodeContext{}
	if err := db.SelectOne(&dbnc, "select id from workflow_node_context where workflow_node_id = $1", wn.ID); err != nil {
		return nil, sdk.WrapError(err, "LoadNode> Unable to load node context %d", wn.ID)
	}
	ctx := sdk.WorkflowNodeContext(dbnc)
	ctx.WorkflowNodeID = wn.ID

	appID, err := db.SelectNullInt("select application_id from workflow_node_context where id = $1", ctx.ID)
	if err != nil {
		return nil, err
	}
	if appID.Valid {
		ctx.ApplicationID = appID.Int64
	}

	envID, err := db.SelectNullInt("select environment_id from workflow_node_context where id = $1", ctx.ID)
	if err != nil {
		return nil, err
	}
	if envID.Valid {
		ctx.EnvironmentID = envID.Int64
	}

	//Load the application in the context
	if ctx.ApplicationID != 0 {
		app, err := application.LoadByID(db, ctx.ApplicationID, u)
		if err != nil {
			return nil, sdk.WrapError(err, "LoadNode> Unable to load application %d", ctx.ApplicationID)
		}
		ctx.Application = app
	}

	//Load the env in the context
	if ctx.EnvironmentID != 0 {
		env, err := environment.LoadEnvironmentByID(db, ctx.EnvironmentID)
		if err != nil {
			return nil, sdk.WrapError(err, "LoadNode> Unable to load env %d", ctx.EnvironmentID)
		}
		ctx.Environment = env
	}

	wn.Context = &ctx

	//Load pipeline
	pip, err := pipeline.LoadPipelineByID(db, wn.PipelineID, false)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNode> Unable to load pipeline of %d", id)
	}
	wn.Pipeline = *pip

	return &wn, nil
}

//deleteNode deletes nodes and all its children
func deleteNode(db gorp.SqlExecutor, node *sdk.WorkflowNode) error {
	if err := DeleteNodeDependencies(db, node); err != nil {
		return sdk.WrapError(err, "DeleteNode> Unable to delete node dependencies %d", node.ID)
	}

	dbwn := Node(*node)
	if _, err := db.Delete(&dbwn); err != nil {
		return sdk.WrapError(err, "DeleteNode> Unable to delete node %d", dbwn.ID)
	}

	return nil
}

//DeleteNodeDependencies delete triggers, hooks, context for the node
func DeleteNodeDependencies(db gorp.SqlExecutor, node *sdk.WorkflowNode) error {
	for _, t := range node.Triggers {
		if err := deleteTrigger(db, &t); err != nil {
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
	}

	return nil
}
