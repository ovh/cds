package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// UpdateLastModifiedDate Update workflow last modified date
func UpdateLastModifiedDate(db gorp.SqlExecutor, store cache.Store, u *sdk.User, projKey string, w *sdk.Workflow) error {
	t := time.Now()
	_, err := db.Exec(`UPDATE workflow set last_modified = current_timestamp WHERE id = $1 RETURNING last_modified`, w.ID)
	w.LastModified = t

	if u != nil {
		updates := sdk.LastModification{
			Key:          projKey,
			Name:         w.Name,
			LastModified: t.Unix(),
			Username:     u.Username,
			Type:         sdk.WorkflowLastModificationType,
		}
		b, errP := json.Marshal(updates)
		if errP == nil {
			store.Publish("lastUpdates", string(b))
		}
		return err
	}

	return nil
}

// PostGet is a db hook
func (w *Workflow) PostGet(db gorp.SqlExecutor) error {
	var res = struct {
		Metadata  sql.NullString `db:"metadata"`
		PurgeTags sql.NullString `db:"purge_tags"`
	}{}

	if err := db.SelectOne(&res, "SELECT metadata, purge_tags FROM workflow WHERE id = $1", w.ID); err != nil {
		return sdk.WrapError(err, "PostGet> Unable to load marshalled workflow")
	}

	metadata := sdk.Metadata{}
	if err := gorpmapping.JSONNullString(res.Metadata, &metadata); err != nil {
		return err
	}
	w.Metadata = metadata

	purgeTags := []string{}
	if err := gorpmapping.JSONNullString(res.PurgeTags, &purgeTags); err != nil {
		return err
	}
	w.PurgeTags = purgeTags

	return nil
}

// PostUpdate is a db hook
func (w *Workflow) PostUpdate(db gorp.SqlExecutor) error {
	b, err := json.Marshal(w.Metadata)
	if err != nil {
		return err
	}
	if _, err := db.Exec("update workflow set metadata = $1 where id = $2", b, w.ID); err != nil {
		return err
	}

	pt, errPt := json.Marshal(w.PurgeTags)
	if errPt != nil {
		return errPt
	}
	if _, err := db.Exec("update workflow set purge_tags = $1 where id = $2", pt, w.ID); err != nil {
		return err
	}

	return nil
}

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
		if err := w.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "LoadAll> Unable to execute post get")
		}
		res = append(res, sdk.Workflow(w))
	}

	return res, nil
}

// LoadAllNames loads all workflow names for a project.
func LoadAllNames(db gorp.SqlExecutor, projID int64, u *sdk.User) ([]string, error) {
	query := `
		SELECT workflow.name
		FROM workflow
		WHERE workflow.project_id = $1
		ORDER BY workflow.name ASC`

	res := []string{}
	if _, err := db.Select(&res, query, projID); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "LoadAllNames> Unable to load workflows with project %s", projID)
	}

	return res, nil
}

// Load loads a workflow for a given user (ie. checking permissions)
func Load(db gorp.SqlExecutor, store cache.Store, projectKey, name string, u *sdk.User) (*sdk.Workflow, error) {
	query := `
		select workflow.*
		from workflow
		join project on project.id = workflow.project_id
		where project.projectkey = $1
		and workflow.name = $2`
	res, err := load(db, store, u, query, projectKey, name)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow %s in project %s", name, projectKey)
	}
	return res, nil
}

// LoadByID loads a workflow for a given user (ie. checking permissions)
func LoadByID(db gorp.SqlExecutor, store cache.Store, id int64, u *sdk.User) (*sdk.Workflow, error) {
	query := `
		select *
		from workflow
		where id = $1`
	res, err := load(db, store, u, query, id)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow %d", id)
	}
	return res, nil
}

// LoadByPipelineName loads a workflow for a given project key and pipeline name (ie. checking permissions)
func LoadByPipelineName(db gorp.SqlExecutor, projectKey string, pipName string) ([]sdk.Workflow, error) {
	dbRes := []Workflow{}
	query := `
		select distinct workflow.*
		from workflow
		join project on project.id = workflow.project_id
		join workflow_node on workflow_node.workflow_id = workflow.id
		join pipeline on pipeline.id = workflow_node.pipeline_id
		where project.projectkey = $1 and pipeline.name = $2
		order by workflow.name asc`

	if _, err := db.Select(&dbRes, query, projectKey, pipName); err != nil {
		if err == sql.ErrNoRows {
			return []sdk.Workflow{}, nil
		}
		return nil, sdk.WrapError(err, "LoadByPipelineName> Unable to load workflows for project %s and pipeline %s", projectKey, pipName)
	}

	res := make([]sdk.Workflow, len(dbRes))
	for i, w := range dbRes {
		w.ProjectKey = projectKey
		res[i] = sdk.Workflow(w)
	}

	return res, nil
}

// LoadByApplicationName loads a workflow for a given project key and application name (ie. checking permissions)
func LoadByApplicationName(db gorp.SqlExecutor, projectKey string, appName string) ([]sdk.Workflow, error) {
	dbRes := []Workflow{}
	query := `
		select distinct workflow.*
		from workflow
		join project on project.id = workflow.project_id
		join workflow_node on workflow_node.workflow_id = workflow.id
		join workflow_node_context on workflow_node_context.workflow_node_id = workflow_node.id
		join application on workflow_node_context.application_id = application.id
		where project.projectkey = $1 and application.name = $2
		order by workflow.name asc`

	if _, err := db.Select(&dbRes, query, projectKey, appName); err != nil {
		if err == sql.ErrNoRows {
			return []sdk.Workflow{}, nil
		}
		return nil, sdk.WrapError(err, "LoadByApplicationName> Unable to load workflows for project %s and application %s", projectKey, appName)
	}

	res := make([]sdk.Workflow, len(dbRes))
	for i, w := range dbRes {
		w.ProjectKey = projectKey
		res[i] = sdk.Workflow(w)
	}

	return res, nil
}

// LoadByEnvName loads a workflow for a given project key and environment name (ie. checking permissions)
func LoadByEnvName(db gorp.SqlExecutor, projectKey string, envName string) ([]sdk.Workflow, error) {
	dbRes := []Workflow{}
	query := `
		select distinct workflow.*
		from workflow
		join project on project.id = workflow.project_id
		join workflow_node on workflow_node.workflow_id = workflow.id
		join workflow_node_context on workflow_node_context.workflow_node_id = workflow_node.id
		join environment on workflow_node_context.environment_id = environment.id
		where project.projectkey = $1 and environment.name = $2
		order by workflow.name asc`

	if _, err := db.Select(&dbRes, query, projectKey, envName); err != nil {
		if err == sql.ErrNoRows {
			return []sdk.Workflow{}, nil
		}
		return nil, sdk.WrapError(err, "LoadByEnvName> Unable to load workflows for project %s and environment %s", projectKey, envName)
	}

	res := make([]sdk.Workflow, len(dbRes))
	for i, w := range dbRes {
		w.ProjectKey = projectKey
		res[i] = sdk.Workflow(w)
	}

	return res, nil
}

func load(db gorp.SqlExecutor, store cache.Store, u *sdk.User, query string, args ...interface{}) (*sdk.Workflow, error) {
	t0 := time.Now()
	dbRes := Workflow{}
	if err := db.SelectOne(&dbRes, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "Load> Unable to load workflow")
	}

	res := sdk.Workflow(dbRes)
	res.ProjectKey, _ = db.SelectStr("select projectkey from project where id = $1", res.ProjectID)
	if err := loadWorkflowRoot(db, store, &res, u); err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow root")
	}

	res.Permission = permission.WorkflowPermission(res.ProjectKey, res.Name, u)

	// Load groups
	gps, err := loadWorkflowGroups(db, res)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow groups")
	}
	res.Groups = gps

	// Load joins
	joins, errJ := loadJoins(db, store, &res, u)
	if errJ != nil {
		return nil, sdk.WrapError(errJ, "Load> Unable to load workflow joins")
	}
	res.Joins = joins

	notifs, errN := loadNotifications(db, &res)
	if errN != nil {
		return nil, sdk.WrapError(errN, "Load> Unable to load workflow notification")
	}
	res.Notifications = notifs

	delta := time.Since(t0).Seconds()

	log.Debug("Load> Load workflow (%s/%s)%d took %.3f seconds", res.ProjectKey, res.Name, res.ID, delta)
	w := &res
	Sort(w)
	return w, nil
}

func loadWorkflowRoot(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, u *sdk.User) error {
	var err error
	w.Root, err = loadNode(db, store, w, w.RootID, u)
	if err != nil {
		if err == sdk.ErrWorkflowNodeNotFound {
			log.Debug("Load> Unable to load root %d for workflow %d", w.RootID, w.ID)
			return nil
		}
		return sdk.WrapError(err, "Load> Unable to load workflow root %d", w.RootID)
	}
	return nil
}

// Insert inserts a new workflow
func Insert(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, p *sdk.Project, u *sdk.User) error {
	if err := IsValid(w, p); err != nil {
		return err
	}

	w.LastModified = time.Now()
	if err := db.QueryRow("INSERT INTO workflow (name, description, project_id) VALUES ($1, $2, $3) RETURNING id", w.Name, w.Description, w.ProjectID).Scan(&w.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow %s/%s", w.ProjectKey, w.Name)
	}

	if w.Root == nil {
		return sdk.ErrWorkflowInvalidRoot
	}

	if err := renameNode(db, w); err != nil {
		return sdk.WrapError(err, "Insert> Cannot rename node")
	}

	if err := insertNode(db, store, w, w.Root, u, false); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow root node")
	}

	if _, err := db.Exec("UPDATE workflow SET root_node_id = $2 WHERE id = $1", w.ID, w.Root.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow (%#v, %d)", w.Root, w.ID)
	}

	nodes := w.Nodes(true)
	for i := range w.Joins {
		j := &w.Joins[i]
		var err error
		nodes, err = insertJoin(db, store, w, j, nodes, u)
		if err != nil {
			return sdk.WrapError(err, "Insert> Unable to insert update workflow(%d) join (%#v)", w.ID, j)
		}
	}

	for i := range w.Notifications {
		n := &w.Notifications[i]
		if err := insertNotification(db, store, w, n, nodes, u); err != nil {
			return sdk.WrapError(err, "Insert> Unable to insert update workflow(%d) notification (%#v)", w.ID, n)
		}
	}

	return updateLastModified(db, store, w, u)
}

func renameNode(db gorp.SqlExecutor, w *sdk.Workflow) error {
	nameByPipeline := map[int64][]*sdk.WorkflowNode{}
	maxNumberByPipeline := map[int64]int64{}

	// browse node
	if err := saveNodeByPipeline(db, &nameByPipeline, &maxNumberByPipeline, w.Root); err != nil {
		return err
	}

	// browse join
	for i := range w.Joins {
		join := &w.Joins[i]
		for j := range join.Triggers {
			if err := saveNodeByPipeline(db, &nameByPipeline, &maxNumberByPipeline, &join.Triggers[j].WorkflowDestNode); err != nil {
				return err
			}
		}
	}

	// Generate node name
	for _, v := range nameByPipeline {
		for _, n := range v {
			if n.Name == "" {
				nextNumber := maxNumberByPipeline[n.Pipeline.ID] + 1
				if nextNumber > 1 {
					n.Name = fmt.Sprintf("%s_%d", n.Pipeline.Name, nextNumber)
				} else {
					n.Name = n.Pipeline.Name
				}
				maxNumberByPipeline[n.Pipeline.ID] = nextNumber
			}
		}
	}

	return nil
}

func saveNodeByPipeline(db gorp.SqlExecutor, dict *map[int64][]*sdk.WorkflowNode, mapMaxNumber *map[int64]int64, n *sdk.WorkflowNode) error {
	// get pipeline ID
	if n.Pipeline.ID == 0 {
		n.Pipeline.ID = n.PipelineID
	} else if n.PipelineID == 0 {
		n.PipelineID = n.Pipeline.ID
	}

	// Load pipeline to have name
	if n.Pipeline.Name == "" {
		pip, errorP := pipeline.LoadPipelineByID(db, n.PipelineID, false)
		if errorP != nil {
			return sdk.WrapError(errorP, "saveNodeByPipeline> Cannot load pipeline %d", n.PipelineID)
		}
		n.Pipeline = *pip
	}

	// Save node in pipeline node map
	if _, ok := (*dict)[n.PipelineID]; !ok {
		(*dict)[n.PipelineID] = []*sdk.WorkflowNode{}
	}
	(*dict)[n.PipelineID] = append((*dict)[n.PipelineID], n)

	// Check max number for current pipeline
	if n.Name == n.Pipeline.Name || (n.Name != "" && strings.HasPrefix(n.Name, n.Pipeline.Name+"_")) {
		pipNumber, errI := strconv.ParseInt(strings.Replace(n.Name, n.Pipeline.Name+"_", "", 1), 10, 64)

		if n.Name == n.Pipeline.Name {
			pipNumber = 1
		}

		if errI == nil || pipNumber == 1 {
			currentMax, ok := (*mapMaxNumber)[n.PipelineID]
			if !ok || currentMax < pipNumber {
				(*mapMaxNumber)[n.PipelineID] = pipNumber
			}
		}
	}

	for k := range n.Triggers {
		if err := saveNodeByPipeline(db, dict, mapMaxNumber, &n.Triggers[k].WorkflowDestNode); err != nil {
			return err
		}
	}
	return nil
}

// Update updates a workflow
func Update(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, oldWorkflow *sdk.Workflow, p *sdk.Project, u *sdk.User) error {
	if err := IsValid(w, p); err != nil {
		return err
	}

	if err := renameNode(db, w); err != nil {
		return sdk.WrapError(err, "Update> cannot check pipeline name")
	}

	// Delete all OLD JOIN
	for _, j := range oldWorkflow.Joins {
		if err := deleteJoin(db, j); err != nil {
			return sdk.WrapError(err, "Update> unable to delete all join on workflow(%d)", w.ID)
		}
	}

	if err := deleteNotifications(db, oldWorkflow.ID); err != nil {
		return sdk.WrapError(err, "Update> unable to delete all notifications on workflow(%d)", w.ID)
	}

	// Delete old Root Node
	if oldWorkflow.Root != nil {
		if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
			return sdk.WrapError(err, "Delete> Unable to detache workflow root")
		}
		if err := deleteNode(db, oldWorkflow, oldWorkflow.Root, u); err != nil {
			return sdk.WrapError(err, "Update> unable to delete root node on workflow(%d)", w.ID)
		}
	}

	// Inser new Root Node
	if err := insertNode(db, store, w, w.Root, u, false); err != nil {
		return sdk.WrapError(err, "Update> unable to update root node on workflow(%d)", w.ID)
	}

	w.RootID = w.Root.ID
	nodes := w.Nodes(true)

	// Insert new JOIN
	for i := range w.Joins {
		j := &w.Joins[i]
		var err error
		nodes, err = insertJoin(db, store, w, j, nodes, u)
		if err != nil {
			return sdk.WrapError(err, "Update> Unable to update workflow(%d) join (%#v)", w.ID, j)
		}
	}

	for i := range w.Notifications {
		n := &w.Notifications[i]
		if err := insertNotification(db, store, w, n, nodes, u); err != nil {
			return sdk.WrapError(err, "Update> Unable to update workflow(%d) notification (%#v)", w.ID, n)
		}
	}

	w.LastModified = time.Now()
	dbw := Workflow(*w)
	if _, err := db.Update(&dbw); err != nil {
		return sdk.WrapError(err, "Update> Unable to update workflow")
	}

	return updateLastModified(db, store, w, u)
}

// Delete workflow
func Delete(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, u *sdk.User) error {
	//Detach root from workflow
	if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
		return sdk.WrapError(err, "Delete> Unable to detache workflow root")
	}

	hooks := w.GetHooks()
	// Delete all hooks
	if err := deleteHookConfiguration(db, store, p, hooks); err != nil {
		return sdk.WrapError(err, "Delete> Unable to delete hooks from workflow")
	}

	// Delete all JOINs
	for _, j := range w.Joins {
		if err := deleteJoin(db, j); err != nil {
			return sdk.WrapError(err, "Update> unable to delete all join on workflow(%d)", w.ID)
		}
	}

	//Delete root
	if err := deleteNode(db, w, w.Root, u); err != nil {
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
func updateLastModified(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, u *sdk.User) error {
	t := time.Now()
	if u != nil {
		store.SetWithTTL(cache.Key("lastModified", "workflow", fmt.Sprintf("%d", w.ID)), sdk.LastModification{
			Name:         w.Name,
			Username:     u.Username,
			LastModified: t.Unix(),
		}, 0)
	}
	return nil
}

// HasAccessTo checks if user has full r, rx or rwx access to the workflow
func HasAccessTo(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) (bool, error) {
	return true, nil
}

// IsValid cheks workflow validity
func IsValid(w *sdk.Workflow, proj *sdk.Project) error {
	//Check project is not empty
	if w.ProjectKey == "" {
		return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Invalid project key"))
	}

	//Check workflow name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(w.Name) {
		return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Invalid workflow name. It should match %s", sdk.NamePattern))
	}

	//Check duplicate refs
	refs := w.References()
	for i, ref1 := range refs {
		for j, ref2 := range refs {
			if ref1 == ref2 && i != j {
				return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Duplicate reference %s", ref1))
			}
		}
	}

	//Check refs
	for _, j := range w.Joins {
		if len(j.SourceNodeRefs) == 0 {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Source node references is mandatory"))
		}
	}

	//Checks application are in the current project
	apps := w.InvolvedApplications()
	for _, appID := range apps {
		var found bool
		for _, a := range proj.Applications {
			if appID == a.ID {
				found = true
				break
			}
		}
		if !found {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Unknown application %d", appID))
		}
	}

	//Checks pipelines are in the current project
	pips := w.InvolvedPipelines()
	for _, pipID := range pips {
		var found bool
		for _, p := range proj.Pipelines {
			if pipID == p.ID {
				found = true
				break
			}
		}
		if !found {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Unknown pipeline %d", pipID))
		}
	}

	//Checks environments are in the current project
	envs := w.InvolvedEnvironments()
	for _, envID := range envs {
		var found bool
		for _, e := range proj.Environments {
			if envID == e.ID {
				found = true
				break
			}
		}
		if !found {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Unknown environments %d", envID))
		}
	}

	//Checks hooks conditions
	hooks := w.GetHooks()
	for _, h := range hooks {
		for k := range h.Config {
			switch k {
			case "project", "workflow":
				return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Invalid hooks (%s) configuration %s on node %s", h.WorkflowHookModel.Name, h.UUID, w.GetNode(h.WorkflowNodeID).Name))
			}
		}
	}

	return nil
}
