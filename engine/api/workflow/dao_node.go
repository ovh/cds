package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var nodeNamePattern = sdk.NamePatternRegex

func updateWorkflowTriggerSrc(db gorp.SqlExecutor, n *sdk.WorkflowNode) error {
	//Update node
	query := "UPDATE workflow_node SET workflow_trigger_src_id = $1 WHERE id = $2"
	if _, err := db.Exec(query, n.TriggerSrcID, n.ID); err != nil {
		return sdk.WrapError(err, "updateWorkflowTriggerSrc> Unable to set  workflow_trigger_src_id ON node %d", n.ID)
	}
	return nil
}

func updateWorkflowTriggerJoinSrc(db gorp.SqlExecutor, n *sdk.WorkflowNode) error {
	//Update node
	query := "UPDATE workflow_node SET workflow_trigger_join_src_id = $1 WHERE id = $2"
	if _, err := db.Exec(query, n.TriggerJoinSrcID, n.ID); err != nil {
		return sdk.WrapError(err, "updateWorkflowTriggerSrc> Unable to set  workflow_trigger_join_src_id ON node %d", n.ID)
	}
	return nil
}

func insertNode(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, n *sdk.WorkflowNode, u *sdk.User, nodes []sdk.WorkflowNode, skipDependencies bool) ([]sdk.WorkflowNode, error) {
	log.Debug("insertNode> insert or update node %s %d (%s) on %s(%#v)", n.Name, n.ID, n.Ref, n.Pipeline.Name, n.Context)

	if !nodeNamePattern.MatchString(n.Name) {
		return nodes, sdk.WrapError(sdk.ErrInvalidNodeNamePattern, "insertNode> node has a wrong name %s", n.Name)
	}

	n.WorkflowID = w.ID
	n.ID = 0

	// Set pipeline ID
	if n.PipelineID == 0 {
		n.PipelineID = n.Pipeline.ID
	}

	// Init context
	if n.Context == nil {
		n.Context = &sdk.WorkflowNodeContext{}
	}

	if n.Context.ApplicationID == 0 && n.Context.Application != nil {
		n.Context.ApplicationID = n.Context.Application.ID
	}
	if n.Context.EnvironmentID == 0 && n.Context.Environment != nil {
		n.Context.EnvironmentID = n.Context.Environment.ID
	}

	//Checks pipeline parameters
	if len(n.Context.DefaultPipelineParameters) > 0 {
		for i := range n.Context.DefaultPipelineParameters {
			var paramFound bool
			param := &n.Context.DefaultPipelineParameters[i]
			for _, pipParam := range n.Pipeline.Parameter {
				if pipParam.Name == param.Name {
					param.Type = pipParam.Type
					paramFound = true
				}
			}
			if !paramFound {
				return nil, sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("invalid parameter %s in pipeline %s", param.Name, n.Pipeline.Name))
			}
		}
	}

	//Insert new node
	dbwn := Node(*n)
	if err := db.Insert(&dbwn); err != nil {
		return nodes, sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node")
	}
	n.ID = dbwn.ID
	nodes = syncNodesArray(nodes, *n)
	if skipDependencies {
		return nodes, nil
	}

	//Insert context
	n.Context.WorkflowNodeID = n.ID
	if err := insertNodeContext(db, n.Context); err != nil {
		return nodes, sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node %d context", n.ID)
	}

	//Insert hooks
	hooksUUIDs := []string{}
	for i := range n.Hooks {
		h := &n.Hooks[i]

		if h.WorkflowHookModel.Name == sdk.RepositoryWebHookModelName && n.Context.ApplicationID == 0 {
			// Remove repository webhook
			hooksUUIDs = append(hooksUUIDs, h.UUID)
			continue
		}

		//Configure the hook
		h.Config["project"] = sdk.WorkflowNodeHookConfigValue{
			Value:        w.ProjectKey,
			Configurable: false,
		}

		h.Config["workflow"] = sdk.WorkflowNodeHookConfigValue{
			Value:        w.Name,
			Configurable: false,
		}

		if h.WorkflowHookModel.Name == sdk.RepositoryWebHookModelName || h.WorkflowHookModel.Name == sdk.GitPollerModelName {
			if n.Context.Application == nil {
				app, errA := application.LoadByID(db, store, n.Context.ApplicationID, u)
				if errA != nil {
					return nodes, sdk.WrapError(errA, "InsertOrUpdateNode> Cannot load application %d", n.Context.ApplicationID)
				}
				n.Context.Application = app
			}

			if n.Context.Application == nil || n.Context.Application.RepositoryFullname == "" || n.Context.Application.VCSServer == "" {
				return nodes, sdk.WrapError(sdk.ErrForbidden, "InsertOrUpdateNode> Cannot create a git poller or repository webhook on an application without a repository")
			}
			h.Config["vcsServer"] = sdk.WorkflowNodeHookConfigValue{
				Value:        n.Context.Application.VCSServer,
				Configurable: false,
			}
			h.Config["repoFullName"] = sdk.WorkflowNodeHookConfigValue{
				Value:        n.Context.Application.RepositoryFullname,
				Configurable: false,
			}
		}

		//Insert the hook
		if err := insertHook(db, n, h); err != nil {
			return nodes, sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node hook")
		}
	}

	// Delete hook if needed
	if len(hooksUUIDs) > 0 {
		hooks := []sdk.WorkflowNodeHook{}
		for _, h := range n.Hooks {
			found := false
			for _, uuid := range hooksUUIDs {
				if uuid == h.UUID {
					found = true
					break
				}
			}
			if !found {
				hooks = append(hooks, h)
			}
		}
		n.Hooks = hooks
	}

	//Insert triggers
	for i := range n.Triggers {
		t := &n.Triggers[i]
		var errT error
		nodes, errT = insertTrigger(db, store, w, n, t, nodes, u)
		if errT != nil {
			return nodes, sdk.WrapError(errT, "InsertOrUpdateNode> Unable to insert workflow node trigger")
		}
	}
	return nodes, nil
}

func syncNodesArray(nodes []sdk.WorkflowNode, n sdk.WorkflowNode) []sdk.WorkflowNode {
	for i := range nodes {
		if nodes[i].Name == n.Name {
			nodes[i].ID = n.ID
			break
		}
	}
	return nodes
}

type sqlContext struct {
	ID                        int64          `db:"id"`
	WorkflowNodeID            int64          `db:"workflow_node_id"`
	AppID                     sql.NullInt64  `db:"application_id"`
	EnvID                     sql.NullInt64  `db:"environment_id"`
	DefaultPayload            sql.NullString `db:"default_payload"`
	DefaultPipelineParameters sql.NullString `db:"default_pipeline_parameters"`
	Conditions                sql.NullString `db:"conditions"`
	Mutex                     sql.NullBool   `db:"mutex"`
}

// UpdateNodeContext updates the node context in database
func UpdateNodeContext(db gorp.SqlExecutor, c *sdk.WorkflowNodeContext) error {
	var sqlContext = sqlContext{}
	sqlContext.ID = c.ID
	sqlContext.WorkflowNodeID = c.WorkflowNodeID
	sqlContext.Mutex = sql.NullBool{Bool: c.Mutex, Valid: true}

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
			return sdk.WrapError(errM, "updateNodeContext> Unable to marshall workflow node context(%d) default payload", c.ID)
		}
		sqlContext.DefaultPayload = sql.NullString{String: string(b), Valid: true}
	}

	// Set PipelineParameters in context
	if c.DefaultPipelineParameters != nil {
		b, errM := json.Marshal(c.DefaultPipelineParameters)
		if errM != nil {
			return sdk.WrapError(errM, "updateNodeContext> Unable to marshall workflow node context(%d) default pipeline parameters", c.ID)
		}
		sqlContext.DefaultPipelineParameters = sql.NullString{String: string(b), Valid: true}
	}

	var errC error
	sqlContext.Conditions, errC = gorpmapping.JSONToNullString(c.Conditions)
	if errC != nil {
		return sdk.WrapError(errC, "updateNodeContext> Unable to marshall workflow node context(%d) conditions", c.ID)
	}

	if _, err := db.Update(&sqlContext); err != nil {
		return sdk.WrapError(err, "updateNodeContext> Unable to update workflow node context(%d)", c.ID)
	}
	return nil
}

func insertNodeContext(db gorp.SqlExecutor, c *sdk.WorkflowNodeContext) error {
	if err := db.QueryRow("INSERT INTO workflow_node_context (workflow_node_id) VALUES ($1) RETURNING id", c.WorkflowNodeID).Scan(&c.ID); err != nil {
		return sdk.WrapError(err, "insertNodeContext> Unable to insert workflow node context")
	}

	return UpdateNodeContext(db, c)
}

// CountPipeline Count the number of workflow that use the given pipeline
func CountPipeline(db gorp.SqlExecutor, pipelineID int64) (bool, error) {
	query := `SELECT count(1) FROM workflow_node WHERE pipeline_id= $1`
	nbWorkfow := -1
	err := db.QueryRow(query, pipelineID).Scan(&nbWorkfow)
	return nbWorkfow != 0, err
}

// loadNode loads a node in a workflow
func loadNode(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, id int64, u *sdk.User, opts LoadOptions) (*sdk.WorkflowNode, error) {
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
	triggers, errTrig := loadTriggers(db, store, w, &wn, u, opts)
	if errTrig != nil {
		return nil, sdk.WrapError(errTrig, "LoadNode> Unable to load triggers of %d", id)
	}
	wn.Triggers = triggers

	//Load context
	ctx, errCtx := LoadNodeContext(db, store, w.ProjectKey, wn.ID, u, opts)
	if errCtx != nil {
		return nil, sdk.WrapError(errCtx, "LoadNode> Unable to load context of %d", id)
	}
	wn.Context = ctx

	//Load hooks
	hooks, errHooks := loadHooks(db, &wn)
	if errHooks != nil {
		return nil, sdk.WrapError(errHooks, "LoadNode> Unable to load hooks of %d", id)
	}
	wn.Hooks = hooks

	//Load pipeline
	pip, err := pipeline.LoadPipelineByID(db, wn.PipelineID, opts.DeepPipeline)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNode> Unable to load pipeline of %d", id)
	}
	wn.Pipeline = *pip

	if wn.Name == "" {
		wn.Name = pip.Name
	}

	return &wn, nil
}

// LoadNodeContextByNodeName load the context for a given node name and user
func LoadNodeContextByNodeName(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User, workflowName, nodeName string, opts LoadOptions) (*sdk.WorkflowNodeContext, error) {
	dbnc := NodeContext{}
	query := `
		SELECT workflow_node_context.id, workflow_node_context.workflow_node_id
		FROM workflow_node_context
		JOIN workflow_node ON workflow_node.id = workflow_node_context.workflow_node_id
		JOIN workflow ON workflow.id = workflow_node.workflow_id
		JOIN project ON workflow.project_id = project.id
		WHERE workflow_node.name = $1
		AND project.id = $2
		AND workflow.name = $3
	`
	if err := db.SelectOne(&dbnc, query, nodeName, proj.ID, workflowName); err != nil {
		return nil, sdk.WrapError(err, "LoadNodeContextByNodeName> Unable to load node context %s in workflow %s in project %s", nodeName, workflowName, proj.Name)
	}
	ctx := sdk.WorkflowNodeContext(dbnc)

	if err := postLoadNodeContext(db, store, proj.Key, u, &ctx, opts); err != nil {
		return nil, sdk.WrapError(err, "LoadNodeContextByNodeName> Unable to load node context dependencies")
	}

	return &ctx, nil
}

// LoadNodeContext load the context for a given node id and user
func LoadNodeContext(db gorp.SqlExecutor, store cache.Store, projectKey string, nodeID int64, u *sdk.User, opts LoadOptions) (*sdk.WorkflowNodeContext, error) {
	dbnc := NodeContext{}
	if err := db.SelectOne(&dbnc, "select id from workflow_node_context where workflow_node_id = $1", nodeID); err != nil {
		return nil, sdk.WrapError(err, "LoadNodeContext> Unable to load node context %d", nodeID)
	}
	ctx := sdk.WorkflowNodeContext(dbnc)
	ctx.WorkflowNodeID = nodeID

	if err := postLoadNodeContext(db, store, projectKey, u, &ctx, opts); err != nil {
		return nil, sdk.WrapError(err, "LoadNodeContext> Unable to load node context dependencies")
	}

	return &ctx, nil
}

func postLoadNodeContext(db gorp.SqlExecutor, store cache.Store, projectKey string, u *sdk.User, ctx *sdk.WorkflowNodeContext, opts LoadOptions) error {
	var sqlContext = sqlContext{}
	if err := db.SelectOne(&sqlContext,
		"select application_id, environment_id, default_payload, default_pipeline_parameters, conditions, mutex from workflow_node_context where id = $1", ctx.ID); err != nil {
		return err
	}
	if sqlContext.AppID.Valid {
		ctx.ApplicationID = sqlContext.AppID.Int64
	}
	if sqlContext.EnvID.Valid {
		ctx.EnvironmentID = sqlContext.EnvID.Int64
	}
	if sqlContext.Mutex.Valid {
		ctx.Mutex = sqlContext.Mutex.Bool
	}

	//Unmarshal payload
	if err := gorpmapping.JSONNullString(sqlContext.DefaultPayload, &ctx.DefaultPayload); err != nil {
		return sdk.WrapError(err, "postLoadNodeContext> Unable to unmarshall context %d default payload", ctx.ID)
	}

	//Unmarshal pipeline parameters
	if err := gorpmapping.JSONNullString(sqlContext.DefaultPipelineParameters, &ctx.DefaultPipelineParameters); err != nil {
		return sdk.WrapError(err, "postLoadNodeContext> Unable to unmarshall context %d default pipeline parameters", ctx.ID)
	}

	//Load the application in the context
	if ctx.ApplicationID != 0 {
		app, err := application.LoadByID(db, store, ctx.ApplicationID, nil, application.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "postLoadNodeContext> Unable to load application %d", ctx.ApplicationID)
		}
		if opts.Base64Keys {
			if err := application.LoadAllBase64Keys(db, app); err != nil {
				return sdk.WrapError(err, "postLoadNodeContext> Unable to load application %d base64keys", ctx.ApplicationID)
			}
		} else {
			if err := application.LoadAllKeys(db, app); err != nil {
				return sdk.WrapError(err, "postLoadNodeContext> Unable to load application %d keys", ctx.ApplicationID)
			}
		}

		if u != nil {
			if err := application.LoadPermission(db, store, app, u); err != nil {
				return sdk.WrapError(err, "postLoadNodeContext> Unable to load application permission %d", ctx.ApplicationID)
			}
		}

		ctx.Application = app
	}

	//Load the env in the context
	if ctx.EnvironmentID != 0 {
		env, err := environment.LoadEnvironmentByID(db, ctx.EnvironmentID)
		if err != nil {
			return sdk.WrapError(err, "postLoadNodeContext> Unable to load env %d", ctx.EnvironmentID)
		}
		ctx.Environment = env

		if opts.Base64Keys {
			if errE := environment.LoadAllBase64Keys(db, env); errE != nil {
				return sdk.WrapError(errE, "postLoadNodeContext> Unable to load env %d keys", ctx.EnvironmentID)
			}
		}

		ctx.Environment.Permission = environment.Permission(env.ProjectKey, env.Name, u)
	}

	if err := gorpmapping.JSONNullString(sqlContext.Conditions, &ctx.Conditions); err != nil {
		return sdk.WrapError(err, "postLoadNodeContext> Unable to unmarshall context %d conditions", ctx.ID)
	}

	return nil
}

//deleteNode deletes nodes and all its children
func deleteNode(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, u *sdk.User) error {
	log.Debug("deleteNode> Delete node %d", node.ID)

	dbwn := Node(*node)
	if _, err := db.Delete(&dbwn); err != nil {
		return sdk.WrapError(err, "deleteNode> Unable to delete node %d", dbwn.ID)
	}

	node.ID = 0
	return nil
}
