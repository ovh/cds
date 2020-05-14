package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type LoadAllWorkflowsOptionsFilters struct {
	ProjectKey            string
	WorkflowName          string
	VCSServer             string
	ApplicationRepository string
	FromRepository        string
	GroupIDs              []int64
	WorkflowIDs           []int64
}

type LoadAllWorkflowsOptionsLoaders struct {
	WithApplications       bool
	WithPipelines          bool
	WithDeepPipelines      bool
	WithEnvironments       bool
	WithIntegrations       bool
	WithIcon               bool
	WithAsCodeUpdateEvents bool
	WithTemplate           bool
	WithNotifications      bool
	WithLabels             bool
	WithAudits             bool
	WithFavoritesForUserID string
}

type LoadAllWorkflowsOptions struct {
	Filters   LoadAllWorkflowsOptionsFilters
	Loaders   LoadAllWorkflowsOptionsLoaders
	Offset    int
	Limit     int
	Ascending bool
	Lock      bool
}

func (opt LoadAllWorkflowsOptions) Query() gorpmapping.Query {
	var queryString = `
    WITH 
    workflow_root_application_id AS (
        SELECT 
            id as "workflow_id", 
            project_id,
            name as "workflow_name",
            (workflow_data -> 'node' -> 'context' ->> 'application_id')::BIGINT as "root_application_id"
        FROM workflow
    ),
    project_permission AS (
        SELECT 
            project_id,
            ARRAY_AGG(group_id) as "groups"
        FROM project_group
        GROUP BY project_id
    ),
    selected_workflow AS (
        SELECT 
        project.id, 
            workflow_root_application_id.workflow_id, 
            project.projectkey, 
            workflow_name, 
            application.id, 
            application.name, 
            application.vcs_server, 
            application.repo_fullname, 
            project_permission.groups
            FROM workflow_root_application_id
        LEFT OUTER JOIN application ON application.id = root_application_id
        JOIN project ON project.id = workflow_root_application_id.project_id
        JOIN project_permission ON project_permission.project_id = project.id	
    )
    SELECT workflow.* , selected_workflow.projectkey as "project_key"
    FROM workflow 
	JOIN selected_workflow ON selected_workflow.workflow_id = workflow.id
	WHERE workflow.to_delete  = false
    `

	var filters []string
	var args []interface{}
	if opt.Filters.ProjectKey != "" {
		filters = append(filters, "selected_workflow.projectkey = $%d")
		args = append(args, opt.Filters.ProjectKey)
	}
	if opt.Filters.WorkflowName != "" {
		filters = append(filters, "selected_workflow.workflow_name = $%d")
		args = append(args, opt.Filters.WorkflowName)
	}
	if opt.Filters.VCSServer != "" {
		filters = append(filters, "selected_workflow.vcs_server = $%d")
		args = append(args, opt.Filters.VCSServer)
	}
	if opt.Filters.ApplicationRepository != "" {
		filters = append(filters, "selected_workflow.repo_fullname = $%d")
		args = append(args, opt.Filters.ApplicationRepository)
	}
	if opt.Filters.FromRepository != "" {
		filters = append(filters, "workflow.from_repository = $%d")
		args = append(args, opt.Filters.FromRepository)
	}
	if len(opt.Filters.GroupIDs) != 0 {
		filters = append(filters, "selected_workflow.groups && $%d")
		args = append(args, pq.Int64Array(opt.Filters.GroupIDs))
	}
	if len(opt.Filters.WorkflowIDs) != 0 {
		filters = append(filters, "workflow.id = ANY($%d)")
		args = append(args, pq.Int64Array(opt.Filters.WorkflowIDs))
	}

	for i, f := range filters {
		queryString += " AND "
		queryString += fmt.Sprintf(f, i+1)
	}

	var order = " ORDER BY selected_workflow.projectkey, selected_workflow.workflow_name "
	if opt.Ascending {
		order += "ASC"
	} else {
		order += "DESC"
	}
	queryString += order

	if opt.Offset != 0 {
		queryString += fmt.Sprintf(" OFFSET %d", opt.Offset)
	}

	if opt.Limit != 0 {
		queryString += fmt.Sprintf(" LIMIT %d", opt.Limit)
	}

	if opt.Lock {
		queryString += " for update skip locked"
	}

	q := gorpmapping.NewQuery(queryString).Args(args...)

	log.Debug("workflow.LoadAllWorkflowsOptions.Query> %v", q)

	return q
}

func (opt LoadAllWorkflowsOptions) GetLoaders() []gorpmapping.GetOptionFunc {

	var loaders = []gorpmapping.GetOptionFunc{}

	if opt.Loaders.WithApplications {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withApplications(db, ws)
		})
	}

	if opt.Loaders.WithEnvironments {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withEnvironments(db, ws)
		})
	}

	if opt.Loaders.WithDeepPipelines {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withPipelines(db, ws, true)
		})
	} else if opt.Loaders.WithPipelines {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withPipelines(db, ws, false)
		})
	}

	if opt.Loaders.WithAsCodeUpdateEvents {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withAsCodeUpdateEvents(db, ws)
		})
	}

	if !opt.Loaders.WithIcon {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			for j := range *ws {
				w := (*ws)[j]
				w.Icon = ""
			}
			return nil
		})
	}

	if opt.Loaders.WithIntegrations {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withIntegrations(db, ws)
		})
	}

	if opt.Loaders.WithTemplate {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withTemplates(db, ws)
		})
	}

	if opt.Loaders.WithNotifications {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withNotifications(db, ws)
		})
	}

	if opt.Loaders.WithLabels {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withLabels(db, ws)
		})
	}

	if opt.Loaders.WithFavoritesForUserID != "" {
		loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return opt.withFavorites(db, ws, opt.Loaders.WithFavoritesForUserID)
		})
	}

	loaders = append(loaders, func(db gorp.SqlExecutor, i interface{}) error {
		ws := i.(*[]Workflow)
		return opt.withGroups(db, ws)
	})

	return loaders
}

func (opt LoadAllWorkflowsOptions) withGroups(db gorp.SqlExecutor, ws *[]Workflow) error {
	var mapIDs = map[int64]*Workflow{}
	for _, w := range *ws {
		mapIDs[w.ID] = &w
	}

	var ids = make([]int64, 0, len(mapIDs))
	for id := range mapIDs {
		ids = append(ids, id)
	}

	perms, err := group.LoadWorkflowGroupsByWorkflowIDs(db, ids)
	if err != nil {
		return err
	}

	for workflowID, perm := range perms {
		for i, w := range *ws {
			if w.ID == workflowID {
				w.Groups = perm
				(*ws)[i] = w
				break
			}
		}
	}

	return nil
}

func (opt LoadAllWorkflowsOptions) withEnvironments(db gorp.SqlExecutor, ws *[]Workflow) error {
	var mapIDs = map[int64]*sdk.Environment{}
	for _, w := range *ws {
		nodesArray := w.WorkflowData.Array()
		for _, n := range nodesArray {
			if n.Context != nil && n.Context.EnvironmentID != 0 {
				if _, ok := mapIDs[n.Context.EnvironmentID]; !ok {
					mapIDs[n.Context.EnvironmentID] = nil
				}
			}
		}
	}

	var ids = make([]int64, 0, len(mapIDs))
	for id := range mapIDs {
		ids = append(ids, id)
	}

	envs, err := environment.LoadAllByIDs(db, ids)
	if err != nil {
		return err
	}

	for id := range mapIDs {
		for i := range envs {
			if id == envs[i].ID {
				mapIDs[id] = &envs[i]
			}
		}
	}

	for x := range *ws {
		w := &(*ws)[x]
		w.InitMaps()
		nodesArray := w.WorkflowData.Array()
		for i := range nodesArray {
			n := nodesArray[i]
			if n.Context != nil && n.Context.EnvironmentID != 0 {
				if env, ok := mapIDs[n.Context.EnvironmentID]; ok {
					if env == nil {
						return sdk.WrapError(sdk.ErrNotFound, "unable to find environment %d", n.Context.EnvironmentID)
					}
					w.Environments[n.Context.EnvironmentID] = *env
				}
			}
		}
	}

	return nil
}

func (opt LoadAllWorkflowsOptions) withPipelines(db gorp.SqlExecutor, ws *[]Workflow, deep bool) error {
	var mapIDs = map[int64]*sdk.Pipeline{}
	for _, w := range *ws {
		nodesArray := w.WorkflowData.Array()
		for _, n := range nodesArray {
			if n.Context != nil && n.Context.PipelineID != 0 {
				if _, ok := mapIDs[n.Context.PipelineID]; !ok {
					mapIDs[n.Context.PipelineID] = nil
				}
			}
		}
	}

	var ids = make([]int64, 0, len(mapIDs))
	for id := range mapIDs {
		ids = append(ids, id)
	}

	pips, err := pipeline.LoadAllByIDs(db, ids, deep)
	if err != nil {
		return err
	}

	for id := range mapIDs {
		for i := range pips {
			if id == pips[i].ID {
				mapIDs[id] = &pips[i]
			}
		}
	}

	for x := range *ws {
		w := &(*ws)[x]
		w.InitMaps()
		nodesArray := w.WorkflowData.Array()
		for i := range nodesArray {
			n := nodesArray[i]
			if n.Context != nil && n.Context.PipelineID != 0 {
				if pip, ok := mapIDs[n.Context.PipelineID]; ok {
					if pip == nil {
						return sdk.WrapError(sdk.ErrNotFound, "unable to find pipeline %d", n.Context.PipelineID)
					}
					w.Pipelines[n.Context.PipelineID] = *pip
				}
			}
		}
	}

	return nil
}

func (opt LoadAllWorkflowsOptions) withTemplates(db gorp.SqlExecutor, ws *[]Workflow) error {
	var mapIDs = map[int64]struct{}{}
	for _, w := range *ws {
		mapIDs[w.ID] = struct{}{}
	}

	var ids = make([]int64, 0, len(mapIDs))
	for id := range mapIDs {
		ids = append(ids, id)
	}

	wtis, err := workflowtemplate.LoadInstanceByWorkflowIDs(context.Background(), db, ids, workflowtemplate.LoadInstanceOptions.WithTemplate)
	if err != nil {
		return err
	}

	for x := range *ws {
		w := &(*ws)[x]
		w.InitMaps()
		for _, wti := range wtis {
			if wti.WorkflowID != nil && w.ID == *wti.WorkflowID {
				w.TemplateInstance = &wti
				w.FromTemplate = fmt.Sprintf("%s@%d", wti.Template.Path(), wti.WorkflowTemplateVersion)
				w.TemplateUpToDate = wti.Template.Version == wti.WorkflowTemplateVersion
				break
			}
		}
	}

	return nil
}

func (opt LoadAllWorkflowsOptions) withIntegrations(db gorp.SqlExecutor, ws *[]Workflow) error {
	var mapIDs = map[int64]*sdk.ProjectIntegration{}
	for _, w := range *ws {
		nodesArray := w.WorkflowData.Array()
		for _, n := range nodesArray {
			if n.Context != nil && n.Context.ProjectIntegrationID != 0 {
				log.Debug("found ProjectIntegrationID=%d(%s) on workflow %d, node=%d", n.Context.ProjectIntegrationID, n.Context.ProjectIntegrationName, w.ID, n.ID)
				if _, ok := mapIDs[n.Context.ProjectIntegrationID]; !ok {
					mapIDs[n.Context.ProjectIntegrationID] = nil
				}
			}
		}
	}

	var ids = make([]int64, 0, len(mapIDs))
	for id := range mapIDs {
		ids = append(ids, id)
	}

	projectIntegrations, err := integration.LoadIntegrationsByIDs(db, ids)
	if err != nil {
		return err
	}

	for id := range mapIDs {
		for i := range projectIntegrations {
			if id == projectIntegrations[i].ID {
				mapIDs[id] = &projectIntegrations[i]
			}
		}
	}

	for x := range *ws {
		w := &(*ws)[x]
		w.InitMaps()
		nodesArray := w.WorkflowData.Array()
		for i := range nodesArray {
			n := nodesArray[i]
			if n.Context != nil && n.Context.ProjectIntegrationID != 0 {
				if integ, ok := mapIDs[n.Context.ProjectIntegrationID]; ok {
					if integ == nil {
						return sdk.WrapError(sdk.ErrNotFound, "unable to find integration %d", n.Context.ProjectIntegrationID)
					}
					w.ProjectIntegrations[n.Context.ProjectIntegrationID] = *integ
				}
			}
		}
	}

	return nil
}

func (opt LoadAllWorkflowsOptions) withAsCodeUpdateEvents(db gorp.SqlExecutor, ws *[]Workflow) error {
	var ids = make([]int64, 0, len(*ws))
	for _, w := range *ws {
		ids = append(ids, w.ID)
	}

	asCodeEvents, err := ascode.LoadAsCodeEventByWorkflowIDs(context.Background(), db, ids)
	if err != nil {
		return err
	}

	for x := range *ws {
		w := &(*ws)[x]
		w.InitMaps()

		for _, evt := range asCodeEvents {
			if _, ok := evt.Data.Workflows[w.ID]; ok {
				w.AsCodeEvent = append(w.AsCodeEvent, evt)
			}
		}
	}

	return nil
}

func (opt LoadAllWorkflowsOptions) withApplications(db gorp.SqlExecutor, ws *[]Workflow) error {
	var mapIDs = map[int64]*sdk.Application{}
	for _, w := range *ws {
		nodesArray := w.WorkflowData.Array()
		for _, n := range nodesArray {
			if n.Context != nil && n.Context.ApplicationID != 0 {
				if _, ok := mapIDs[n.Context.ApplicationID]; !ok {
					mapIDs[n.Context.ApplicationID] = nil
				}
			}
		}
	}

	var ids = make([]int64, 0, len(mapIDs))
	for id := range mapIDs {
		ids = append(ids, id)
	}

	apps, err := application.LoadAllByIDs(db, ids)
	if err != nil {
		return err
	}

	for id := range mapIDs {
		for i := range apps {
			if id == apps[i].ID {
				mapIDs[id] = &apps[i]
			}
		}
	}

	for x := range *ws {
		w := &(*ws)[x]
		w.InitMaps()
		nodesArray := w.WorkflowData.Array()
		for i := range nodesArray {
			n := nodesArray[i]
			if n.Context != nil && n.Context.ApplicationID != 0 {
				if app, ok := mapIDs[n.Context.ApplicationID]; ok {
					if app == nil {
						return sdk.WrapError(sdk.ErrNotFound, "unable to find application %d", n.Context.ApplicationID)
					}
					w.Applications[n.Context.ApplicationID] = *app
				}
			}
		}
	}

	return nil
}

func (opt LoadAllWorkflowsOptions) withNotifications(db gorp.SqlExecutor, ws *[]Workflow) error {
	return nil
}

func (opt LoadAllWorkflowsOptions) withLabels(db gorp.SqlExecutor, ws *[]Workflow) error {

	var ids = make([]int64, 0, len(*ws))
	for _, w := range *ws {
		ids = append(ids, w.ID)
	}

	labels, err := LoadLabels(db, ids...)
	if err != nil {
		return err
	}

	for x := range *ws {
		w := &(*ws)[x]
		for _, label := range labels {
			if w.ID == label.WorkflowID {
				w.Labels = append(w.Labels, label)
			}
		}
	}

	return nil
}

func (opt LoadAllWorkflowsOptions) withFavorites(db gorp.SqlExecutor, ws *[]Workflow, userID string) error {
	workflowIDs, err := UserFavoriteWorkflowIDs(db, userID)
	if err != nil {
		return err
	}

	for x := range *ws {
		w := &(*ws)[x]
		w.Favorite = sdk.IsInInt64Array(w.ID, workflowIDs)
	}

	return nil
}

func LoadAllWorkflows(ctx context.Context, db gorp.SqlExecutor, opts LoadAllWorkflowsOptions) (sdk.Workflows, error) {
	t0 := time.Now()

	var workflows []Workflow
	if err := gorpmapping.GetAll(ctx, db, opts.Query(), &workflows, opts.GetLoaders()...); err != nil {
		return nil, err
	}
	ws := make(sdk.Workflows, 0, len(workflows))
	for i := range workflows {
		if err := workflows[i].PostGet(db); err != nil {
			return nil, err
		}
		w := workflows[i].Get()
		w.Normalize()
		ws = append(ws, w)
	}

	// TODO load workflow groupd and node_groups properly in mandatory loaders

	delta := time.Since(t0).Seconds()
	log.Debug("LoadAllWorkflows - %d results in %.3f seconds", len(ws), delta)

	return ws, nil
}
