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
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// GetAllByIDs returns all workflows by ids.
func GetAllByIDs(db gorp.SqlExecutor, ids []int64) ([]sdk.Workflow, error) {
	ws := []sdk.Workflow{}

	if _, err := db.Select(&ws,
		`SELECT id, name FROM workflow WHERE id = ANY(string_to_array($1, ',')::int[])`,
		gorpmapping.IDsToQueryString(ids),
	); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflows")
	}

	return ws, nil
}

// LoadOptions custom option for loading workflow
type LoadOptions struct {
	DeepPipeline          bool
	WithoutNode           bool
	Base64Keys            bool
	OnlyRootNode          bool
	WithFavorites         bool
	WithLabels            bool
	WithIcon              bool
	WithAsCodeUpdateEvent bool
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
		return false, sdk.WithStack(err)
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
		return nil, sdk.WrapError(err, "Unable to count var in workflow")
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
		Metadata     sql.NullString `db:"metadata"`
		PurgeTags    sql.NullString `db:"purge_tags"`
		WorkflowData sql.NullString `db:"workflow_data"`
	}{}

	if err := db.SelectOne(&res, "SELECT metadata, purge_tags, workflow_data FROM workflow WHERE id = $1", w.ID); err != nil {
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

	data := &sdk.WorkflowData{}
	if err := gorpmapping.JSONNullString(res.WorkflowData, data); err != nil {
		return sdk.WrapError(err, "Unable to unmarshall workflow data")
	}
	if data.Node.ID != 0 {
		w.WorkflowData = data
	}

	return nil
}

// PreUpdate is a db hook
func (w *Workflow) PreUpdate(db gorp.SqlExecutor) error {
	if w.FromRepository != "" && strings.HasPrefix(w.FromRepository, "http") {
		fromRepoURL, err := url.Parse(w.FromRepository)
		if err != nil {
			return sdk.WrapError(err, "Cannot parse url %s", w.FromRepository)
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

	data, errD := gorpmapping.JSONToNullString(w.WorkflowData)
	if errD != nil {
		return sdk.WrapError(errD, "Workflow.PostUpdate> Unable to marshall workflow data")
	}
	if _, err := db.Exec("update workflow set purge_tags = $1, workflow_data = $3 where id = $2", pt, w.ID, data); err != nil {
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
		return nil, sdk.WrapError(err, "Unable to load workflows project %s", projectKey)
	}

	for _, w := range dbRes {
		w.ProjectKey = projectKey
		if err := w.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "Unable to execute post get")
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
		return nil, sdk.WrapError(err, "Unable to load workflows with project %d", projID)
	}
	for i := range res {
		var err error
		res[i].Labels, err = Labels(db, res[i].ID)
		if err != nil {
			return res, sdk.WrapError(err, "cannot load labels for workflow %s", res[i].Name)
		}
	}

	return res, nil
}

// Load loads a workflow for a given user (ie. checking permissions)
func Load(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, name string, u *sdk.User, opts LoadOptions) (*sdk.Workflow, error) {
	ctx, end := observability.Span(ctx, "workflow.Load",
		observability.Tag(observability.TagWorkflow, name),
		observability.Tag(observability.TagProjectKey, proj.Key),
		observability.Tag("with_pipeline", opts.DeepPipeline),
		observability.Tag("only_root", opts.OnlyRootNode),
		observability.Tag("with_base64_keys", opts.Base64Keys),
		observability.Tag("without_node", opts.WithoutNode),
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
		return nil, sdk.WrapError(err, "Unable to load workflow %s in project %s", name, proj.Key)
	}
	res.ProjectKey = proj.Key

	if !opts.WithoutNode {
		if err := IsValid(ctx, store, db, res, proj, u); err != nil {
			return nil, sdk.WrapError(err, "Unable to valid workflow")
		}
	}

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
		return nil, sdk.WrapError(err, "Unable to load workflow %d", id)
	}

	if !opts.WithoutNode {
		if err := IsValid(context.TODO(), store, db, res, proj, u); err != nil {
			return nil, sdk.WrapError(err, "Unable to valid workflow")
		}
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
		return nil, sdk.WrapError(err, "Unable to load workflows for project %s and pipeline %s", projectKey, pipName)
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
		return nil, sdk.WrapError(err, "Unable to load workflows for project %s and application %s", projectKey, appName)
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
		return nil, sdk.WrapError(err, "Unable to load workflows for project %s and environment %s", projectKey, envName)
	}

	res := make([]sdk.Workflow, len(dbRes))
	for i, w := range dbRes {
		w.ProjectKey = projectKey
		res[i] = sdk.Workflow(w)
	}

	return res, nil
}

// LoadByWorkflowTemplateID load all workflows linked to a workflow template but without loading workflow details
func LoadByWorkflowTemplateID(ctx context.Context, db gorp.SqlExecutor, templateID int64, u *sdk.User) ([]sdk.Workflow, error) {
	var dbRes []Workflow
	query := `
	SELECT workflow.*
		FROM workflow
			JOIN workflow_template_instance ON workflow_template_instance.workflow_id = workflow.id
		WHERE workflow_template_instance.workflow_template_id = $1 AND workflow.to_delete = false
	`
	args := []interface{}{templateID}

	if !u.Admin {
		query = `
			SELECT workflow.*
				FROM workflow
					JOIN workflow_template_instance ON workflow_template_instance.workflow_id = workflow.id
					JOIN project ON workflow.project_id = project.id
				WHERE workflow_template_instance.workflow_template_id = $1
				AND workflow.to_delete = false
				AND project.id IN (
					SELECT project_group.project_id
						FROM project_group
					WHERE
						project_group.group_id = ANY(string_to_array($2, ',')::int[])
						OR
						$3 = ANY(string_to_array($2, ',')::int[])
				)
			`
		args = append(args, gorpmapping.IDsToQueryString(sdk.GroupsToIDs(u.Groups)), group.SharedInfraGroup.ID)
	}

	if _, err := db.Select(&dbRes, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	workflows := make([]sdk.Workflow, len(dbRes))
	for i, wf := range dbRes {
		var err error
		wf.ProjectKey, err = db.SelectStr("SELECT projectkey FROM project WHERE id = $1", wf.ProjectID)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot load project key for workflow %s and project_id %d", wf.Name, wf.ProjectID)
		}
		workflows[i] = sdk.Workflow(wf)
	}

	return workflows, nil
}

func load(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, opts LoadOptions, u *sdk.User, query string, args ...interface{}) (*sdk.Workflow, error) {
	t0 := time.Now()
	dbRes := Workflow{}

	_, next := observability.Span(ctx, "workflow.load.selectOne")
	if err := db.SelectOne(&dbRes, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "Unable to load workflow")
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
	_, next = observability.Span(ctx, "workflow.load.loadWorkflowGroups")
	gps, err := group.LoadWorkflowGroups(db, res.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to load workflow groups")
	}
	res.Groups = gps
	next()

	res.Pipelines = map[int64]sdk.Pipeline{}
	res.Applications = map[int64]sdk.Application{}
	res.Environments = map[int64]sdk.Environment{}
	res.HookModels = map[int64]sdk.WorkflowHookModel{}
	res.OutGoingHookModels = map[int64]sdk.WorkflowHookModel{}

	if !opts.WithoutNode {
		_, next = observability.Span(ctx, "workflow.load.loadNodes")
		err := loadWorkflowRoot(ctx, db, store, proj, &res, u, opts)
		next()

		if err != nil {
			return nil, sdk.WrapError(err, "Unable to load workflow root")
		}

		// Load joins
		if !opts.OnlyRootNode {
			_, next = observability.Span(ctx, "workflow.load.loadJoins")
			joins, errJ := loadJoins(ctx, db, store, proj, &res, u, opts)
			next()

			if errJ != nil {
				return nil, sdk.WrapError(errJ, "Load> Unable to load workflow joins")
			}
			res.Joins = joins
		}

	}

	if opts.WithFavorites {
		_, next = observability.Span(ctx, "workflow.load.loadFavorite")
		fav, errF := loadFavorite(db, &res, u)
		next()

		if errF != nil {
			return nil, sdk.WrapError(errF, "Load> unable to load favorite")
		}
		res.Favorite = fav
	}

	if opts.WithLabels {
		_, next = observability.Span(ctx, "workflow.load.Labels")
		labels, errL := Labels(db, res.ID)
		next()

		if errL != nil {
			return nil, sdk.WrapError(errL, "Load> unable to load labels")
		}
		res.Labels = labels
	}

	if opts.WithAsCodeUpdateEvent {
		_, next = observability.Span(ctx, "workflow.load.AddCodeUpdateEvents")
		asCodeEvents, errAS := LoadAsCodeEvent(db, res.ID)
		next()

		if errAS != nil {
			return nil, sdk.WrapError(errAS, "Load> unable to load as code update events")
		}
		res.AsCodeEvent = asCodeEvents
	}

	_, next = observability.Span(ctx, "workflow.load.loadNotifications")
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
		_, next = observability.Span(ctx, "workflow.load.Sort")
		Sort(w)
		next()
	}
	return w, nil
}

func loadWorkflowRoot(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, u *sdk.User, opts LoadOptions) error {
	var err error
	w.Root, err = loadNode(ctx, db, store, proj, w, w.RootID, u, opts)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrWorkflowNodeNotFound) {
			log.Debug("Load> Unable to load root %d for workflow %d", w.RootID, w.ID)
			return nil
		}
		return sdk.WrapError(err, "Unable to load workflow root %d", w.RootID)
	}
	return nil
}

func loadFavorite(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) (bool, error) {
	count, err := db.SelectInt("SELECT COUNT(1) FROM workflow_favorite WHERE user_id = $1 AND workflow_id = $2", u.ID, w.ID)
	if err != nil {
		return false, sdk.WithStack(err)
	}
	return count > 0, nil
}

// Insert inserts a new workflow
func Insert(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, p *sdk.Project, u *sdk.User) error {
	if err := IsValid(context.TODO(), store, db, w, p, u); err != nil {
		return sdk.WrapError(err, "Unable to validate workflow")
	}

	if w.HistoryLength == 0 {
		w.HistoryLength = sdk.DefaultHistoryLength
	}

	w.LastModified = time.Now()
	if err := db.QueryRow("INSERT INTO workflow (name, description, icon, project_id, history_length, from_repository) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id", w.Name, w.Description, w.Icon, w.ProjectID, w.HistoryLength, w.FromRepository).Scan(&w.ID); err != nil {
		return sdk.WrapError(err, "Unable to insert workflow %s/%s", w.ProjectKey, w.Name)
	}

	dbw := Workflow(*w)
	if err := dbw.PostInsert(db); err != nil {
		return sdk.WrapError(err, "Cannot post insert hook")
	}

	if len(w.Groups) > 0 {
		for i := range w.Groups {
			if w.Groups[i].Group.ID != 0 {
				continue
			}
			g, err := group.LoadGroup(db, w.Groups[i].Group.Name)
			if err != nil {
				return sdk.WrapError(err, "Unable to load group %s", w.Groups[i].Group.Name)
			}
			w.Groups[i].Group = *g
		}
		if err := group.UpsertAllWorkflowGroups(db, w, w.Groups); err != nil {
			return sdk.WrapError(err, "Unable to update workflow")
		}
	} else {
		for _, gp := range p.ProjectGroups {
			if err := group.AddWorkflowGroup(db, w, gp); err != nil {
				return sdk.WrapError(err, "Cannot add group %s", gp.Group.Name)
			}
		}
	}

	if w.Root == nil {
		return sdk.WrapError(sdk.ErrWorkflowInvalidRoot, "Root node is not here")
	}

	if errIN := insertNode(db, store, w, w.Root, u, false); errIN != nil {
		return sdk.WrapError(errIN, "Unable to insert workflow root node")
	}
	w.RootID = w.Root.ID

	if w.Root.IsLinkedToRepo() {
		if w.Metadata == nil {
			w.Metadata = sdk.Metadata{}
		}
		if w.Metadata["default_tags"] == "" {
			w.Metadata["default_tags"] = "git.branch,git.author"
		} else {
			if !strings.Contains(w.Metadata["default_tags"], "git.branch") {
				w.Metadata["default_tags"] = "git.branch," + w.Metadata["default_tags"]
			}
			if !strings.Contains(w.Metadata["default_tags"], "git.author") {
				w.Metadata["default_tags"] = "git.author," + w.Metadata["default_tags"]
			}
		}

		if err := UpdateMetadata(db, w.ID, w.Metadata); err != nil {
			return sdk.WrapError(err, "Unable to insert workflow metadata (%#v, %d)", w.Root, w.ID)
		}
	}

	if _, err := db.Exec("UPDATE workflow SET root_node_id = $2 WHERE id = $1", w.ID, w.Root.ID); err != nil {
		return sdk.WrapError(err, "Unable to insert workflow (%#v, %d)", w.Root, w.ID)
	}

	for i := range w.Joins {
		j := &w.Joins[i]
		if err := insertJoin(db, store, w, j, u); err != nil {
			return sdk.WrapError(err, "Unable to insert update workflow(%d) join (%#v)", w.ID, j)
		}
	}

	// Insert notifications
	for i := range w.Notifications {
		n := &w.Notifications[i]
		if err := insertNotification(db, store, w, n, u); err != nil {
			return sdk.WrapError(err, "Unable to insert update workflow(%d) notification (%#v)", w.ID, n)
		}
	}

	// TODO Delete in last migration step
	hooks := w.GetHooks()
	w.WorkflowData.Node.Hooks = make([]sdk.NodeHook, 0, len(hooks))
	for _, h := range hooks {
		w.WorkflowData.Node.Hooks = append(w.WorkflowData.Node.Hooks, sdk.NodeHook{
			Ref:           h.Ref,
			HookModelID:   h.WorkflowHookModelID,
			Config:        h.Config,
			UUID:          h.UUID,
			HookModelName: h.WorkflowHookModel.Name,
		})
	}

	if err := InsertWorkflowData(db, w); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert Workflow Data")
	}

	dbWorkflow := Workflow(*w)
	if err := dbWorkflow.PostUpdate(db); err != nil {
		return sdk.WrapError(err, "Insert> Unable to create workflow data")
	}

	event.PublishWorkflowAdd(p.Key, *w, u)

	return nil
}

func RenameNode(db gorp.SqlExecutor, w *sdk.Workflow) error {
	nodes := w.WorkflowData.Array()
	var maxJoinNumber int
	maxNumberByPipeline := map[int64]int{}
	maxNumberByHookModel := map[int64]int{}
	var maxForkNumber int

	nodesToNamed := []*sdk.Node{}
	// Search max numbers by nodes type
	for i := range nodes {
		if nodes[i].Name == "" {
			nodesToNamed = append(nodesToNamed, nodes[i])
		}

		switch nodes[i].Type {
		case sdk.NodeTypePipeline:
			if w.Pipelines == nil {
				w.Pipelines = make(map[int64]sdk.Pipeline)
			}
			_, has := w.Pipelines[nodes[i].Context.PipelineID]
			if !has {
				p, errPip := pipeline.LoadPipelineByID(context.TODO(), db, nodes[i].Context.PipelineID, true)
				if errPip != nil {
					return sdk.WrapError(errPip, "renameNode> Unable to load pipeline %d", nodes[i].Context.PipelineID)
				}
				w.Pipelines[nodes[i].Context.PipelineID] = *p
			}
		case sdk.NodeTypeOutGoingHook:
			if w.OutGoingHookModels == nil {
				w.OutGoingHookModels = make(map[int64]sdk.WorkflowHookModel)
			}
			_, has := w.OutGoingHookModels[nodes[i].OutGoingHookContext.HookModelID]
			if !has {
				m, errM := LoadOutgoingHookModelByID(db, nodes[i].OutGoingHookContext.HookModelID)
				if errM != nil {
					return sdk.WrapError(errM, "renameNode> Unable to load outgoing hook model %d", nodes[i].OutGoingHookContext.HookModelID)
				}
				w.OutGoingHookModels[nodes[i].OutGoingHookContext.HookModelID] = *m
			}
		}

		switch nodes[i].Type {
		case sdk.NodeTypePipeline:
			pip := w.Pipelines[nodes[i].Context.PipelineID]
			// Check if node is named pipName_12
			if nodes[i].Name == pip.Name || strings.HasPrefix(nodes[i].Name, pip.Name+"_") {
				var pipNumber int
				if nodes[i].Name == pip.Name {
					pipNumber = 1
				} else {
					// Retrieve Number
					current, errI := strconv.Atoi(strings.Replace(nodes[i].Name, pip.Name+"_", "", 1))
					if errI == nil {
						pipNumber = current
					}
				}
				currentMax, ok := maxNumberByPipeline[pip.ID]
				if !ok || currentMax < pipNumber {
					maxNumberByPipeline[pip.ID] = pipNumber
				}
			}
		case sdk.NodeTypeJoin:
			if nodes[i].Name == sdk.NodeTypeJoin || strings.HasPrefix(nodes[i].Name, sdk.NodeTypeJoin+"_") {
				var joinNumber int
				if nodes[i].Name == sdk.NodeTypeJoin {
					joinNumber = 1
				} else {
					// Retrieve Number
					current, errI := strconv.Atoi(strings.Replace(nodes[i].Name, sdk.NodeTypeJoin+"_", "", 1))
					if errI == nil {
						joinNumber = current
					}
				}
				if maxJoinNumber < joinNumber {
					maxJoinNumber = joinNumber
				}
			}
		case sdk.NodeTypeFork:
			if nodes[i].Name == sdk.NodeTypeFork || strings.HasPrefix(nodes[i].Name, sdk.NodeTypeFork+"_") {
				var forkNumber int
				if nodes[i].Name == sdk.NodeTypeFork {
					forkNumber = 1
				} else {
					// Retrieve Number
					current, errI := strconv.Atoi(strings.Replace(nodes[i].Name, sdk.NodeTypeFork+"_", "", 1))
					if errI == nil {
						forkNumber = current
					}
				}
				if maxForkNumber < forkNumber {
					maxForkNumber = forkNumber
				}
			}
		case sdk.NodeTypeOutGoingHook:
			model := w.OutGoingHookModels[nodes[i].OutGoingHookContext.HookModelID]
			// Check if node is named pipName_12
			if nodes[i].Name == model.Name || strings.HasPrefix(nodes[i].Name, model.Name+"_") {
				var hookNumber int
				if nodes[i].Name == model.Name {
					hookNumber = 1
				} else {
					// Retrieve Number
					current, errI := strconv.Atoi(strings.Replace(nodes[i].Name, model.Name+"_", "", 1))
					if errI == nil {
						hookNumber = current
					}
				}
				currentMax, ok := maxNumberByHookModel[model.ID]
				if !ok || currentMax < hookNumber {
					maxNumberByHookModel[model.ID] = hookNumber
				}
			}
		}

		if nodes[i].Ref == "" {
			nodes[i].Ref = nodes[i].Name
		}
	}

	// Name node
	for i := range nodesToNamed {
		switch nodesToNamed[i].Type {
		case sdk.NodeTypePipeline:
			pipID := nodesToNamed[i].Context.PipelineID
			nextNumber := maxNumberByPipeline[pipID] + 1
			if nextNumber > 1 {
				nodesToNamed[i].Name = fmt.Sprintf("%s_%d", w.Pipelines[pipID].Name, nextNumber)
			} else {
				nodesToNamed[i].Name = w.Pipelines[pipID].Name
			}
			maxNumberByPipeline[pipID] = nextNumber
		case sdk.NodeTypeJoin:
			nextNumber := maxJoinNumber + 1
			if nextNumber > 1 {
				nodesToNamed[i].Name = fmt.Sprintf("%s_%d", sdk.NodeTypeJoin, nextNumber)
			} else {
				nodesToNamed[i].Name = sdk.NodeTypeJoin
			}
			maxJoinNumber++
		case sdk.NodeTypeFork:
			nextNumber := maxForkNumber + 1
			if nextNumber > 1 {
				nodesToNamed[i].Name = fmt.Sprintf("%s_%d", sdk.NodeTypeFork, nextNumber)
			} else {
				nodesToNamed[i].Name = sdk.NodeTypeFork
			}
			maxForkNumber++
		case sdk.NodeTypeOutGoingHook:
			hookModelID := nodesToNamed[i].OutGoingHookContext.HookModelID
			nextNumber := maxNumberByHookModel[hookModelID] + 1
			if nextNumber > 1 {
				nodesToNamed[i].Name = fmt.Sprintf("%s_%d", w.OutGoingHookModels[hookModelID].Name, nextNumber)
			} else {
				nodesToNamed[i].Name = w.OutGoingHookModels[hookModelID].Name
			}
			maxNumberByHookModel[hookModelID] = nextNumber
		}
		if nodesToNamed[i].Ref == "" {
			nodesToNamed[i].Ref = nodesToNamed[i].Name
		}
	}

	return nil
}

// Update updates a workflow
func Update(ctx context.Context, db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, oldWorkflow *sdk.Workflow, p *sdk.Project, u *sdk.User) error {
	if err := IsValid(ctx, store, db, w, p, u); err != nil {
		return err
	}

	// Delete all OLD JOIN
	for _, j := range oldWorkflow.Joins {
		if err := deleteJoin(db, j); err != nil {
			return sdk.WrapError(err, "unable to delete all joins on workflow(%d)", w.ID)
		}
	}

	if err := deleteNotifications(db, oldWorkflow.ID); err != nil {
		return sdk.WrapError(err, "unable to delete all notifications on workflow(%d)", w.ID)
	}

	// Delete old Root Node
	if oldWorkflow.Root != nil {
		if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
			return sdk.WrapError(err, "Unable to detach workflow root")
		}

		if err := deleteNode(db, oldWorkflow, oldWorkflow.Root); err != nil {
			return sdk.WrapError(err, "unable to delete root node on workflow(%d)", w.ID)
		}
	}

	// Delete workflow data
	if err := DeleteWorkflowData(db, *oldWorkflow); err != nil {
		return sdk.WrapError(err, "Update> unable to delete workflow data(%d)", w.ID)
	}

	// Delete all node ID
	w.ResetIDs()

	if err := insertNode(db, store, w, w.Root, u, false); err != nil {
		return sdk.WrapError(sdk.ErrWorkflowNodeRootUpdate, "unable to update root node on workflow(%d) : %v", w.ID, err)
	}
	w.RootID = w.Root.ID

	// Insert new JOIN
	for i := range w.Joins {
		j := &w.Joins[i]
		if err := insertJoin(db, store, w, j, u); err != nil {
			return sdk.WrapError(err, "Unable to update workflow(%d) join (%#v)", w.ID, j)
		}
	}

	// Insert notifications
	for i := range w.Notifications {
		n := &w.Notifications[i]
		if err := insertNotification(db, store, w, n, u); err != nil {
			return sdk.WrapError(err, "Unable to update workflow(%d) notification (%#v)", w.ID, n)
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

	// TODO: DELETE in step 3: Synchronize HOOK datas
	hooks := w.GetHooks()
	w.WorkflowData.Node.Hooks = make([]sdk.NodeHook, 0, len(hooks))
	for _, h := range hooks {
		w.WorkflowData.Node.Hooks = append(w.WorkflowData.Node.Hooks, sdk.NodeHook{
			Ref:           h.Ref,
			HookModelID:   h.WorkflowHookModelID,
			Config:        h.Config,
			UUID:          h.UUID,
			HookModelName: h.WorkflowHookModel.Name,
		})
	}

	if err := InsertWorkflowData(db, w); err != nil {
		return sdk.WrapError(err, "Update> Unable to insert workflow data")
	}

	w.LastModified = time.Now()
	dbw := Workflow(*w)
	if _, err := db.Update(&dbw); err != nil {
		return sdk.WrapError(err, "Unable to update workflow")
	}
	*w = sdk.Workflow(dbw)

	event.PublishWorkflowUpdate(p.Key, *w, *oldWorkflow, u)
	return nil
}

// MarkAsDelete marks a workflow to be deleted
func MarkAsDelete(db gorp.SqlExecutor, w *sdk.Workflow) error {
	if _, err := db.Exec("update workflow set to_delete = true where id = $1", w.ID); err != nil {
		return sdk.WrapError(err, "Unable to mark as delete workflow id %d", w.ID)
	}
	return nil
}

// Delete workflow
func Delete(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow) error {
	log.Debug("Delete> deleting workflow %d", w.ID)

	//Detach root from workflow
	if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
		return sdk.WrapError(err, "Unable to detach workflow root")
	}

	hooks := w.GetHooks()
	// Delete all hooks
	if err := DeleteHookConfiguration(ctx, db, store, p, hooks); err != nil {
		return sdk.WrapError(err, "Unable to delete hooks from workflow")
	}

	// Delete all JOINs
	for _, j := range w.Joins {
		if err := deleteJoin(db, j); err != nil {
			return sdk.WrapError(err, "unable to delete all join on workflow(%d)", w.ID)
		}
	}

	//Delete root
	if err := deleteNode(db, w, w.Root); err != nil {
		return sdk.WrapError(err, "Unable to delete workflow root")
	}

	if err := DeleteWorkflowData(db, *w); err != nil {
		return sdk.WrapError(err, "Delete> Unable to delete workflow data")
	}

	//Delete workflow
	dbw := Workflow(*w)
	if _, err := db.Delete(&dbw); err != nil {
		return sdk.WrapError(err, "Unable to delete workflow")
	}

	return nil
}

// IsValid cheks workflow validity
func IsValid(ctx context.Context, store cache.Store, db gorp.SqlExecutor, w *sdk.Workflow, proj *sdk.Project, u *sdk.User) error {
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

	if w.Pipelines == nil {
		w.Pipelines = make(map[int64]sdk.Pipeline)
	}
	if w.Applications == nil {
		w.Applications = make(map[int64]sdk.Application)
	}
	if w.Environments == nil {
		w.Environments = make(map[int64]sdk.Environment)
	}
	if w.ProjectIntegrations == nil {
		w.ProjectIntegrations = make(map[int64]sdk.ProjectIntegration)
	}
	if w.HookModels == nil {
		w.HookModels = make(map[int64]sdk.WorkflowHookModel)
	}
	if w.OutGoingHookModels == nil {
		w.OutGoingHookModels = make(map[int64]sdk.WorkflowHookModel)
	}

	if w.WorkflowData == nil {
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

		//Checks integrations are in the current project
		pfs := w.InvolvedIntegrations()
		for _, id := range pfs {
			var found bool
			for _, p := range proj.Integrations {
				if id == p.ID {
					found = true
					break
				}
			}
			if !found {
				return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Unknown integrations %d", id))
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

	// Fill empty node type
	w.AssignEmptyType()
	if err := w.ValidateType(); err != nil {
		return err
	}

	nodesArray := w.WorkflowData.Array()
	for i := range nodesArray {
		n := nodesArray[i]
		if n.Context == nil {
			continue
		}

		if err := checkPipeline(ctx, db, proj, w, n); err != nil {
			return err
		}
		if err := checkApplication(store, db, proj, w, n); err != nil {
			return err
		}
		if err := checkEnvironment(db, proj, w, n); err != nil {
			return err
		}
		if err := checkProjectIntegration(proj, w, n); err != nil {
			return err
		}
		if err := checkHooks(db, w, n); err != nil {
			return err
		}
		if err := checkOutGoingHook(db, w, n); err != nil {
			return err
		}

		if n.Context.ApplicationID != 0 && n.Context.ProjectIntegrationID != 0 {
			if err := n.CheckApplicationDeploymentStrategies(proj, w); err != nil {
				return sdk.NewError(sdk.ErrWorkflowInvalid, err)
			}
		}
	}

	return nil
}

func checkOutGoingHook(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node) error {
	if n.OutGoingHookContext == nil {
		return nil
	}

	if n.OutGoingHookContext.HookModelID != 0 {
		hm, ok := w.OutGoingHookModels[n.OutGoingHookContext.HookModelID]
		if !ok {
			hmDB, err := LoadOutgoingHookModelByID(db, n.OutGoingHookContext.HookModelID)
			if err != nil {
				return err
			}
			hm = *hmDB
			w.OutGoingHookModels[n.OutGoingHookContext.HookModelID] = hm
		}
		n.OutGoingHookContext.HookModelName = hm.Name
		return nil
	}

	if n.OutGoingHookContext.HookModelName != "" {
		hmDB, err := LoadOutgoingHookModelByName(db, n.OutGoingHookContext.HookModelName)
		if err != nil {
			return err
		}
		w.OutGoingHookModels[hmDB.ID] = *hmDB
		n.OutGoingHookContext.HookModelID = hmDB.ID
		return nil
	}
	return nil
}

func checkHooks(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node) error {
	for i := range n.Hooks {
		h := &n.Hooks[i]
		if h.HookModelID != 0 {
			hm, ok := w.HookModels[h.HookModelID]
			if !ok {
				hmDB, err := LoadHookModelByID(db, h.HookModelID)
				if err != nil {
					return err
				}
				hm = *hmDB
				w.HookModels[h.HookModelID] = hm
			}
			h.HookModelName = hm.Name
		} else if h.HookModelName != "" {
			hm, err := LoadHookModelByName(db, h.HookModelName)
			if err != nil {
				return err
			}
			w.HookModels[hm.ID] = *hm
			h.HookModelID = hm.ID
		}
	}
	return nil
}

// CheckProjectIntegration checks CheckProjectIntegration data
func checkProjectIntegration(proj *sdk.Project, w *sdk.Workflow, n *sdk.Node) error {
	if n.Context.ProjectIntegrationID != 0 {
		pp, ok := w.ProjectIntegrations[n.Context.ProjectIntegrationID]
		if !ok {
			for _, pl := range proj.Integrations {
				if pl.ID == n.Context.ProjectIntegrationID {
					pp = pl
					break
				}
			}
			if pp.ID == 0 {
				return sdk.WrapError(sdk.ErrNotFound, "Integration %d not found", n.Context.ProjectIntegrationID)
			}
			w.ProjectIntegrations[n.Context.ProjectIntegrationID] = pp
		}
		n.Context.ProjectIntegrationName = pp.Name
		return nil
	}
	if n.Context.ProjectIntegrationName != "" {
		var ppProj sdk.ProjectIntegration
		for _, pl := range proj.Integrations {
			if pl.Name == n.Context.ProjectIntegrationName {
				ppProj = pl
				break
			}
		}
		if ppProj.ID == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "Integration %s not found", n.Context.ProjectIntegrationName)
		}
		w.ProjectIntegrations[ppProj.ID] = ppProj
		n.Context.ProjectIntegrationID = ppProj.ID
	}
	return nil
}

// CheckEnvironment checks environment data
func checkEnvironment(db gorp.SqlExecutor, proj *sdk.Project, w *sdk.Workflow, n *sdk.Node) error {
	if n.Context.EnvironmentID != 0 {
		env, ok := w.Environments[n.Context.EnvironmentID]
		if !ok {
			found := false
			for _, e := range proj.Environments {
				if e.ID == n.Context.EnvironmentID {
					found = true
				}
			}
			if !found {
				return sdk.WithStack(sdk.ErrNoEnvironment)
			}

			// Load environment from db to get stage/jobs
			envDB, err := environment.LoadEnvironmentByID(db, n.Context.EnvironmentID)
			if err != nil {
				return sdk.WrapError(err, "unable to load environment %d", n.Context.EnvironmentID)
			}
			env = *envDB
			w.Environments[n.Context.EnvironmentID] = env
		}
		n.Context.EnvironmentName = env.Name
		return nil
	}
	if n.Context.EnvironmentName != "" {
		envDB, err := environment.LoadEnvironmentByName(db, proj.Key, n.Context.EnvironmentName)
		if err != nil {
			return sdk.WrapError(err, "unable to load environment %s", n.Context.EnvironmentName)
		}
		w.Environments[envDB.ID] = *envDB
		n.Context.EnvironmentID = envDB.ID
	}
	return nil
}

// CheckApplication checks application data
func checkApplication(store cache.Store, db gorp.SqlExecutor, proj *sdk.Project, w *sdk.Workflow, n *sdk.Node) error {
	if n.Context.ApplicationID != 0 {
		app, ok := w.Applications[n.Context.ApplicationID]
		if !ok {
			found := false
			for _, a := range proj.Applications {
				if a.ID == n.Context.ApplicationID {
					app = a
					found = true
				}
			}
			if !found {
				return sdk.WithStack(sdk.ErrApplicationNotFound)
			}
			w.Applications[n.Context.ApplicationID] = app
		}
		n.Context.ApplicationName = app.Name
		return nil
	}
	if n.Context.ApplicationName != "" {
		appDB, err := application.LoadByName(db, store, proj.Key, n.Context.ApplicationName, application.LoadOptions.WithDeploymentStrategies, application.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "unable to load application %s", n.Context.ApplicationName)
		}
		w.Applications[appDB.ID] = *appDB
		n.Context.ApplicationID = appDB.ID
	}
	return nil
}

// CheckPipeline checks pipeline data
func checkPipeline(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project, w *sdk.Workflow, n *sdk.Node) error {
	if n.Context.PipelineID != 0 {
		pip, ok := w.Pipelines[n.Context.PipelineID]
		if !ok {
			found := false
			for _, p := range proj.Pipelines {
				if p.ID == n.Context.PipelineID {
					found = true
				}
			}
			if !found {
				return sdk.NewErrorFrom(sdk.ErrPipelineNotFound, "Can not found a pipeline with id %d", n.Context.PipelineID)
			}

			// Load pipeline from db to get stage/jobs
			pipDB, err := pipeline.LoadPipelineByID(ctx, db, n.Context.PipelineID, true)
			if err != nil {
				return sdk.WrapError(err, "unable to load pipeline %d", n.Context.PipelineID)
			}
			pip = *pipDB
			w.Pipelines[n.Context.PipelineID] = pip
		}
		n.Context.PipelineName = pip.Name
		return nil
	}
	if n.Context.PipelineName != "" {
		pipDB, err := pipeline.LoadPipeline(db, proj.Key, n.Context.PipelineName, true)
		if err != nil {
			return sdk.WrapError(err, "unable to load pipeline %s", n.Context.PipelineName)
		}
		w.Pipelines[pipDB.ID] = *pipDB
		n.Context.PipelineID = pipDB.ID
	}
	return nil
}

// Push push a workflow from cds files
func Push(ctx context.Context, db *gorp.DbMap, store cache.Store, proj *sdk.Project, tr *tar.Reader, opts *PushOption, u *sdk.User, decryptFunc keys.DecryptFunc) ([]sdk.Message, *sdk.Workflow, error) {
	ctx, end := observability.Span(ctx, "workflow.Push")
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
			return nil, nil, sdk.WithStack(err)
		}

		log.Debug("Push> Reading %s", hdr.Name)

		buff := new(bytes.Buffer)
		if _, err := io.Copy(buff, tr); err != nil {
			err = sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Unable to read tar file"))
			return nil, nil, sdk.WithStack(err)
		}

		var workflowFileName string
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
			// if a workflow was already found, it's a mistake
			if workflowFileName != "" {
				log.Error("two workflows files found: %s and %s", workflowFileName, hdr.Name)
				mError.Append(fmt.Errorf("two workflows files found: %s and %s", workflowFileName, hdr.Name))
				break
			}
			if err := yaml.Unmarshal(b, &wrkflw); err != nil {
				log.Error("Push> Unable to unmarshal workflow %s: %v", hdr.Name, err)
				mError.Append(fmt.Errorf("Unable to unmarshal workflow %s: %v", hdr.Name, err))
				continue
			}
			workflowFileName = hdr.Name
		}
	}

	// We only use the multiError during unmarshalling steps.
	// When a DB transaction has been started, just return at the first error
	// because transaction may have to be aborted
	if !mError.IsEmpty() {
		return nil, nil, sdk.NewError(sdk.ErrWorkflowInvalid, mError)
	}

	// load the workflow from database if exists
	workflowExists, err := Exists(db, proj.Key, wrkflw.Name)
	if err != nil {
		return nil, nil, sdk.WrapError(err, "Cannot check if workflow exists")
	}
	var wf *sdk.Workflow
	if workflowExists {
		wf, err = Load(ctx, db, store, proj, wrkflw.Name, u, LoadOptions{WithIcon: true})
		if err != nil {
			return nil, nil, sdk.WrapError(err, "Unable to load existing workflow")
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, nil, sdk.WrapError(err, "Unable to start tx")
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

	// In workflow as code context, if we only have the repowebhook, we skip it
	//  because it will be automatically recreated later with the proper configuration
	if opts != nil && opts.FromRepository != "" {
		if len(wrkflw.Workflow) == 0 {
			if len(wrkflw.PipelineHooks) == 1 && wrkflw.PipelineHooks[0].Model == sdk.RepositoryWebHookModelName {
				wrkflw.PipelineHooks = nil
			}
		} else {
			for node, hooks := range wrkflw.Hooks {
				if len(hooks) == 1 && hooks[0].Model == sdk.RepositoryWebHookModelName {
					wrkflw.Hooks[node] = nil
				}
			}
		}
	}

	wf, msgList, err := ParseAndImport(ctx, tx, store, proj, wf, &wrkflw, u, ImportOptions{DryRun: dryRun, Force: true})
	if err != nil {
		log.Error("Push> Unable to import workflow: %v", err)
		return nil, nil, sdk.WrapError(err, "unable to import workflow %s", wrkflw.Name)
	}

	// TODO workflow as code, manage derivation workflow
	if opts != nil {
		wf.FromRepository = opts.FromRepository
		if !opts.IsDefaultBranch {
			wf.DerivationBranch = opts.Branch
		}
		// do not override application data if no opts were given
		if wf.Root.Context.Application != nil && opts.VCSServer != "" {
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
				if wf.Root.Context.DefaultPayload, err = DefaultPayload(ctx, tx, store, proj, wf); err != nil {
					return nil, nil, sdk.WrapError(err, "Unable to get default payload")
				}
				wf.WorkflowData.Node.Context.DefaultPayload = wf.Root.Context.DefaultPayload
			}

			if wf.Root.Context.Application != nil {
				if err := application.Update(tx, store, wf.Root.Context.Application); err != nil {
					return nil, nil, sdk.WrapError(err, "Unable to update application vcs datas")
				}
			}
		}

		if err := Update(ctx, tx, store, wf, wf, proj, u); err != nil {
			return nil, nil, sdk.WrapError(err, "Unable to update workflow")
		}

		if !opts.DryRun {
			if errHr := HookRegistration(ctx, tx, store, nil, *wf, proj); errHr != nil {
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
			return nil, nil, sdk.WrapError(err, "Cannot commit transaction")
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
	return sdk.WithStack(err)
}

// IsDeploymentIntegrationUsed checks if a deployment integration is used on any workflow
func IsDeploymentIntegrationUsed(db gorp.SqlExecutor, projectID int64, appID int64, pfName string) (bool, error) {
	query := `
	SELECT count(1)
	FROM workflow_node_context
	JOIN project_integration ON project_integration.id = workflow_node_context.project_integration_id
	WHERE workflow_node_context.application_id = $2
	AND project_integration.project_id = $1
	AND project_integration.name = $3
	`

	nb, err := db.SelectInt(query, projectID, appID, pfName)
	if err != nil {
		return false, sdk.WrapError(err, "IsDeploymentIntegrationUsed")
	}

	return nb > 0, nil
}
