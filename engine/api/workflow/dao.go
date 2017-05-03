package workflow

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"fmt"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadAll loads all workflows for a project. All users in a project can list all workflows in a project
func LoadAll(db gorp.SqlExecutor, projectKey string) ([]sdk.Workflow, error) {
	res := []sdk.Workflow{}
	dbRes := []Workflow{}

	query := `
		select workflow.* 
		from workflow
		join project on project.id = workflow.project_id
		where project.projectkey = $1
		order by workflow.name asc`

	if _, err := db.Select(&dbRes, query, projectKey); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "LoadAll> Unable to load workflows project %s", projectKey)
	}

	for _, w := range dbRes {
		w.ProjectKey = projectKey
		res = append(res, sdk.Workflow(w))
	}

	return res, nil
}

func loadWorkflowRoot(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	var err error
	w.Root, err = LoadNode(db, w, w.RootID, u)
	if err != nil {
		if err == sdk.ErrWorkflowNodeNotFound {
			log.Debug("Load> Unable to load root %d for workflow %d", w.RootID, w.ID)
			return nil
		}
		return sdk.WrapError(err, "Load> Unable to load workflow root %d", w.RootID)
	}
	return nil
}

// Load loads a workflow for a given user (ie. checking permissions)
func Load(db gorp.SqlExecutor, projectKey, name string, u *sdk.User) (*sdk.Workflow, error) {
	query := `
		select workflow.* 
		from workflow
		join project on project.id = workflow.project_id
		where project.projectkey = $1
		and workflow.name = $2`
	res, err := load(db, u, query, projectKey, name)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow %s in project %s", name, projectKey)
	}
	return res, nil
}

// LoadByID loads a workflow for a given user (ie. checking permissions)
func LoadByID(db gorp.SqlExecutor, id int64, u *sdk.User) (*sdk.Workflow, error) {
	query := `
		select * 
		from workflow
		where id = $1`
	res, err := load(db, u, query, id)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow %d", id)
	}
	return res, nil
}

func load(db gorp.SqlExecutor, u *sdk.User, query string, args ...interface{}) (*sdk.Workflow, error) {
	t0 := time.Now()
	dbRes := Workflow{}
	if err := db.SelectOne(&dbRes, query, args...); err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow")
	}

	res := sdk.Workflow(dbRes)
	res.ProjectKey, _ = db.SelectStr("select projectkey from project where id = $1", res.ProjectID)
	if err := loadWorkflowRoot(db, &res, u); err != nil {
		return nil, err
	}

	delta := time.Since(t0).Seconds()
	log.Debug("Load> Load workflow (%s/%s)%d took %.3f seconds", res.ProjectKey, res.Name, res.ID, delta)
	return &res, nil

}

// Insert inserts a new workflow
func Insert(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	w.LastModified = time.Now()
	if err := db.QueryRow("INSERT INTO workflow (name, description, project_id) VALUES ($1, $2, $3) RETURNING id", w.Name, w.Description, w.ProjectID).Scan(&w.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow %s/%s", w.ProjectKey, w.Name)
	}

	if err := InsertOrUpdateNode(db, w, w.Root, u); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow root node")
	}

	if _, err := db.Exec("UPDATE workflow SET root_node_id = $2 WHERE id = $1", w.ID, w.Root.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow (%#v, %d)", w.Root, w.ID)
	}
	return nil
}

// Update updates a workflow
func Update(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	w.LastModified = time.Now()
	dbw := Workflow(*w)
	if _, err := db.Update(&dbw); err != nil {
		return sdk.WrapError(err, "Update> Unable to update workflow")
	}
	if w.Root != nil {
		return InsertOrUpdateNode(db, w, w.Root, u)
	}
	return nil
}

// Delete workflow
func Delete(db gorp.SqlExecutor, w *sdk.Workflow) error {
	//Detach root from workflow
	if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
		return sdk.WrapError(err, "Delete> Unable to detache workflow root")
	}

	//Delete root
	if err := DeleteNode(db, w.Root); err != nil {
		return sdk.WrapError(err, "Delete> Unable to delete workflow root")
	}

	//Delete workflow
	dbw := Workflow(*w)
	if _, err := db.Delete(&dbw); err != nil {
		return sdk.WrapError(err, "Delete> Unable to delete workflow")
	}

	return nil
}

// UpdateLastModified updates the workflow
func UpdateLastModified(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	t := time.Now()

	if u != nil {
		cache.SetWithTTL(cache.Key("lastModified", "workflow", fmt.Sprintf("%d", w.ID)), sdk.LastModification{
			Name:         w.Name,
			Username:     u.Username,
			LastModified: t.Unix(),
		}, 0)
	}

	if _, err := db.Exec("update workflow set last_modified = $2 where id = $1", w.ID, t); err != nil {
		return sdk.WrapError(err, "UpdateLastModified> Unable to update workflow")
	}
	return nil
}

// InsertOrUpdateNode insert or update a node for the workflow
func InsertOrUpdateNode(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.WorkflowNode, u *sdk.User) error {
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
		oldNode, err = LoadNode(db, w, n.ID, u)
		if err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to load workflow node")
		}
		n.Context.WorkflowNodeID = oldNode.ID
	}

	//Delete old node
	var isRoot bool
	if oldNode != nil {
		if w.Root.ID != n.ID {
			if err := DeleteNode(db, oldNode); err != nil {
				return sdk.WrapError(err, "InsertOrUpdateNode> Unable to delete workflow node %d", oldNode.ID)
			}
		} else {
			isRoot = true
			//Update the root node
			log.Debug("InsertOrUpdateNode> Updating root node %d", oldNode.ID)
			dbwn := Node(*n)
			if _, err := db.Update(&dbwn); err != nil {
				return sdk.WrapError(err, "InsertOrUpdateNode> Unable to update workflow root node")
			}
			if err := DeleteNodeDependencies(db, n); err != nil {
				return sdk.WrapError(err, "InsertOrUpdateNode> Unable to delete workflow root node dependencies")
			}
		}
	}
	if !isRoot {
		//Insert new node
		dbwn := Node(*n)
		if err := db.Insert(&dbwn); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node")
		}
		n.ID = dbwn.ID
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
	for _, h := range n.Hooks {
		if err := InsertOrUpdateHook(db, n, &h); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node trigger")
		}
	}

	//Insert triggers
	for _, t := range n.Triggers {
		if err := InsertOrUpdateTrigger(db, w, n, &t, u); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateNode> Unable to insert workflow node trigger")
		}
	}

	return nil
}

// LoadNode loads a node in a workflow
func LoadNode(db gorp.SqlExecutor, w *sdk.Workflow, id int64, u *sdk.User) (*sdk.WorkflowNode, error) {
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
	triggers, err := LoadTriggers(db, w, &wn, u)
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

// InsertOrUpdateTrigger inserts or updates a trigger
func InsertOrUpdateTrigger(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, trigger *sdk.WorkflowNodeTrigger, u *sdk.User) error {
	trigger.WorkflowNodeID = node.ID
	var oldTrigger *sdk.WorkflowNodeTrigger

	//Try to load the trigger
	if trigger.ID != 0 {
		var err error
		oldTrigger, err = LoadTrigger(db, w, node, trigger.ID, u)
		if err != nil {
			return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to load trigger %d", trigger.ID)
		}
	}

	//Delete the old trigger
	if oldTrigger != nil {
		if err := DeleteTrigger(db, oldTrigger); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to delete trigger %d", trigger.ID)
		}
	}

	//Setup destination node
	if err := InsertOrUpdateNode(db, w, &trigger.WorkflowDestNode, u); err != nil {
		return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to setup destination node")
	}
	trigger.WorkflowDestNodeID = trigger.WorkflowDestNode.ID

	//Insert trigger
	dbt := NodeTrigger(*trigger)
	if err := db.Insert(&dbt); err != nil {
		return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to insert trigger")
	}
	trigger.ID = dbt.ID

	//Manage conditions
	b, err := json.Marshal(trigger.Conditions)
	if err != nil {
		return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to marshal trigger conditions")
	}
	if _, err := db.Exec("UPDATE workflow_node_trigger SET conditions = $1 where id = $2", b, trigger.ID); err != nil {
		return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to set trigger conditions in database")
	}

	return nil
}

// DeleteTrigger deletes a trigger and all chrildren
func DeleteTrigger(db gorp.SqlExecutor, trigger *sdk.WorkflowNodeTrigger) error {
	dbt := NodeTrigger(*trigger)
	if _, err := db.Delete(&dbt); err != nil {
		return sdk.WrapError(err, "DeleteTrigger> Unable to delete trigger %d", dbt.ID)
	}
	return nil
}

// InsertOrUpdateHook inserts or updates a hook
func InsertOrUpdateHook(db gorp.SqlExecutor, node *sdk.WorkflowNode, hook *sdk.WorkflowNodeHook) error {
	return nil
}

// DeleteHook deletes a hook
func DeleteHook(db gorp.SqlExecutor, hook *sdk.WorkflowNodeHook) error {
	return nil
}

//DeleteNode deletes nodes and all its children
func DeleteNode(db gorp.SqlExecutor, node *sdk.WorkflowNode) error {
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
		if err := DeleteTrigger(db, &t); err != nil {
			return sdk.WrapError(err, "DeleteNodeDependencies> Unable to delete trigger %d", t.ID)
		}
	}

	for _, h := range node.Hooks {
		if err := DeleteHook(db, &h); err != nil {
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

// LoadTriggers loads trigger from a node
func LoadTriggers(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, u *sdk.User) ([]sdk.WorkflowNodeTrigger, error) {
	dbtriggers := []NodeTrigger{}
	if _, err := db.Select(&dbtriggers, "select * from workflow_node_trigger where workflow_node_id = $1", node.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "LoadTriggers> Unable to load triggers")
	}

	if len(dbtriggers) == 0 {
		return nil, nil
	}

	triggers := []sdk.WorkflowNodeTrigger{}
	for _, dbt := range dbtriggers {
		t := sdk.WorkflowNodeTrigger(dbt)
		if t.WorkflowDestNodeID != 0 {
			//Load destination node
			dest, err := LoadNode(db, w, t.WorkflowDestNodeID, u)
			if err != nil {
				return nil, sdk.WrapError(err, "LoadTriggers> Unable to load destination node %d", t.WorkflowDestNodeID)
			}
			t.WorkflowDestNode = *dest
		}

		//Load conditions
		sqlConditions, err := db.SelectNullStr("select conditions from workflow_node_trigger where id = $1", t.ID)
		if err != nil {
			return nil, sdk.WrapError(err, "LoadTriggers> Unable to load conditions for trigger %d", t.ID)
		}
		if sqlConditions.Valid {
			if err := json.Unmarshal([]byte(sqlConditions.String), &t.Conditions); err != nil {
				return nil, sdk.WrapError(err, "LoadTriggers> Unable to unmarshall conditions for trigger %d", t.ID)
			}
		}

		triggers = append(triggers, t)
	}
	return triggers, nil
}

// LoadTrigger loads a specific trigger from a node
func LoadTrigger(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, id int64, u *sdk.User) (*sdk.WorkflowNodeTrigger, error) {
	dbtrigger := NodeTrigger{}
	if err := db.SelectOne(&dbtrigger, "select * from workflow_node_trigger where workflow_node_id = $1 and id = $2", node.ID, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "LoadTriggers> Unable to load trigger %d", id)
	}

	t := sdk.WorkflowNodeTrigger(dbtrigger)

	if t.WorkflowDestNodeID != 0 {
		dest, err := LoadNode(db, w, t.WorkflowDestNodeID, u)
		if err != nil {
			return nil, sdk.WrapError(err, "LoadTrigger> Unable to load destination node %d", t.WorkflowDestNodeID)
		}
		t.WorkflowDestNode = *dest
	}

	//Load conditions
	sqlConditions, err := db.SelectNullStr("select conditions from workflow_node_trigger where id = $1", t.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadTriggers> Unable to load conditions for trigger %d", t.ID)
	}
	if sqlConditions.Valid {
		if err := json.Unmarshal([]byte(sqlConditions.String), t.Conditions); err != nil {
			return nil, sdk.WrapError(err, "LoadTriggers> Unable to unmarshall conditions for trigger %d", t.ID)
		}
	}

	return &t, nil
}

// HasAccessTo checks if user has full r, rx or rwx access to the workflow
func HasAccessTo(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) (bool, error) {
	return true, nil
}
