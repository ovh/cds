package workflow

import (
	"archive/tar"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/tracing"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// LoadOptions custom option for loading workflow
type LoadOptions struct {
	DeepPipeline  bool
	WithoutNode   bool
	Base64Keys    bool
	OnlyRootNode  bool
	WithFavorites bool
	WithIcon      bool
}

// CountVarInWorkflowData represents the result of CountVariableInWorkflow function
type CountVarInWorkflowData struct {
	WorkflowName string `db:"workflow_name"`
	NodeName     string `db:"node_name"`
}

// Exists checks if a workflow exists
func Exists(db gorp.SqlExecutor, key string, name string) (bool, error) {
	query := `
		select count(1)
		from workflow
		join project on project.id = workflow.project_id
		where project.projectkey = $1
		and workflow.name = $2`
	count, err := db.SelectInt(query, key, name)
	if err != nil {
		return false, sdk.WrapError(err, "Exists>")
	}
	return count > 0, nil
}

// CountVariableInWorkflow counts how many time the given variable is used on all workflows of the given project
func CountVariableInWorkflow(db gorp.SqlExecutor, projectKey string, varName string) ([]CountVarInWorkflowData, error) {
	query := `
		SELECT DISTINCT workflow.name as workflow_name, workflow_node.name as node_name
		FROM workflow
		JOIN project ON project.id = workflow.project_id
		JOIN workflow_node ON workflow_node.workflow_id = workflow.id
		JOIN workflow_node_context ON workflow_node_context.workflow_node_id = workflow_node.id
		WHERE project.projectkey = $1
		AND (
			workflow_node_context.default_pipeline_parameters::TEXT LIKE $2
			OR
			workflow_node_context.default_payload::TEXT LIKE $2
		);
	`
	var datas []CountVarInWorkflowData
	if _, err := db.Select(&datas, query, projectKey, fmt.Sprintf("%%%s%%", varName)); err != nil {
		return nil, sdk.WrapError(err, "CountVariableInWorkflow> Unable to count var in workflow")
	}
	return datas, nil
}

// UpdateMetadata update the metadata of a workflow
func UpdateMetadata(db gorp.SqlExecutor, workflowID int64, metadata sdk.Metadata) error {
	b, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	if _, err := db.Exec("update workflow set metadata = $1 where id = $2", b, workflowID); err != nil {
		return err
	}

	return nil
}

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

// PreInsert is a db hook
func (w *Workflow) PreInsert(db gorp.SqlExecutor) error {
	return w.PreUpdate(db)
}

// PostInsert is a db hook
func (w *Workflow) PostInsert(db gorp.SqlExecutor) error {
	return w.PostUpdate(db)
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

// PreUpdate is a db hook
func (w *Workflow) PreUpdate(db gorp.SqlExecutor) error {
	if w.FromRepository != "" && strings.HasPrefix(w.FromRepository, "http") {
		fromRepoURL, err := url.Parse(w.FromRepository)
		if err != nil {
			return sdk.WrapError(err, "Workflow.PreUpdate> Cannot parse url %s", w.FromRepository)
		}
		fromRepoURL.User = nil
		w.FromRepository = fromRepoURL.String()
	}

	return nil
}

// PostUpdate is a db hook
func (w *Workflow) PostUpdate(db gorp.SqlExecutor) error {
	if err := UpdateMetadata(db, w.ID, w.Metadata); err != nil {
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
		and workflow.to_delete = false
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
func LoadAllNames(db gorp.SqlExecutor, projID int64, u *sdk.User) ([]sdk.IDName, error) {
	query := `
		SELECT workflow.name, workflow.id, workflow.description, workflow.icon
		FROM workflow
		WHERE workflow.project_id = $1
		AND workflow.to_delete = false
		ORDER BY workflow.name ASC`

	var res []sdk.IDName
	if _, err := db.Select(&res, query, projID); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "LoadAllNames> Unable to load workflows with project %s", projID)
	}

	return res, nil
}

// Load loads a workflow for a given user (ie. checking permissions)
func Load(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, name string, u *sdk.User, opts LoadOptions) (*sdk.Workflow, error) {
	ctx, end := tracing.Span(ctx, "workflow.Load",
		tracing.Tag("workflow", name),
		tracing.Tag("project_key", proj.Key),
		tracing.Tag("with_pipeline", opts.DeepPipeline),
		tracing.Tag("only_root", opts.OnlyRootNode),
		tracing.Tag("with_base64_keys", opts.Base64Keys),
		tracing.Tag("without_node", opts.WithoutNode),
	)
	defer end()

	var icon string
	if opts.WithIcon {
		icon = "workflow.icon,"
	}
	query := fmt.Sprintf(`
		select workflow.id,
		workflow.project_id,
		workflow.name,
		workflow.description,
		%s
		workflow.last_modified,
		workflow.root_node_id,
		workflow.metadata,
		workflow.history_length,
		workflow.purge_tags,
		workflow.from_repository,
		workflow.derived_from_workflow_id,
		workflow.derived_from_workflow_name,
		workflow.derivation_branch,
		workflow.to_delete
		from workflow
		join project on project.id = workflow.project_id
		where project.projectkey = $1
		and workflow.name = $2`, icon)
	res, err := load(ctx, db, store, proj, opts, u, query, proj.Key, name)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow %s in project %s", name, proj.Key)
	}
	res.ProjectKey = proj.Key
	return res, nil
}

// LoadByID loads a workflow for a given user (ie. checking permissions)
func LoadByID(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, id int64, u *sdk.User, opts LoadOptions) (*sdk.Workflow, error) {
	query := `
		select *
		from workflow
		where id = $1`
	res, err := load(context.TODO(), db, store, proj, opts, u, query, id)
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
		and workflow.to_delete = false
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
		and workflow.to_delete = false
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
		and workflow.to_delete = false
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

func load(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, opts LoadOptions, u *sdk.User, query string, args ...interface{}) (*sdk.Workflow, error) {
	t0 := time.Now()
	dbRes := Workflow{}

	_, next := tracing.Span(ctx, "workflow.load.selectOne")
	if err := db.SelectOne(&dbRes, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "Load> Unable to load workflow")
	}
	next()

	res := sdk.Workflow(dbRes)
	if proj.Key == "" {
		res.ProjectKey, _ = db.SelectStr("select projectkey from project where id = $1", res.ProjectID)
	} else {
		res.ProjectKey = proj.Key
	}

	if u != nil {
		res.Permission = permission.WorkflowPermission(res.ProjectKey, res.Name, u)
	}

	// Load groups
	_, next = tracing.Span(ctx, "workflow.load.loadWorkflowGroups")
	gps, err := loadWorkflowGroups(db, res)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow groups")
	}
	res.Groups = gps
	next()

	res.Pipelines = map[int64]sdk.Pipeline{}

	if !opts.WithoutNode {
		_, next = tracing.Span(ctx, "workflow.load.loadNodes")
		err := loadWorkflowRoot(ctx, db, store, proj, &res, u, opts)
		next()

		if err != nil {
			return nil, sdk.WrapError(err, "Load> Unable to load workflow root")
		}

		// Load joins
		if !opts.OnlyRootNode {
			_, next = tracing.Span(ctx, "workflow.load.loadJoins")
			joins, errJ := loadJoins(ctx, db, store, proj, &res, u, opts)
			next()

			if errJ != nil {
				return nil, sdk.WrapError(errJ, "Load> Unable to load workflow joins")
			}
			res.Joins = joins
		}

	}

	if opts.WithFavorites {
		_, next = tracing.Span(ctx, "workflow.load.loadFavorite")
		fav, errF := loadFavorite(db, &res, u)
		next()

		if errF != nil {
			return nil, sdk.WrapError(errF, "Load> unable to load favorite")
		}
		res.Favorite = fav
	}

	_, next = tracing.Span(ctx, "workflow.load.loadNotifications")
	notifs, errN := loadNotifications(db, &res)
	next()

	if errN != nil {
		return nil, sdk.WrapError(errN, "Load> Unable to load workflow notification")
	}
	res.Notifications = notifs

	delta := time.Since(t0).Seconds()

	log.Debug("Load> Load workflow (%s/%s)%d took %.3f seconds", res.ProjectKey, res.Name, res.ID, delta)
	w := &res
	if !opts.WithoutNode {
		_, next = tracing.Span(ctx, "workflow.load.Sort")
		Sort(w)
		next()
	}
	return w, nil
}

func loadWorkflowRoot(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, u *sdk.User, opts LoadOptions) error {
	var err error
	w.Root, err = loadNode(ctx, db, store, proj, w, w.RootID, u, opts)
	if err != nil {
		if err == sdk.ErrWorkflowNodeNotFound {
			log.Debug("Load> Unable to load root %d for workflow %d", w.RootID, w.ID)
			return nil
		}
		return sdk.WrapError(err, "Load> Unable to load workflow root %d", w.RootID)
	}
	return nil
}

func loadFavorite(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) (bool, error) {
	count, err := db.SelectInt("SELECT COUNT(1) FROM workflow_favorite WHERE user_id = $1 AND workflow_id = $2", u.ID, w.ID)
	if err != nil {
		return false, sdk.WrapError(err, "workflow.loadFavorite>")
	}
	return count > 0, nil
}

// Insert inserts a new workflow
func Insert(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, p *sdk.Project, u *sdk.User) error {
	if err := IsValid(w, p); err != nil {
		return err
	}

	if w.HistoryLength == 0 {
		w.HistoryLength = sdk.DefaultHistoryLength
	}

	w.LastModified = time.Now()
	if err := db.QueryRow("INSERT INTO workflow (name, description, icon, project_id, history_length, from_repository) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id", w.Name, w.Description, w.Icon, w.ProjectID, w.HistoryLength, w.FromRepository).Scan(&w.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow %s/%s", w.ProjectKey, w.Name)
	}

	dbw := Workflow(*w)
	if err := dbw.PostInsert(db); err != nil {
		return sdk.WrapError(err, "Insert> Cannot post insert hook")
	}

	if w.Root == nil {
		return sdk.ErrWorkflowInvalidRoot
	}

	if err := renameNode(db, w); err != nil {
		return sdk.WrapError(err, "Insert> Cannot rename node")
	}

	if errIN := insertNode(db, store, w, w.Root, u, false); errIN != nil {
		return sdk.WrapError(errIN, "Insert> Unable to insert workflow root node")
	}
	w.RootID = w.Root.ID

	if w.Root.IsLinkedToRepo() {
		if w.Metadata == nil {
			w.Metadata = sdk.Metadata{}
		}
		w.Metadata["default_tags"] = "git.branch,git.author"

		if err := UpdateMetadata(db, w.ID, w.Metadata); err != nil {
			return sdk.WrapError(err, "Insert> Unable to insert workflow metadata (%#v, %d)", w.Root, w.ID)
		}
	}

	if _, err := db.Exec("UPDATE workflow SET root_node_id = $2 WHERE id = $1", w.ID, w.Root.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow (%#v, %d)", w.Root, w.ID)
	}

	for i := range w.Joins {
		j := &w.Joins[i]
		if err := insertJoin(db, store, w, j, u); err != nil {
			return sdk.WrapError(err, "Insert> Unable to insert update workflow(%d) join (%#v)", w.ID, j)
		}
	}

	nodes := w.Nodes(true)
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
	if err := saveNodeByPipeline(db, &nameByPipeline, &maxNumberByPipeline, w.Root, w); err != nil {
		return err
	}

	// browse join
	for i := range w.Joins {
		join := &w.Joins[i]
		for j := range join.Triggers {
			if err := saveNodeByPipeline(db, &nameByPipeline, &maxNumberByPipeline, &join.Triggers[j].WorkflowDestNode, w); err != nil {
				return err
			}
		}
	}

	// Generate node name
	for _, v := range nameByPipeline {
		for _, n := range v {
			if n.Name == "" {
				nextNumber := maxNumberByPipeline[n.PipelineID] + 1
				if nextNumber > 1 {
					n.Name = fmt.Sprintf("%s_%d", w.Pipelines[n.PipelineID].Name, nextNumber)
				} else {
					n.Name = w.Pipelines[n.PipelineID].Name
				}
				maxNumberByPipeline[n.PipelineID] = nextNumber
			}
		}
	}

	return nil
}

func saveNodeByPipeline(db gorp.SqlExecutor, dict *map[int64][]*sdk.WorkflowNode, mapMaxNumber *map[int64]int64, n *sdk.WorkflowNode, w *sdk.Workflow) error {
	// Load pipeline to have name
	pip, has := w.Pipelines[n.PipelineID]
	if !has {
		pip2, errorP := pipeline.LoadPipelineByID(context.TODO(), db, n.PipelineID, true)
		if errorP != nil {
			return sdk.WrapError(errorP, "saveNodeByPipeline> Cannot load pipeline %d", n.PipelineID)
		}
		if w.Pipelines == nil {
			w.Pipelines = map[int64]sdk.Pipeline{}
		}
		w.Pipelines[n.PipelineID] = *pip2
		pip = *pip2
	}
	n.PipelineName = pip.Name

	// Save node in pipeline node map
	if _, ok := (*dict)[n.PipelineID]; !ok {
		(*dict)[n.PipelineID] = []*sdk.WorkflowNode{}
	}
	(*dict)[n.PipelineID] = append((*dict)[n.PipelineID], n)

	// Check max number for current pipeline
	if n.Name == n.PipelineName || (n.Name != "" && strings.HasPrefix(n.Name, n.PipelineName+"_")) {
		pipNumber, errI := strconv.ParseInt(strings.Replace(n.Name, n.PipelineName+"_", "", 1), 10, 64)

		if n.Name == n.PipelineName {
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
		if err := saveNodeByPipeline(db, dict, mapMaxNumber, &n.Triggers[k].WorkflowDestNode, w); err != nil {
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
			return sdk.WrapError(err, "Update> unable to delete all joins on workflow(%d)", w.ID)
		}
	}

	if err := deleteNotifications(db, oldWorkflow.ID); err != nil {
		return sdk.WrapError(err, "Update> unable to delete all notifications on workflow(%d)", w.ID)
	}

	// Delete old Root Node
	if oldWorkflow.Root != nil {
		if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
			return sdk.WrapError(err, "Delete> Unable to detach workflow root")
		}

		if err := deleteNode(db, oldWorkflow, oldWorkflow.Root); err != nil {
			return sdk.WrapError(err, "Update> unable to delete root node on workflow(%d)", w.ID)
		}
	}

	// Delete all node ID
	w.ResetIDs()

	if err := insertNode(db, store, w, w.Root, u, false); err != nil {
		return sdk.WrapError(err, "Update> unable to update root node on workflow(%d)", w.ID)
	}
	w.RootID = w.Root.ID

	// Insert new JOIN
	for i := range w.Joins {
		j := &w.Joins[i]
		if err := insertJoin(db, store, w, j, u); err != nil {
			return sdk.WrapError(err, "Update> Unable to update workflow(%d) join (%#v)", w.ID, j)
		}
	}

	nodes := w.Nodes(true)
	for i := range w.Notifications {
		n := &w.Notifications[i]
		if err := insertNotification(db, store, w, n, nodes, u); err != nil {
			return sdk.WrapError(err, "Update> Unable to update workflow(%d) notification (%#v)", w.ID, n)
		}
	}

	filteredPurgeTags := []string{}
	for _, t := range w.PurgeTags {
		if t != "" {
			filteredPurgeTags = append(filteredPurgeTags, t)
		}
	}
	w.PurgeTags = filteredPurgeTags

	if w.Icon == "" {
		w.Icon = oldWorkflow.Icon
	}

	w.LastModified = time.Now()
	dbw := Workflow(*w)
	if _, err := db.Update(&dbw); err != nil {
		return sdk.WrapError(err, "Update> Unable to update workflow")
	}

	return updateLastModified(db, store, w, u)
}

// MarkAsDelete marks a workflow to be deleted
func MarkAsDelete(db gorp.SqlExecutor, w *sdk.Workflow) error {
	if _, err := db.Exec("update workflow set to_delete = true where id = $1", w.ID); err != nil {
		return sdk.WrapError(err, "MarkAsDelete> Unable to mark as delete workflow id %d", w.ID)
	}
	return nil
}

// Delete workflow
func Delete(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow) error {
	log.Debug("Delete> deleting workflow %d", w.ID)

	//Detach root from workflow
	if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
		return sdk.WrapError(err, "Delete> Unable to detach workflow root")
	}

	hooks := w.GetHooks()
	// Delete all hooks
	if err := DeleteHookConfiguration(db, store, p, hooks); err != nil {
		return sdk.WrapError(err, "Delete> Unable to delete hooks from workflow")
	}

	// Delete all JOINs
	for _, j := range w.Joins {
		if err := deleteJoin(db, j); err != nil {
			return sdk.WrapError(err, "Delete> unable to delete all join on workflow(%d)", w.ID)
		}
	}

	//Delete root
	if err := deleteNode(db, w, w.Root); err != nil {
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

// IsValid cheks workflow validity
func IsValid(w *sdk.Workflow, proj *sdk.Project) error {
	//Check project is not empty
	if w.ProjectKey == "" {
		return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Invalid project key"))
	}

	if w.Icon != "" {
		if !strings.HasPrefix(w.Icon, sdk.IconFormat) {
			return sdk.ErrIconBadFormat
		}
		if len(w.Icon) > sdk.MaxIconSize {
			return sdk.ErrIconBadSize
		}
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

	//Checks platforms are in the current project
	pfs := w.InvolvedPlatforms()
	for _, id := range pfs {
		var found bool
		for _, p := range proj.Platforms {
			if id == p.ID {
				found = true
				break
			}
		}
		if !found {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Unknown platforms %d", id))
		}
	}

	//Check contexts
	nodes := w.Nodes(true)
	for _, n := range nodes {
		if err := n.CheckApplicationDeploymentStrategies(proj); err != nil {
			return sdk.NewError(sdk.ErrWorkflowInvalid, err)
		}
	}

	return nil
}

// Push push a workflow from cds files
func Push(ctx context.Context, db *gorp.DbMap, store cache.Store, proj *sdk.Project, tr *tar.Reader, opts *PushOption, u *sdk.User, decryptFunc keys.DecryptFunc) ([]sdk.Message, *sdk.Workflow, error) {
	ctx, end := tracing.Span(ctx, "workflow.Push")
	defer end()

	apps := make(map[string]exportentities.Application)
	pips := make(map[string]exportentities.PipelineV1)
	envs := make(map[string]exportentities.Environment)
	var wrkflw exportentities.Workflow

	mError := new(sdk.MultiError)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			err = sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Unable to read tar file"))
			return nil, nil, sdk.WrapError(err, "Push>")
		}

		log.Debug("Push> Reading %s", hdr.Name)

		buff := new(bytes.Buffer)
		if _, err := io.Copy(buff, tr); err != nil {
			err = sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Unable to read tar file"))
			return nil, nil, sdk.WrapError(err, "Push>")
		}

		b := buff.Bytes()
		switch {
		case strings.Contains(hdr.Name, ".app."):
			var app exportentities.Application
			if err := yaml.Unmarshal(b, &app); err != nil {
				log.Error("Push> Unable to unmarshal application %s: %v", hdr.Name, err)
				mError.Append(fmt.Errorf("Unable to unmarshal application %s: %v", hdr.Name, err))
				continue
			}
			apps[hdr.Name] = app
		case strings.Contains(hdr.Name, ".pip."):
			var pip exportentities.PipelineV1
			if err := yaml.Unmarshal(b, &pip); err != nil {
				log.Error("Push> Unable to unmarshal pipeline %s: %v", hdr.Name, err)
				mError.Append(fmt.Errorf("Unable to unmarshal pipeline %s: %v", hdr.Name, err))
				continue
			}
			pips[hdr.Name] = pip
		case strings.Contains(hdr.Name, ".env."):
			var env exportentities.Environment
			if err := yaml.Unmarshal(b, &env); err != nil {
				log.Error("Push> Unable to unmarshal environment %s: %v", hdr.Name, err)
				mError.Append(fmt.Errorf("Unable to unmarshal environment %s: %v", hdr.Name, err))
				continue
			}
			envs[hdr.Name] = env
		default:
			if err := yaml.Unmarshal(b, &wrkflw); err != nil {
				log.Error("Push> Unable to unmarshal workflow %s: %v", hdr.Name, err)
				mError.Append(fmt.Errorf("Unable to unmarshal workflow %s: %v", hdr.Name, err))
				continue
			}
		}
	}

	// We only use the multiError the une unmarshalling steps.
	// When a DB transaction has been started, just return at the first error
	// because transaction may have to be aborted
	if !mError.IsEmpty() {
		return nil, nil, sdk.NewError(sdk.ErrWorkflowInvalid, mError)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, nil, sdk.WrapError(err, "Push> Unable to start tx")
	}
	defer tx.Rollback()

	allMsg := []sdk.Message{}
	for filename, app := range apps {
		log.Debug("Push> Parsing %s", filename)
		appDB, msgList, err := application.ParseAndImport(tx, store, proj, &app, true, decryptFunc, u)
		if err != nil {
			err = fmt.Errorf("unable to import application %s: %v", app.Name, err)
			return nil, nil, sdk.NewError(sdk.ErrWrongRequest, err)
		}
		allMsg = append(allMsg, msgList...)

		// Update application data on project
		found := false
		for i, a := range proj.Applications {
			if a.Name == appDB.Name {
				proj.Applications[i] = *appDB
				found = true
				break
			}
		}
		if !found {
			proj.Applications = append(proj.Applications, *appDB)
		}

		log.Debug("Push> -- %s OK", filename)
	}

	for filename, env := range envs {
		log.Debug("Push> Parsing %s", filename)
		envDB, msgList, err := environment.ParseAndImport(tx, store, proj, &env, true, decryptFunc, u)
		if err != nil {
			err = fmt.Errorf("unable to import environment %s: %v", env.Name, err)
			return nil, nil, sdk.NewError(sdk.ErrWrongRequest, err)
		}
		allMsg = append(allMsg, msgList...)

		// Update environment data on project
		found := false
		for i, e := range proj.Environments {
			if e.Name == envDB.Name {
				proj.Environments[i] = *envDB
				found = true
				break
			}
		}
		if !found {
			proj.Environments = append(proj.Environments, *envDB)
		}

		log.Debug("Push> -- %s OK", filename)
	}

	for filename, pip := range pips {
		log.Debug("Push> Parsing %s", filename)
		pipDB, msgList, err := pipeline.ParseAndImport(tx, store, proj, &pip, u, pipeline.ImportOptions{Force: true})
		if err != nil {
			err = fmt.Errorf("unable to import pipeline %s: %v", pip.Name, err)
			return nil, nil, sdk.NewError(sdk.ErrWrongRequest, err)
		}
		allMsg = append(allMsg, msgList...)

		// Update pipeline data on project
		found := false
		for i, pi := range proj.Pipelines {
			if pi.Name == pipDB.Name {
				proj.Pipelines[i] = *pipDB
				found = true
				break
			}
		}
		if !found {
			proj.Pipelines = append(proj.Pipelines, *pipDB)
		}

		log.Debug("Push> -- %s OK", filename)
	}

	var dryRun bool
	if opts != nil {
		dryRun = opts.DryRun
	}

	wf, msgList, err := ParseAndImport(ctx, tx, store, proj, &wrkflw, u, ImportOptions{DryRun: dryRun, Force: true})
	if err != nil {
		log.Error("Push> Unable to import workflow: %v", err)
		err = sdk.SetError(err, "unable to import workflow %s", wrkflw.Name)
		return nil, nil, sdk.WrapError(err, "Push> %v ", err)
	}

	// TODO workflow as code, manage derivation workflow
	if opts != nil {
		wf.FromRepository = opts.FromRepository
		if !opts.IsDefaultBranch {
			wf.DerivationBranch = opts.Branch
		}
		if wf.Root.Context.Application != nil {
			wf.Root.Context.Application.VCSServer = opts.VCSServer
			wf.Root.Context.Application.RepositoryFullname = opts.RepositoryName
			wf.Root.Context.Application.RepositoryStrategy = opts.RepositoryStrategy
		}

		if wf.FromRepository != "" {
			if len(wf.Root.Hooks) == 0 {
				wf.Root.Hooks = append(wf.Root.Hooks, sdk.WorkflowNodeHook{
					WorkflowHookModel: sdk.RepositoryWebHookModel,
					Config:            sdk.RepositoryWebHookModel.DefaultConfig,
					UUID:              opts.HookUUID,
				})
				if wf.Root.Context.DefaultPayload, err = DefaultPayload(tx, store, proj, u, wf); err != nil {
					return nil, nil, sdk.WrapError(err, "Push> Unable to get default payload")
				}
			}

			if wf.Root.Context.Application != nil {
				if err := application.Update(tx, store, wf.Root.Context.Application, u); err != nil {
					return nil, nil, sdk.WrapError(err, "Push> Unable to update application vcs datas")
				}
			}
		}

		if err := Update(tx, store, wf, wf, proj, u); err != nil {
			return nil, nil, sdk.WrapError(err, "Push> Unable to update workflow", err)
		}

		if !opts.DryRun {
			if errHr := HookRegistration(tx, store, nil, *wf, proj); errHr != nil {
				return nil, nil, sdk.WrapError(errHr, "Push> hook registration failed")
			}
		}

	}

	allMsg = append(allMsg, msgList...)

	isDefaultBranch := false
	if opts != nil {
		isDefaultBranch = opts.IsDefaultBranch
	}
	if dryRun && !isDefaultBranch {
		_ = tx.Rollback()
	} else {
		if err := tx.Commit(); err != nil {
			return nil, nil, sdk.WrapError(err, "Push> Cannot commit transaction")
		}
	}

	return allMsg, wf, nil
}

// UpdateFavorite add or delete workflow from user favorites
func UpdateFavorite(db gorp.SqlExecutor, workflowID int64, u *sdk.User, add bool) error {
	var query string
	if add {
		query = "INSERT INTO workflow_favorite (user_id, workflow_id) VALUES ($1, $2)"
	} else {
		query = "DELETE FROM workflow_favorite WHERE user_id = $1 AND workflow_id = $2"
	}

	_, err := db.Exec(query, u.ID, workflowID)
	return sdk.WrapError(err, "UpdateFavorite>")
}

// LoadAllByPlatformName loads all workflow using a platform (project_platform table)
func LoadAllByPlatformName(db gorp.SqlExecutor, projID int64, pfName string) ([]sdk.Workflow, error) {
	query := `
	SELECT workflow.*
	FROM workflow
	JOIN workflow_node ON workflow.id = workflow_node.workflow_id
	JOIN workflow_node_context ON workflow_node.id = workflow_node_context.workflow_node_id
	JOIN project_platform ON project_platform.project_id = workflow.project_id
	WHERE workflow.project_id = $1
	AND project_platform.name = $2
	AND workflow_node_context.project_platform_id = project_platform.id
	`

	res := []sdk.Workflow{}
	dbRes := []Workflow{}
	if _, err := db.Select(&dbRes, query, projID, pfName); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "LoadAllByPlatformName> Unable to load workflows")
	}

	for _, w := range dbRes {
		if err := w.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "LoadAllByPlatformName> Unable to execute post get")
		}
		res = append(res, sdk.Workflow(w))
	}
	return res, nil
}
