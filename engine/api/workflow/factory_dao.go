package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

// LoadOptions custom option for loading workflow
type LoadOptions struct {
	Minimal                bool
	DeepPipeline           bool
	WithLabels             bool
	WithIcon               bool
	WithAsCodeUpdateEvent  bool
	WithIntegrations       bool
	WithTemplate           bool
	WithFavoritesForUserID string
}

type asCodeLoader struct {
	db     gorp.SqlExecutor
	vcs    map[string]*sdk.VCSProject
	repo   map[string]*sdk.ProjectRepository
	action map[string]sdk.V2Action
}

func (loadOpts LoadOptions) GetWorkflowDAO() WorkflowDAO {
	var dao WorkflowDAO

	if !loadOpts.Minimal {
		dao.Loaders.WithPipelines = true
		dao.Loaders.WithApplications = true
		dao.Loaders.WithEnvironments = true
		dao.Loaders.WithIntegrations = true
		dao.Loaders.WithFavoritesForUserID = loadOpts.WithFavoritesForUserID

		if loadOpts.WithIcon {
			dao.Loaders.WithIcon = true
		}
		if loadOpts.WithAsCodeUpdateEvent {
			dao.Loaders.WithAsCodeUpdateEvents = true
		}
		if loadOpts.WithTemplate {
			dao.Loaders.WithTemplate = true
		}
		if loadOpts.DeepPipeline {
			dao.Loaders.WithDeepPipelines = true
		}
		if loadOpts.WithLabels {
			dao.Loaders.WithLabels = true
		}
	}
	return dao
}

type LoadAllWorkflowsOptionsFilters struct {
	ProjectKey                   string
	WorkflowName                 string
	VCSServer                    string
	ApplicationRepository        string
	FromRepository               string
	GroupIDs                     []int64
	WorkflowIDs                  []int64
	DisableFilterDeletedWorkflow bool
	AsCode                       bool
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
	WithLabels             bool
	WithAudits             bool
	WithFavoritesForUserID string
	WithRuns               int
}

type WorkflowDAO struct {
	Filters   LoadAllWorkflowsOptionsFilters
	Loaders   LoadAllWorkflowsOptionsLoaders
	Offset    int
	Limit     int
	Ascending bool
	Lock      bool
}

func (dao WorkflowDAO) Query() gorpmapping.Query {
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
	JOIN project_permission ON project_permission.project_id = workflow.project_id
    `

	var filters []string
	var args []interface{}
	if dao.Filters.ProjectKey != "" {
		filters = append(filters, "selected_workflow.projectkey = $%d")
		args = append(args, dao.Filters.ProjectKey)
	}
	if dao.Filters.WorkflowName != "" {
		filters = append(filters, "selected_workflow.workflow_name = $%d")
		args = append(args, dao.Filters.WorkflowName)
	}
	if dao.Filters.VCSServer != "" {
		filters = append(filters, "selected_workflow.vcs_server = $%d")
		args = append(args, dao.Filters.VCSServer)
	}
	if dao.Filters.ApplicationRepository != "" {
		filters = append(filters, "selected_workflow.repo_fullname = $%d")
		args = append(args, dao.Filters.ApplicationRepository)
	}
	if dao.Filters.FromRepository != "" {
		filters = append(filters, "workflow.from_repository = $%d")
		args = append(args, dao.Filters.FromRepository)
	}
	if len(dao.Filters.GroupIDs) != 0 {
		filters = append(filters, "selected_workflow.groups && $%d")
		args = append(args, pq.Int64Array(dao.Filters.GroupIDs))
	}
	if len(dao.Filters.WorkflowIDs) != 0 {
		filters = append(filters, "workflow.id = ANY($%d)")
		args = append(args, pq.Int64Array(dao.Filters.WorkflowIDs))
	}

	queryFilters := make([]string, 0, len(filters))
	for i, f := range filters {
		queryFilters = append(queryFilters, fmt.Sprintf(f, i+1))
	}

	if !dao.Filters.DisableFilterDeletedWorkflow {
		queryFilters = append(queryFilters, "workflow.to_delete  = false")
	}

	if dao.Filters.AsCode {
		queryFilters = append(queryFilters, "workflow.from_repository <> ''")
	}

	if len(queryFilters) > 0 {
		queryString += "WHERE " + strings.Join(queryFilters, " AND ")
	}

	var order = " ORDER BY selected_workflow.projectkey, selected_workflow.workflow_name "
	if dao.Ascending {
		order += "ASC"
	} else {
		order += "DESC"
	}
	queryString += order

	if dao.Offset != 0 {
		queryString += fmt.Sprintf(" OFFSET %d", dao.Offset)
	}

	if dao.Limit != 0 {
		queryString += fmt.Sprintf(" LIMIT %d", dao.Limit)
	}

	if dao.Lock {
		queryString += " for update skip locked"
	}

	q := gorpmapping.NewQuery(queryString).Args(args...)

	log.Debug(context.TODO(), "workflow.WorkflowDAO.Query> %v", q)

	return q
}

func (dao WorkflowDAO) GetLoaders() []gorpmapping.GetOptionFunc {

	var loaders = make([]gorpmapping.GetOptionFunc, 0)

	if dao.Loaders.WithApplications {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withApplications(ctx, db, ws)
		})
	}

	if dao.Loaders.WithEnvironments {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withEnvironments(ctx, db, ws)
		})
	}

	if dao.Loaders.WithDeepPipelines {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withPipelines(ctx, db, ws, true)
		})
	} else if dao.Loaders.WithPipelines {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withPipelines(ctx, db, ws, false)
		})
	}

	if dao.Loaders.WithAsCodeUpdateEvents {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withAsCodeUpdateEvents(ctx, db, ws)
		})
	}

	if !dao.Loaders.WithIcon {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			for j := range *ws {
				w := (*ws)[j]
				w.Icon = ""
			}
			return nil
		})
	}

	if dao.Loaders.WithIntegrations {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withIntegrations(ctx, db, ws)
		})
	}

	if dao.Loaders.WithTemplate {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withTemplates(ctx, db, ws)
		})
	}

	if dao.Loaders.WithLabels {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withLabels(ctx, db, ws)
		})
	}

	if dao.Loaders.WithFavoritesForUserID != "" {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withFavorites(ctx, db, ws, dao.Loaders.WithFavoritesForUserID)
		})
	}

	if dao.Loaders.WithRuns != 0 {
		loaders = append(loaders, func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withRuns(ctx, db, ws, dao.Loaders.WithRuns)
		})
	}

	loaders = append(loaders,
		func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withGroups(ctx, db, ws)
		},
		func(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
			ws := i.(*[]Workflow)
			return dao.withNotifications(ctx, db, ws)
		})

	return loaders
}

func (dao WorkflowDAO) withGroups(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow) error {
	_, end := telemetry.Span(ctx, "workflowDAO.withGroups")
	defer end()

	var mapIDs = map[int64]*Workflow{}
	for _, w := range *ws {
		mapIDs[w.ID] = &w
	}

	var ids = make([]int64, 0, len(mapIDs))
	for id := range mapIDs {
		ids = append(ids, id)
	}

	perms, err := group.LoadWorkflowGroupsByWorkflowIDs(ctx, db, ids)
	if err != nil {
		return err
	}

	for workflowID, perm := range perms {
		for i, w := range *ws {
			if w.ID == workflowID {
				w.Groups = perm
				w.Organization, err = w.Groups.ComputeOrganization()
				if err != nil {
					return err
				}
				(*ws)[i] = w
				break
			}
		}
	}

	return nil
}

func (dao WorkflowDAO) withEnvironments(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow) error {
	_, end := telemetry.Span(ctx, "workflowDAO.withEnvironments")
	defer end()

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

func (dao WorkflowDAO) withPipelines(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow, deep bool) error {
	_, end := telemetry.Span(ctx, "workflowDAO.withPipelines")
	defer end()

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

					// Load ascode actions
					if deep {
						ascodeLoader := &asCodeLoader{
							db:     db,
							vcs:    make(map[string]*sdk.VCSProject),
							repo:   make(map[string]*sdk.ProjectRepository),
							action: make(map[string]sdk.V2Action),
						}
						for stageIndex := range pip.Stages {
							s := &pip.Stages[stageIndex]
							for jobIndex := range s.Jobs {
								j := &s.Jobs[jobIndex]
								if err := ascodeLoader.browseNonAscodeActionSteps(ctx, j.Action.Actions); err != nil {
									return err
								}
							}
						}
						w.AscodeActions = ascodeLoader.action
					}
					w.Pipelines[n.Context.PipelineID] = *pip
				}
			}
		}
	}
	return nil
}

func (loader *asCodeLoader) browseNonAscodeActionSteps(ctx context.Context, actions []sdk.Action) error {
	for _, a := range actions {
		if a.Type != sdk.AsCodeAction && len(a.Actions) > 0 {
			if err := loader.browseNonAscodeActionSteps(ctx, a.Actions); err != nil {
				return err
			}
			continue
		}
		if a.Type == sdk.AsCodeAction {
			if err := loader.loadAsCodeActionStep(ctx, strings.TrimPrefix(a.StepName, "actions/")); err != nil {
				return err
			}
		}
	}
	return nil
}

func (loader *asCodeLoader) loadAsCodeActionStep(ctx context.Context, stepName string) error {
	// If already loaded, skip it
	if _, has := loader.action[stepName]; has {
		return nil
	}

	branchSplit := strings.Split(stepName, "@")
	var projKey, vcsName, repoName, actionName, branch string
	if len(branchSplit) != 2 {
		return nil
	}
	branch = branchSplit[1]
	actionSplit := strings.Split(branchSplit[0], "/")
	if len(actionSplit) != 5 {
		return nil
	}
	projKey = actionSplit[0]
	vcsName = actionSplit[1]
	repoName = actionSplit[2] + "/" + actionSplit[3]
	actionName = actionSplit[4]
	vcsServer, has := loader.vcs[projKey+"/"+vcsName]
	if !has {
		var err error
		vcsServer, err = vcs.LoadVCSByProject(ctx, loader.db, projKey, vcsName)
		if err != nil {
			return err
		}
		loader.vcs[projKey+"/"+vcsName] = vcsServer
	}
	repo, has := loader.repo[projKey+"/"+vcsName+"/"+repoName]
	if !has {
		var err error
		repo, err = repository.LoadRepositoryByName(ctx, loader.db, vcsServer.ID, repoName)
		if err != nil {
			return err
		}
		loader.repo[projKey+"/"+vcsName+"/"+repoName] = repo
	}
	var currentAction sdk.V2Action
	if err := entity.LoadAndUnmarshalByBranchTypeName(ctx, loader.db, repo.ID, branch, sdk.EntityTypeAction, actionName, &currentAction); err != nil {
		return err
	}
	loader.action[stepName] = currentAction
	if err := loader.browseAsCodeActionSteps(ctx, currentAction.Runs.Steps); err != nil {
		return err
	}
	return nil
}

func (loader *asCodeLoader) browseAsCodeActionSteps(ctx context.Context, steps []sdk.ActionStep) error {
	for _, step := range steps {
		if step.Uses == "" {
			continue
		}
		if err := loader.loadAsCodeActionStep(ctx, strings.TrimPrefix(step.Uses, "actions/")); err != nil {
			return err
		}
	}
	return nil
}

func (dao WorkflowDAO) withTemplates(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow) error {
	ctx, end := telemetry.Span(ctx, "workflowDAO.withTemplates")
	defer end()

	var mapIDs = map[int64]struct{}{}
	for _, w := range *ws {
		mapIDs[w.ID] = struct{}{}
	}

	var ids = make([]int64, 0, len(mapIDs))
	for id := range mapIDs {
		ids = append(ids, id)
	}

	wtis, err := workflowtemplate.LoadInstanceByWorkflowIDs(ctx, db, ids, workflowtemplate.LoadInstanceOptions.WithTemplate)
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

func (dao WorkflowDAO) withIntegrations(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow) error {
	ctx, end := telemetry.Span(ctx, "workflowDAO.withIntegrations")
	defer end()

	var mapIDs = map[int64]*sdk.ProjectIntegration{}
	for _, w := range *ws {
		nodesArray := w.WorkflowData.Array()
		for _, n := range nodesArray {
			if n.Context != nil && n.Context.ProjectIntegrationID != 0 {
				log.Debug(context.TODO(), "found ProjectIntegrationID=%d(%s) on workflow %d, node=%d", n.Context.ProjectIntegrationID, n.Context.ProjectIntegrationName, w.ID, n.ID)
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

	projectIntegrations, err := integration.LoadIntegrationsByIDs(ctx, db, ids)
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
		var err error
		w.Integrations, err = LoadWorkflowIntegrationsByWorkflowID(ctx, db, w.ID)
		if err != nil {
			return err
		}

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

func (dao WorkflowDAO) withAsCodeUpdateEvents(_ context.Context, db gorp.SqlExecutor, ws *[]Workflow) error {
	var ids = make([]int64, 0, len(*ws))
	for _, w := range *ws {
		ids = append(ids, w.ID)
	}

	asCodeEvents, err := ascode.LoadEventsByWorkflowIDs(context.Background(), db, ids)
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

func (dao WorkflowDAO) withApplications(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow) error {
	_, end := telemetry.Span(ctx, "workflowDAO.withApplications")
	defer end()
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

	apps, err := application.LoadAllByIDs(ctx, db, ids, application.LoadOptions.WithVariables, application.LoadOptions.WithDeploymentStrategies, application.LoadOptions.WithKeys)
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

func (dao WorkflowDAO) withNotifications(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow) error {
	_, end := telemetry.Span(ctx, "workflowDAO.withNotifications")
	defer end()
	var ids = make([]int64, 0, len(*ws))
	for _, w := range *ws {
		ids = append(ids, w.ID)
	}

	notificationsMap, err := LoadNotificationsByWorkflowIDs(db, ids)
	if err != nil {
		return err
	}

	for x := range *ws {
		w := &(*ws)[x]
		w.Notifications = notificationsMap[w.ID]
	}
	return nil
}

func (dao WorkflowDAO) withLabels(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow) error {
	_, end := telemetry.Span(ctx, "workflowDAO.withLabels")
	defer end()
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

func (dao WorkflowDAO) withRuns(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow, limit int) error {
	_, end := telemetry.Span(ctx, "workflowDAO.withRuns")
	defer end()
	var ids = make([]int64, 0, len(*ws))
	for _, w := range *ws {
		ids = append(ids, w.ID)
	}

	runs, err := LoadLastRuns(ctx, db, ids, limit)
	if err != nil {
		return err
	}

	for x := range *ws {
		w := &(*ws)[x]
		for _, run := range runs {
			if w.ID == run.WorkflowID {
				w.Runs = append(w.Runs, run)
			}
		}
	}

	return nil
}

func (dao WorkflowDAO) withFavorites(ctx context.Context, db gorp.SqlExecutor, ws *[]Workflow, userID string) error {
	_, end := telemetry.Span(ctx, "workflowDAO.withFavorites")
	defer end()
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

func (dao WorkflowDAO) Load(ctx context.Context, db gorp.SqlExecutor) (sdk.Workflow, error) {
	ctx, end := telemetry.Span(ctx, "workflowDAO.Load")
	defer end()
	dao.Limit = 1
	ws, err := dao.LoadAll(ctx, db)
	if err != nil {
		return sdk.Workflow{}, err
	}
	if len(ws) == 0 {
		return sdk.Workflow{}, sdk.WithStack(sdk.ErrNotFound)
	}
	return ws[0], nil
}

func (dao WorkflowDAO) LoadAll(ctx context.Context, db gorp.SqlExecutor) (sdk.Workflows, error) {
	ctx, end := telemetry.Span(ctx, "workflowDAO.LoadAll")
	defer end()
	t0 := time.Now()

	var workflows []Workflow
	if err := gorpmapping.GetAll(ctx, db, dao.Query(), &workflows, dao.GetLoaders()...); err != nil {
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

	delta := time.Since(t0).Seconds()
	log.Info(ctx, "WorkflowDAO> LoadAll - %d results in %.3f seconds", len(ws), delta)

	return ws, nil
}
