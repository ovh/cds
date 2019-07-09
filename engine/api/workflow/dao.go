package workflow

import (
	"archive/tar"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

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
	Minimal               bool
	DeepPipeline          bool
	Base64Keys            bool
	WithFavorites         bool
	WithLabels            bool
	WithIcon              bool
	WithAsCodeUpdateEvent bool
}

// UpdateOptions is option to parse a workflow
type UpdateOptions struct {
	DisableHookManagement bool
	OldWorkflow           *sdk.Workflow
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
		SELECT DISTINCT workflow.name as workflow_name, w_node.name as node_name
		FROM workflow
		JOIN project ON project.id = workflow.project_id
		JOIN w_node ON w_node.workflow_id = workflow.id
		JOIN w_node_context ON w_node_context.node_id = w_node.id
		WHERE project.projectkey = $1
		AND (
			w_node_context.default_pipeline_parameters::TEXT LIKE $2
			OR
			w_node_context.default_payload::TEXT LIKE $2
		);
	`
	var datas []CountVarInWorkflowData
	if _, err := db.Select(&datas, query, projectKey, fmt.Sprintf("%%%s%%", varName)); err != nil {
		return nil, sdk.WrapError(err, "Unable to count var in workflow")
	}
	return datas, nil
}

// UpdateIcon update the icon of a workflow
func UpdateIcon(db gorp.SqlExecutor, workflowID int64, icon string) error {
	if _, err := db.Exec("update workflow set icon = $1 where id = $2", icon, workflowID); err != nil {
		return sdk.WrapError(err, "cannot update workflow icon for workflow id %d", workflowID)
	}

	return nil
}

// UpdateMetadata update the metadata of a workflow
func UpdateMetadata(db gorp.SqlExecutor, workflowID int64, metadata sdk.Metadata) error {
	b, err := json.Marshal(metadata)
	if err != nil {
		return sdk.WithStack(err)
	}
	if _, err := db.Exec("update workflow set metadata = $1 where id = $2", b, workflowID); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// updateFromRepository update the from_repository of a workflow
func updateFromRepository(db gorp.SqlExecutor, workflowID int64, fromRepository string) error {
	if _, err := db.Exec("UPDATE workflow SET from_repository = $1, last_modified = current_timestamp WHERE id = $2", fromRepository, workflowID); err != nil {
		return sdk.WithStack(err)
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

	nodes := w.WorkflowData.Array()
	for i := range nodes {
		var err error
		nodes[i].Groups, err = group.LoadGroupsByNode(db, nodes[i].ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load node groups")
		}
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
func LoadAllNames(db gorp.SqlExecutor, projID int64, u *sdk.User) (sdk.IDNames, error) {
	query := `
		SELECT workflow.name, workflow.id, workflow.description, workflow.icon
		FROM workflow
		WHERE workflow.project_id = $1
		AND workflow.to_delete = false
		ORDER BY workflow.name ASC`

	var res sdk.IDNames
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
		observability.Tag("minimal", opts.Minimal),
		observability.Tag("with_pipeline", opts.DeepPipeline),
		observability.Tag("with_base64_keys", opts.Base64Keys),
	)
	defer end()

	var icon string
	if !opts.Minimal {
		if opts.WithIcon {
			icon = "workflow.icon,"
		}
	} else {
		// if minimal, reset load options to load only from table workflow
		opts = LoadOptions{Minimal: true}
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

	if !opts.Minimal {
		if err := IsValid(ctx, store, db, res, proj, u, opts); err != nil {
			return nil, sdk.WrapError(err, "Unable to valid workflow")
		}
	}

	return res, nil
}

// LoadWorkflowIDsWithNotifications loads workflow id of workflow with notifications
func LoadWorkflowIDsWithNotifications(db gorp.SqlExecutor) ([]int64, error) {
	query := `
    SELECT id from WORKFLOW
    WHERE id IN (SELECT workflow_id FROM workflow_notification)`
	var ids []int64
	_, err := db.Select(&ids, query)
	return ids, sdk.WrapError(err, "unable to select workflow with notification")
}

func LoadAndLock(db gorp.SqlExecutor, id int64, store cache.Store, proj *sdk.Project, opts LoadOptions, u *sdk.User) (*sdk.Workflow, error) {
	query := `
		select *
		from workflow
		where id = $1 FOR UPDATE SKIP LOCKED`
	res, err := load(context.TODO(), db, store, proj, opts, u, query, id)
	if err != nil {
		if err == sdk.ErrWorkflowNotFound {
			err = sdk.ErrLocked
		}
		return nil, sdk.WithStack(err)
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

	if err := IsValid(context.TODO(), store, db, res, proj, u, opts); err != nil {
		return nil, sdk.WrapError(err, "Unable to valid workflow")
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
		join w_node on w_node.workflow_id = workflow.id
    join w_node_context on w_node_context.node_id = w_node.id
		join pipeline on pipeline.id = w_node_context.pipeline_id
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
		join w_node on w_node.workflow_id = workflow.id
		join w_node_context on w_node_context.node_id = w_node.id
		join application on w_node_context.application_id = application.id
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
		join w_node on w_node.workflow_id = workflow.id
		join w_node_context on w_node_context.node_id = w_node.id
		join environment on w_node_context.environment_id = environment.id
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
	err := db.SelectOne(&dbRes, query, args...)
	next()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "Unable to load workflow")
	}

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
	next()
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to load workflow groups")
	}
	res.Groups = gps

	res.Pipelines = map[int64]sdk.Pipeline{}
	res.Applications = map[int64]sdk.Application{}
	res.Environments = map[int64]sdk.Environment{}
	res.HookModels = map[int64]sdk.WorkflowHookModel{}
	res.OutGoingHookModels = map[int64]sdk.WorkflowHookModel{}

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
	return w, nil
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
	if err := IsValid(context.TODO(), store, db, w, p, u, LoadOptions{}); err != nil {
		return sdk.WrapError(err, "Unable to validate workflow")
	}

	if w.WorkflowData.Node.Context != nil && w.WorkflowData.Node.Context.ApplicationID != 0 {
		var err error
		if w.WorkflowData.Node.Context.DefaultPayload, err = DefaultPayload(context.TODO(), db, store, p, w); err != nil {
			log.Warning("postWorkflowHandler> Cannot set default payload : %v", err)
		}
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

	if w.WorkflowData.Node.IsLinkedToRepo(w) {
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
			return err
		}
	}

	// Manage new hooks
	if len(w.WorkflowData.Node.Hooks) > 0 {
		if err := hookRegistration(context.TODO(), db, store, p, w, nil); err != nil {
			return err
		}
	}

	if err := InsertWorkflowData(db, w); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert Workflow Data")
	}

	// Insert notifications
	for i := range w.Notifications {
		n := &w.Notifications[i]
		if err := InsertNotification(db, w, n); err != nil {
			return sdk.WrapError(err, "Unable to insert update workflow(%d) notification (%#v)", w.ID, n)
		}
	}

	dbWorkflow := Workflow(*w)
	if err := dbWorkflow.PostUpdate(db); err != nil {
		return sdk.WrapError(err, "Insert> Unable to create workflow data")
	}

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
func Update(ctx context.Context, db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, p *sdk.Project, u *sdk.User, uptOption UpdateOptions) error {
	ctx, end := observability.Span(ctx, "workflow.Update")
	defer end()
	if err := IsValid(ctx, store, db, w, p, u, LoadOptions{}); err != nil {
		return err
	}

	if err := DeleteNotifications(db, w.ID); err != nil {
		return sdk.WrapError(err, "unable to delete all notifications on workflow(%d - %s)", w.ID, w.Name)
	}

	// Delete workflow data
	if uptOption.OldWorkflow != nil {
		if err := DeleteWorkflowData(db, *uptOption.OldWorkflow); err != nil {
			return sdk.WrapError(err, "unable to delete from old workflow data(%d - %s)", w.ID, w.Name)
		}
	} else {
		if err := DeleteWorkflowData(db, *w); err != nil {
			return sdk.WrapError(err, "unable to delete from workflow data(%d - %s)", w.ID, w.Name)
		}
	}

	// Delete all node ID
	w.ResetIDs()

	filteredPurgeTags := []string{}
	for _, t := range w.PurgeTags {
		if t != "" {
			filteredPurgeTags = append(filteredPurgeTags, t)
		}
	}
	w.PurgeTags = filteredPurgeTags

	if w.WorkflowData.Node.Context != nil && w.WorkflowData.Node.Context.ApplicationID != 0 {
		var err error
		if w.WorkflowData.Node.Context.DefaultPayload, err = DefaultPayload(ctx, db, store, p, w); err != nil {
			log.Warning("putWorkflowHandler> Cannot set default payload : %v", err)
		}
	}

	if !uptOption.DisableHookManagement {
		if err := hookRegistration(ctx, db, store, p, w, uptOption.OldWorkflow); err != nil {
			return err
		}
		if uptOption.OldWorkflow != nil {
			hookToDelete := computeHookToDelete(w, uptOption.OldWorkflow)
			if err := hookUnregistration(ctx, db, store, p, hookToDelete); err != nil {
				return err
			}
		}

	}

	if err := InsertWorkflowData(db, w); err != nil {
		return sdk.WrapError(err, "Update> Unable to insert workflow data")
	}

	// Insert notifications
	for i := range w.Notifications {
		n := &w.Notifications[i]
		if err := InsertNotification(db, w, n); err != nil {
			return sdk.WrapError(err, "Unable to update workflow(%d) notification (%#v)", w.ID, n)
		}
	}

	w.LastModified = time.Now()
	dbw := Workflow(*w)
	if _, err := db.Update(&dbw); err != nil {
		return sdk.WrapError(err, "Unable to update workflow")
	}
	*w = sdk.Workflow(dbw)

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

	// Delete all hooks
	if err := hookUnregistration(ctx, db, store, p, w.WorkflowData.GetHooks()); err != nil {
		return sdk.WrapError(err, "Unable to delete hooks from workflow")
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
func IsValid(ctx context.Context, store cache.Store, db gorp.SqlExecutor, w *sdk.Workflow, proj *sdk.Project, u *sdk.User, opts LoadOptions) error {
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

	//Check refs
	for _, j := range w.WorkflowData.Joins {
		if len(j.JoinContext) == 0 {
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

	if w.WorkflowData.Node.Context != nil && w.WorkflowData.Node.Context.DefaultPayload != nil {
		defaultPayload, err := w.WorkflowData.Node.Context.DefaultPayloadToMap()
		if err != nil {
			return sdk.WrapError(err, "cannot transform default payload to map")
		}
		for payloadKey := range defaultPayload {
			if strings.HasPrefix(payloadKey, "cds.") {
				return sdk.WrapError(sdk.ErrInvalidPayloadVariable, "cannot have key %s in default payload", payloadKey)
			}
		}
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

		if err := checkPipeline(ctx, db, proj, w, n, opts); err != nil {
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
			return sdk.ErrorWithData(sdk.ErrIntegrationtNotFound, n.Context.ProjectIntegrationName)
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

			// Load environment from db to get stage/jobs
			envDB, err := environment.LoadEnvironmentByID(db, n.Context.EnvironmentID)
			if err != nil {
				return sdk.WrapError(err, "unable to load environment %d", n.Context.EnvironmentID)
			}
			env = *envDB

			if env.ProjectID != proj.ID {
				return sdk.NewErrorFrom(sdk.ErrEnvironmentNotFound, "can not found a environment with id %d", n.Context.EnvironmentID)
			}

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
			appDB, errA := application.LoadByID(db, store, n.Context.ApplicationID, application.LoadOptions.WithDeploymentStrategies, application.LoadOptions.WithVariables)
			if errA != nil {
				return errA
			}
			app = *appDB
			if app.ProjectKey != proj.Key {
				return sdk.NewErrorFrom(sdk.ErrResourceNotInProject, "can not found a application with id %d", n.Context.ApplicationID)
			}

			w.Applications[n.Context.ApplicationID] = app
		}
		n.Context.ApplicationName = app.Name
		return nil
	}
	if n.Context.ApplicationName != "" {
		appDB, err := application.LoadByName(db, store, proj.Key, n.Context.ApplicationName, application.LoadOptions.WithDeploymentStrategies, application.LoadOptions.WithVariables)
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrPipelineNotFound) {
				return sdk.ErrorWithData(sdk.ErrApplicationNotFound, n.Context.ApplicationName)
			}
			return sdk.WrapError(err, "unable to load application %s", n.Context.ApplicationName)
		}
		w.Applications[appDB.ID] = *appDB
		n.Context.ApplicationID = appDB.ID
	}
	return nil
}

// CheckPipeline checks pipeline data
func checkPipeline(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project, w *sdk.Workflow, n *sdk.Node, opts LoadOptions) error {
	if n.Context.PipelineID != 0 {
		pip, ok := w.Pipelines[n.Context.PipelineID]
		if !ok {
			// Load pipeline from db to get stage/jobs
			pipDB, err := pipeline.LoadPipelineByID(ctx, db, n.Context.PipelineID, opts.DeepPipeline)
			if err != nil {
				return sdk.WrapError(err, "unable to load pipeline %d", n.Context.PipelineID)
			}
			pip = *pipDB

			if pip.ProjectKey != proj.Key {
				return sdk.NewErrorFrom(sdk.ErrResourceNotInProject, "can not found a pipeline with id %d", n.Context.PipelineID)
			}

			w.Pipelines[n.Context.PipelineID] = pip
		}
		n.Context.PipelineName = pip.Name
		return nil
	}
	if n.Context.PipelineName != "" {
		pipDB, err := pipeline.LoadPipeline(db, proj.Key, n.Context.PipelineName, opts.DeepPipeline)
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
	allMsg := []sdk.Message{}

	data, err := ExtractFromCDSFiles(tr)
	if err != nil {
		return nil, nil, err
	}

	var workflowExists bool
	var oldWf *sdk.Workflow

	if opts != nil && opts.OldWorkflow != nil {
		oldWf = opts.OldWorkflow
	} else {
		// load the workflow from database if exists
		workflowExists, err = Exists(db, proj.Key, data.wrkflw.Name)
		if err != nil {
			return nil, nil, sdk.WrapError(err, "Cannot check if workflow exists")
		}
		if workflowExists {
			oldWf, err = Load(ctx, db, store, proj, data.wrkflw.Name, u, LoadOptions{WithIcon: true})
			if err != nil {
				return nil, nil, sdk.WrapError(err, "Unable to load existing workflow")
			}
		}
	}

	// if a old workflow as code exists, we want to check if the new workflow is also as code on the same repository
	if oldWf != nil && oldWf.FromRepository != "" && (opts == nil || opts.FromRepository != oldWf.FromRepository) {
		return nil, nil, sdk.WithStack(sdk.ErrWorkflowAlreadyAsCode)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, nil, sdk.WrapError(err, "Unable to start tx")
	}
	defer tx.Rollback()

	for filename, app := range data.apps {
		log.Debug("Push> Parsing %s", filename)
		var fromRepo string
		if opts != nil {
			fromRepo = opts.FromRepository
		}
		appDB, msgList, err := application.ParseAndImport(tx, store, proj, &app, application.ImportOptions{Force: true, FromRepository: fromRepo}, decryptFunc, u)
		if err != nil {
			return nil, nil, sdk.ErrorWithFallback(err, sdk.ErrWrongRequest, "unable to import application %s/%s", proj.Key, app.Name)
		}
		allMsg = append(allMsg, msgList...)
		proj.SetApplication(*appDB)
		log.Debug("Push> -- %s OK", filename)
	}

	for filename, env := range data.envs {
		log.Debug("Push> Parsing %s", filename)
		var fromRepo string
		if opts != nil {
			fromRepo = opts.FromRepository
		}
		envDB, msgList, err := environment.ParseAndImport(tx, proj, &env, environment.ImportOptions{Force: true, FromRepository: fromRepo}, decryptFunc, u)
		if err != nil {
			return nil, nil, sdk.ErrorWithFallback(err, sdk.ErrWrongRequest, "unable to import environment %s/%s", proj.Key, env.Name)
		}
		allMsg = append(allMsg, msgList...)
		proj.SetEnvironment(*envDB)
		log.Debug("Push> -- %s OK", filename)
	}

	for filename, pip := range data.pips {
		log.Debug("Push> Parsing %s", filename)
		var fromRepo string
		if opts != nil {
			fromRepo = opts.FromRepository
		}
		pipDB, msgList, err := pipeline.ParseAndImport(ctx, tx, store, proj, &pip, u, pipeline.ImportOptions{Force: true, FromRepository: fromRepo})
		if err != nil {
			return nil, nil, sdk.ErrorWithFallback(err, sdk.ErrWrongRequest, "unable to import pipeline %s/%s", proj.Key, pip.Name)

		}
		allMsg = append(allMsg, msgList...)
		proj.SetPipeline(*pipDB)
		log.Debug("Push> -- %s OK", filename)
	}

	isDefaultBranch := true
	if opts != nil {
		isDefaultBranch = opts.IsDefaultBranch
	}

	// In workflow as code context, if we only have the repowebhook, we skip it
	//  because it will be automatically recreated later with the proper configuration
	if opts != nil && opts.FromRepository != "" {
		if len(data.wrkflw.Workflow) == 0 {
			if len(data.wrkflw.PipelineHooks) == 1 && data.wrkflw.PipelineHooks[0].Model == sdk.RepositoryWebHookModelName {
				data.wrkflw.PipelineHooks = nil
			}
		} else {
			for node, hooks := range data.wrkflw.Hooks {
				if len(hooks) == 1 && hooks[0].Model == sdk.RepositoryWebHookModelName {
					data.wrkflw.Hooks[node] = nil
				}
			}
		}
	}

	var importOptions = ImportOptions{
		Force: true,
	}

	if opts != nil {
		importOptions.FromRepository = opts.FromRepository
		importOptions.IsDefaultBranch = opts.IsDefaultBranch
		importOptions.FromBranch = opts.Branch
		importOptions.VCSServer = opts.VCSServer
		importOptions.RepositoryName = opts.RepositoryName
		importOptions.RepositoryStrategy = opts.RepositoryStrategy
		importOptions.HookUUID = opts.HookUUID
	}

	wf, msgList, err := ParseAndImport(ctx, tx, store, proj, oldWf, &data.wrkflw, u, importOptions)
	if err != nil {
		return msgList, nil, sdk.WrapError(err, "unable to import workflow %s", data.wrkflw.Name)
	}

	// If the workflow is "as-code", it should always be linked to a git repository
	if opts != nil && opts.FromRepository != "" {
		if wf.WorkflowData.Node.Context.ApplicationID == 0 {
			return nil, nil, sdk.WithStack(sdk.ErrApplicationMandatoryOnWorkflowAsCode)
		}
		app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]
		if app.VCSServer == "" || app.RepositoryFullname == "" {
			return nil, nil, sdk.WithStack(sdk.ErrApplicationMandatoryOnWorkflowAsCode)
		}
	}

	allMsg = append(allMsg, msgList...)

	if !isDefaultBranch {
		_ = tx.Rollback()
		log.Debug("workflow %s rollbacked because it's not comming from the default branch", wf.Name)
	} else {
		if err := tx.Commit(); err != nil {
			return nil, nil, sdk.WrapError(err, "Cannot commit transaction")
		}

		if oldWf != nil {
			event.PublishWorkflowUpdate(proj.Key, *wf, *oldWf, u)
		} else {
			event.PublishWorkflowAdd(proj.Key, *wf, u)
		}

		log.Debug("workflow %s updated", wf.Name)
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
		return false, sdk.WithStack(err)
	}

	return nb > 0, nil
}
