package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/telemetry"
)

type PushSecrets struct {
	ApplicationsSecrets map[int64][]sdk.Variable
	EnvironmentdSecrets map[int64][]sdk.Variable
}

// LoadAllByProjectIDs returns all workflow for given project ids.
func LoadAllNamesByProjectIDs(ctx context.Context, db gorp.SqlExecutor, projectIDs []int64) ([]sdk.WorkflowName, error) {
	query := `
    SELECT workflow.*, project.projectkey
	FROM workflow
	JOIN project ON project.id = workflow.project_id
    WHERE project_id = ANY($1)`

	var result []sdk.WorkflowName // This struct is not registered as a gorpmapping entity so we can't use gorpmapping.Query
	_, err := db.Select(&result, query, pq.Int64Array(projectIDs))
	if err == sql.ErrNoRows {
		return result, nil
	}
	return result, sdk.WithStack(err)
}

// LoadAllByIDs returns all workflows by ids.
func LoadAllByIDs(ctx context.Context, db gorp.SqlExecutor, ids []int64) (sdk.Workflows, error) {
	var dao WorkflowDAO
	dao.Filters.WorkflowIDs = ids
	return dao.LoadAll(ctx, db)
}

// UpdateOptions is option to parse a workflow
type UpdateOptions struct {
	DisableHookManagement bool
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

func LoadByRepo(ctx context.Context, db gorp.SqlExecutor, proj sdk.Project, repo string, opts LoadOptions) (*sdk.Workflow, error) {
	ctx, end := telemetry.Span(ctx, "workflow.Load")
	defer end()

	dao := opts.GetWorkflowDAO()
	dao.Filters.FromRepository = repo
	dao.Filters.ProjectKey = proj.Key

	ws, err := dao.Load(ctx, db)
	if err != nil {
		return nil, err
	}

	if !opts.Minimal {
		if err := CompleteWorkflow(ctx, db, &ws, proj, opts); err != nil {
			return nil, err
		}
	}

	return &ws, nil
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
func UpdateFromRepository(db gorp.SqlExecutor, workflowID int64, fromRepository string) error {
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
	nodes := w.WorkflowData.Array()
	nodesID := make([]int64, len(nodes))

	for i := range nodes {
		nodesID[i] = nodes[i].ID
	}

	mapGroups, err := group.LoadGroupsByNode(db, nodesID)
	if err != nil {
		return sdk.WrapError(err, "cannot load node groups")
	}

	for i := range nodes {
		nodes[i].Groups = mapGroups[nodes[i].ID]
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
	for _, integ := range w.EventIntegrations {
		if err := integration.AddOnWorkflow(db, w.ID, integ.ID); err != nil {
			return sdk.WrapError(err, "cannot add project event integration (%d) on workflow (%d)", integ.ID, w.ID)
		}
	}
	return nil
}

func (w *Workflow) Get() sdk.Workflow {
	wf := w.Workflow
	if wf.ProjectKey == "" {
		wf.ProjectKey = w.ProjectKey
	}
	return wf
}

// LoadAll loads all workflows for a project. All users in a project can list all workflows in a project
func LoadAll(db gorp.SqlExecutor, projectKey string) (sdk.Workflows, error) {
	dao := WorkflowDAO{
		Filters: LoadAllWorkflowsOptionsFilters{
			ProjectKey: projectKey,
		},
	}
	return dao.LoadAll(context.Background(), db)
}

// LoadAllNames loads all workflow names for a project.
func LoadAllNames(db gorp.SqlExecutor, projID int64) (sdk.IDNames, error) {
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
		res[i].Labels, err = LoadLabels(db, res[i].ID)
		if err != nil {
			return res, sdk.WrapError(err, "cannot load labels for workflow %s", res[i].Name)
		}
	}

	return res, nil
}

// Load loads a workflow for a given user (ie. checking permissions)
func Load(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, name string, opts LoadOptions) (*sdk.Workflow, error) {
	ctx, end := telemetry.Span(ctx, "workflow.Load")
	defer end()

	dao := opts.GetWorkflowDAO()
	dao.Filters.ProjectKey = proj.Key
	dao.Filters.WorkflowName = name

	ws, err := dao.Load(ctx, db)
	if err != nil {
		return nil, err
	}

	if !opts.Minimal {
		if err := CompleteWorkflow(ctx, db, &ws, proj, opts); err != nil {
			return nil, err
		}
	}

	return &ws, nil
}

// LoadAndLockByID loads a workflow

// LoadByID loads a workflow
func LoadByID(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, id int64, opts LoadOptions) (*sdk.Workflow, error) {
	dao := opts.GetWorkflowDAO()
	dao.Filters.WorkflowIDs = []int64{id}

	ws, err := dao.Load(ctx, db)
	if err != nil {
		return nil, err
	}

	if !opts.Minimal {
		if err := CompleteWorkflow(ctx, db, &ws, proj, opts); err != nil {
			return nil, err
		}
	}
	return &ws, nil
}

func UpdateMaxRunsByID(db gorp.SqlExecutor, workflowID int64, maxRuns int64) error {
	_, err := db.Exec("UPDATE workflow set max_runs = $1 WHERE id = $2", maxRuns, workflowID)
	return sdk.WithStack(err)
}

// Insert inserts a new workflow
func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, w *sdk.Workflow) error {
	if err := CompleteWorkflow(ctx, db, w, proj, LoadOptions{}); err != nil {
		return err
	}

	if err := CheckValidity(ctx, db, w); err != nil {
		return err
	}

	if w.WorkflowData.Node.Context != nil && w.WorkflowData.Node.Context.ApplicationID != 0 {
		var err error
		if w.WorkflowData.Node.Context.DefaultPayload, err = DefaultPayload(ctx, db, store, proj, w); err != nil {
			log.Warn(ctx, "postWorkflowHandler> Cannot set default payload : %v", err)
		}
	}

	if w.HistoryLength == 0 {
		w.HistoryLength = sdk.DefaultHistoryLength
	}
	w.MaxRuns = maxRuns

	w.LastModified = time.Now()
	if err := db.QueryRow(`INSERT INTO workflow (
		name, description, icon, project_id, history_length, from_repository, purge_tags, workflow_data, metadata, retention_policy, max_runs
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING id`,
		w.Name, w.Description, w.Icon, w.ProjectID, w.HistoryLength, w.FromRepository, w.PurgeTags, w.WorkflowData, w.Metadata, w.RetentionPolicy, w.MaxRuns).Scan(&w.ID); err != nil {
		return sdk.WrapError(err, "Unable to insert workflow %s/%s", w.ProjectKey, w.Name)
	}

	dbw := Workflow{Workflow: *w}
	if err := dbw.PostInsert(db); err != nil {
		return sdk.WrapError(err, "Cannot post insert hook")
	}

	if len(w.Groups) > 0 {
		for i := range w.Groups {
			if w.Groups[i].Group.ID != 0 {
				continue
			}
			g, err := group.LoadByName(ctx, db, w.Groups[i].Group.Name)
			if err != nil {
				return sdk.WrapError(err, "Unable to load group %s", w.Groups[i].Group.Name)
			}
			w.Groups[i].Group = *g
		}
		if err := group.UpsertAllWorkflowGroups(db, w, w.Groups); err != nil {
			return sdk.WrapError(err, "Unable to update workflow")
		}
	} else {
		log.Debug(ctx, "postWorkflowHandler> inherit permissions from project")
		for _, gp := range proj.ProjectGroups {
			if err := group.AddWorkflowGroup(ctx, db, w, gp); err != nil {
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
		if err := hookRegistration(ctx, db, store, proj, w, nil); err != nil {
			return err
		}
	}

	if err := InsertWorkflowData(db, w); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert Workflow Data")
	}

	customVcsNotif := false
	// Insert notifications
	for i := range w.Notifications {
		n := &w.Notifications[i]
		if n.Type == sdk.VCSUserNotification {
			customVcsNotif = true
		}
		if err := InsertNotification(db, w, n); err != nil {
			return sdk.WrapError(err, "Unable to insert update workflow(%d) notification (%#v)", w.ID, n)
		}
	}

	if !customVcsNotif {
		notif := sdk.WorkflowNotification{
			Settings: sdk.UserNotificationSettings{
				Template: &sdk.UserNotificationTemplate{
					Body: sdk.DefaultWorkflowNodeRunReport,
				},
			},
			WorkflowID: w.ID,
			Type:       sdk.VCSUserNotification,
		}
		for _, node := range w.WorkflowData.Array() {
			if node.IsLinkedToRepo(w) {
				notif.SourceNodeRefs = append(notif.SourceNodeRefs, node.Name)
			}
		}
		if len(notif.SourceNodeRefs) > 0 {
			if err := InsertNotification(db, w, &notif); err != nil {
				return sdk.WrapError(err, "Unable to insert VCS workflow(%d) notification (%#v)", w.ID, notif)
			}
		}
	}

	dbWorkflow := Workflow{Workflow: *w}
	if err := dbWorkflow.PostUpdate(db); err != nil {
		return sdk.WrapError(err, "Insert> Unable to create workflow data")
	}

	return nil
}

func RenameNode(ctx context.Context, db gorp.SqlExecutor, w *sdk.Workflow) error {
	ctx, end := telemetry.Span(ctx, "workflow.RenameNode")
	defer end()

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
				p, errPip := pipeline.LoadPipelineByID(ctx, db, nodes[i].Context.PipelineID, true)
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

	nodeNames := make(map[string]struct{}, len(nodes))
	for i := range nodes {
		if _, ok := nodeNames[nodes[i].Name]; ok {
			return sdk.WithStack(sdk.ErrWorkflowNodeNameDuplicate)
		}
		nodeNames[nodes[i].Name] = struct{}{}
	}

	return nil
}

// Update updates a workflow
func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wf *sdk.Workflow, uptOption UpdateOptions) error {
	ctx, end := telemetry.Span(ctx, "workflow.Update")
	defer end()

	if err := CompleteWorkflow(ctx, db, wf, proj, LoadOptions{}); err != nil {
		return err
	}

	if err := CheckValidity(ctx, db, wf); err != nil {
		return err
	}

	if err := DeleteNotifications(db, wf.ID); err != nil {
		return sdk.WrapError(err, "unable to delete all notifications on workflow(%d - %s)", wf.ID, wf.Name)
	}

	if err := integration.DeleteFromWorkflow(db, wf.ID); err != nil {
		return sdk.WrapError(err, "unable to delete all integrations on workflow(%d - %s)", wf.ID, wf.Name)
	}

	// reload workflow to delete the current workflow data
	oldWf, err := LoadByID(ctx, db, store, proj, wf.ID, LoadOptions{Minimal: true})
	if err != nil {
		return sdk.WrapError(err, "Unable to load existing workflow with proj:%s ID:%d", proj.Key, wf.ID)
	}

	// Keep MaxRun
	wf.MaxRuns = oldWf.MaxRuns

	if err := DeleteWorkflowData(db, *oldWf); err != nil {
		return sdk.WrapError(err, "unable to delete from old workflow data(%d - %s)", wf.ID, wf.Name)
	}

	// Delete all node ID
	wf.ResetIDs()

	filteredPurgeTags := []string{}
	for _, t := range wf.PurgeTags {
		if t != "" {
			filteredPurgeTags = append(filteredPurgeTags, t)
		}
	}
	wf.PurgeTags = filteredPurgeTags
	if wf.WorkflowData.Node.Context != nil && wf.WorkflowData.Node.Context.ApplicationID != 0 {
		var err error
		if wf.WorkflowData.Node.Context.DefaultPayload, err = DefaultPayload(ctx, db, store, proj, wf); err != nil {
			log.Warn(ctx, "workflow.Update> Cannot set default payload : %v", err)
		}
	}

	if !uptOption.DisableHookManagement {
		if err := hookRegistration(ctx, db, store, proj, wf, oldWf); err != nil {
			return err
		}
		if oldWf != nil {
			hookToDelete := computeHookToDelete(wf, oldWf)
			if err := hookUnregistration(ctx, db, store, proj, hookToDelete); err != nil {
				return err
			}
		}
	} else {
		for i := range wf.WorkflowData.Node.Hooks {
			h := &wf.WorkflowData.Node.Hooks[i]
			if h.UUID == "" {
				h.UUID = sdk.UUID()
			}
		}
	}

	if err := InsertWorkflowData(db, wf); err != nil {
		return sdk.WrapError(err, "Update> Unable to insert workflow data")
	}

	// Insert notifications
	for i := range wf.Notifications {
		n := &wf.Notifications[i]
		if err := InsertNotification(db, wf, n); err != nil {
			return sdk.WrapError(err, "Unable to update workflow(%d) notification (%#v)", wf.ID, n)
		}
	}

	wf.LastModified = time.Now()
	dbw := Workflow{Workflow: *wf}
	if _, err := db.UpdateColumns(func(c *gorp.ColumnMap) bool { return c.ColumnName != "project_key" }, &dbw); err != nil {
		return sdk.WrapError(err, "Unable to update workflow")
	}
	*wf = dbw.Get()

	return nil
}

// MarkAsDelete marks a workflow to be deleted
func MarkAsDelete(ctx context.Context, db gorpmapper.SqlExecutorWithTx, cache cache.Store, proj sdk.Project, wkf *sdk.Workflow) error {
	// Remove references of dependencies to be able to delete them before workflow deletion
	nodes := wkf.WorkflowData.Array()
	for _, n := range nodes {
		n.Context.ApplicationID = 0
		n.Context.PipelineID = 0
		n.Context.EnvironmentID = 0
		n.Context.ProjectIntegrationID = 0
	}
	wkf.ToDelete = true
	return Update(ctx, db, cache, proj, wkf, UpdateOptions{DisableHookManagement: true})
}

// Delete workflow
func Delete(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, w *sdk.Workflow) error {
	// Delete all hooks
	if err := hookUnregistration(ctx, db, store, proj, w.WorkflowData.GetHooksMapRef()); err != nil {
		return sdk.WrapError(err, "unable to delete hooks from workflow")
	}

	if err := DeleteWorkflowData(db, *w); err != nil {
		return sdk.WrapError(err, "unable to delete workflow data")
	}

	query := `DELETE FROM w_node_trigger
					WHERE parent_node_id IN
					(SELECT id FROM w_node WHERE workflow_id = $1)
		`
	if _, err := db.Exec(query, w.ID); err != nil {
		return sdk.WrapError(err, "unable to delete node trigger")
	}

	//Delete workflow
	dbw := Workflow{Workflow: *w}
	if _, err := db.Delete(&dbw); err != nil {
		return sdk.WrapError(err, "unable to delete workflow")
	}

	return nil
}

func CompleteWorkflow(ctx context.Context, db gorp.SqlExecutor, w *sdk.Workflow, proj sdk.Project, opts LoadOptions) error {
	ctx, end := telemetry.Span(ctx, "workflow.CompleteWorkflow")
	defer end()

	w.InitMaps()
	w.AssignEmptyType()

	nodesArray := w.WorkflowData.Array()

	if err := checkEventIntegration(proj, w); err != nil {
		return err
	}

	for i := range nodesArray {
		n := nodesArray[i]
		if n.Context == nil {
			continue
		}

		if err := checkPipeline(ctx, db, proj, w, n, opts); err != nil {
			return err
		}
		if err := checkApplication(db, proj, w, n); err != nil {
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

	w.Normalize()

	return nil
}

// CheckValidity checks workflow validity
func CheckValidity(ctx context.Context, db gorp.SqlExecutor, w *sdk.Workflow) error {
	//Check project is not empty
	if w.ProjectKey == "" {
		return sdk.NewErrorFrom(sdk.ErrWorkflowInvalid, "invalid project key")
	}

	if w.Icon != "" {
		if !strings.HasPrefix(w.Icon, sdk.IconFormat) {
			return sdk.WithStack(sdk.ErrIconBadFormat)
		}
		if len(w.Icon) > sdk.MaxIconSize {
			return sdk.WithStack(sdk.ErrIconBadSize)
		}
	}

	//Check workflow name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(w.Name) {
		return sdk.NewErrorFrom(sdk.ErrWorkflowInvalid, "workflow name should match pattern %s", sdk.NamePattern)
	}

	//Check refs
	for _, j := range w.WorkflowData.Joins {
		if len(j.JoinContext) == 0 {
			return sdk.NewErrorFrom(sdk.ErrWorkflowInvalid, "source node references is mandatory")
		}
	}

	if w.WorkflowData.Node.Context != nil && w.WorkflowData.Node.Context.DefaultPayload != nil {
		defaultPayload, err := w.WorkflowData.Node.Context.DefaultPayloadToMap()
		if err != nil {
			return sdk.NewErrorFrom(err, "cannot transform default payload to map")
		}
		for payloadKey := range defaultPayload {
			if strings.HasPrefix(payloadKey, "cds.") {
				return sdk.NewErrorFrom(sdk.ErrInvalidPayloadVariable, "cannot have key %s in default payload", payloadKey)
			}
		}
	}

	// Fill empty node type
	if err := w.ValidateType(); err != nil {
		return err
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
			if _, ok := w.HookModels[h.HookModelID]; !ok {
				hmDB, err := LoadHookModelByID(db, h.HookModelID)
				if err != nil {
					return err
				}
				w.HookModels[h.HookModelID] = *hmDB
			}
		} else {
			hm, err := LoadHookModelByName(db, h.HookModelName)
			if err != nil {
				return err
			}
			w.HookModels[hm.ID] = *hm
			h.HookModelID = hm.ID
		}
		if h.HookModelName == sdk.RepositoryWebHookModelName && (n.Context == nil || n.Context.ApplicationID == 0) {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to find application for the repository webhook: %s/%s", w.Name, n.Name)
		}

		// Add missing default value for hook
		model := w.HookModels[h.HookModelID]
		h.HookModelName = model.Name
		for k := range model.DefaultConfig {
			if _, ok := h.Config[k]; !ok {
				h.Config[k] = model.DefaultConfig[k]
			}
		}

		// Check that given config is valid according hook model
		for k, d := range model.DefaultConfig {
			if !d.Configurable && h.Config[k].Value != d.Value {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given hook config, '%s' is not configurable. Value: %+v in model %+v", k, h.Config[k].Value, model)
			}
			if len(d.MultipleChoiceList) > 0 {
				var found bool
				for i := range d.MultipleChoiceList {
					if h.Config[k].Value == d.MultipleChoiceList[i] {
						found = true
						break
					}
				}
				if !found {
					return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given value for hook config '%s', given value not in choices list", k)
				}
			}
			v := h.Config[k]
			v.Configurable = d.Configurable
			h.Config[k] = v
		}
		// Check hooks duplication
		for j := range n.Hooks {
			h2 := n.Hooks[j]
			if i != j && h.Ref() == h2.Ref() {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid workflow: duplicate hook %s", model.Name)
			}
		}
	}

	return nil
}

// CheckProjectIntegration checks CheckProjectIntegration data
func checkProjectIntegration(proj sdk.Project, w *sdk.Workflow, n *sdk.Node) error {
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
				return sdk.WrapError(sdk.ErrNotFound, "integration %d not found", n.Context.ProjectIntegrationID)
			}
			w.ProjectIntegrations[n.Context.ProjectIntegrationID] = pp
		}
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
			return sdk.WithData(sdk.ErrIntegrationtNotFound, n.Context.ProjectIntegrationName)
		}
		w.ProjectIntegrations[ppProj.ID] = ppProj
		n.Context.ProjectIntegrationID = ppProj.ID
	}
	return nil
}

// checkEventIntegration checks event integration data
func checkEventIntegration(proj sdk.Project, w *sdk.Workflow) error {
	for i := range w.EventIntegrations {
		eventIntegration := w.EventIntegrations[i]
		found := false
		for _, projInt := range proj.Integrations {
			if eventIntegration.Name == projInt.Name {
				eventIntegration.ID = projInt.ID
				w.EventIntegrations[i] = eventIntegration
				found = true
				break
			}
		}
		if !found {
			return sdk.WithData(sdk.ErrIntegrationtNotFound, eventIntegration.Name)
		}
	}

	return nil
}

// CheckEnvironment checks environment data
func checkEnvironment(db gorp.SqlExecutor, proj sdk.Project, w *sdk.Workflow, n *sdk.Node) error {
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
func checkApplication(db gorp.SqlExecutor, proj sdk.Project, w *sdk.Workflow, n *sdk.Node) error {
	if n.Context.ApplicationID != 0 {
		app, ok := w.Applications[n.Context.ApplicationID]
		if !ok {
			appDB, errA := application.LoadByID(db, n.Context.ApplicationID, application.LoadOptions.WithDeploymentStrategies, application.LoadOptions.WithVariables, application.LoadOptions.WithKeys)
			if errA != nil {
				return errA
			}
			app = *appDB
			if app.ProjectKey != proj.Key {
				return sdk.NewErrorFrom(sdk.ErrResourceNotInProject, "can not found a application with id %d", n.Context.ApplicationID)
			}

			w.Applications[n.Context.ApplicationID] = app
		}
		return nil
	}
	if n.Context.ApplicationName != "" {
		appDB, err := application.LoadByName(db, proj.Key, n.Context.ApplicationName, application.LoadOptions.WithDeploymentStrategies, application.LoadOptions.WithVariables, application.LoadOptions.WithKeys)
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.NewErrorFrom(sdk.WithData(sdk.ErrNotFound, n.Context.ApplicationName), "cannot find application with name: %s", n.Context.ApplicationName)
			}
			return sdk.WrapError(err, "unable to load application %s", n.Context.ApplicationName)
		}
		w.Applications[appDB.ID] = *appDB
		n.Context.ApplicationID = appDB.ID
	}
	return nil
}

// CheckPipeline checks pipeline data
func checkPipeline(ctx context.Context, db gorp.SqlExecutor, proj sdk.Project, w *sdk.Workflow, n *sdk.Node, opts LoadOptions) error {
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
		return nil
	}
	if n.Context.PipelineName != "" {
		pipDB, err := pipeline.LoadPipeline(ctx, db, proj.Key, n.Context.PipelineName, opts.DeepPipeline)
		if err != nil {
			return sdk.WrapError(err, "unable to load pipeline %s", n.Context.PipelineName)
		}
		w.Pipelines[pipDB.ID] = *pipDB
		n.Context.PipelineID = pipDB.ID
	}
	return nil
}

// Push push a workflow from cds files
func Push(ctx context.Context, db *gorp.DbMap, store cache.Store, proj *sdk.Project, data exportentities.WorkflowComponents,
	opts *PushOption, u sdk.Identifiable, decryptFunc keys.DecryptFunc) ([]sdk.Message, *sdk.Workflow, *sdk.Workflow, *PushSecrets, error) {
	ctx, end := telemetry.Span(ctx, "workflow.Push")
	defer end()
	if data.Workflow == nil {
		return nil, nil, nil, nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given workflow components, missing workflow file")
	}

	var err error
	var workflowExists bool
	var oldWf *sdk.Workflow

	if opts != nil && opts.OldWorkflow.ID > 0 {
		oldWf = &opts.OldWorkflow
	} else {
		// load the workflow from database if exists
		workflowExists, err = Exists(db, proj.Key, data.Workflow.GetName())
		if err != nil {
			return nil, nil, nil, nil, sdk.WrapError(err, "cannot check if workflow exists")
		}
		if workflowExists {
			oldWf, err = Load(ctx, db, store, *proj, data.Workflow.GetName(), LoadOptions{WithIcon: true})
			if err != nil {
				return nil, nil, nil, nil, sdk.WrapError(err, "unable to load existing workflow")
			}
		}
	}

	// if a old workflow as code exists, we want to check if the new workflow is also as code on the same repository
	if oldWf != nil && oldWf.FromRepository != "" && (opts == nil || opts.FromRepository != oldWf.FromRepository) {
		return nil, nil, nil, nil, sdk.WithStack(sdk.ErrWorkflowAlreadyAsCode)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, nil, nil, nil, sdk.WrapError(err, "unable to start tx")
	}
	defer tx.Rollback() // nolint

	var allMsg []sdk.Message
	allSecrets := PushSecrets{
		ApplicationsSecrets: make(map[int64][]sdk.Variable),
		EnvironmentdSecrets: make(map[int64][]sdk.Variable),
	}
	for _, app := range data.Applications {
		var fromRepo string
		if opts != nil {
			fromRepo = opts.FromRepository
		}
		appDB, appSecrets, msgList, err := application.ParseAndImport(ctx, tx, store, *proj, &app, application.ImportOptions{Force: true, FromRepository: fromRepo}, decryptFunc, u)
		allMsg = append(allMsg, msgList...)
		if err != nil {
			return allMsg, nil, nil, nil, sdk.ErrorWithFallback(err, sdk.ErrWrongRequest, "unable to import application %s/%s", proj.Key, app.Name)
		}
		proj.SetApplication(*appDB)
		allSecrets.ApplicationsSecrets[appDB.ID] = appSecrets
	}

	for _, env := range data.Environments {
		var fromRepo string
		if opts != nil {
			fromRepo = opts.FromRepository
		}
		envDB, envsSecrets, msgList, err := environment.ParseAndImport(ctx, tx, *proj, env, environment.ImportOptions{Force: true, FromRepository: fromRepo}, decryptFunc, u)
		allMsg = append(allMsg, msgList...)
		if err != nil {
			return allMsg, nil, nil, nil, sdk.ErrorWithFallback(err, sdk.ErrWrongRequest, "unable to import environment %s/%s", proj.Key, env.Name)
		}
		proj.SetEnvironment(*envDB)
		allSecrets.EnvironmentdSecrets[envDB.ID] = envsSecrets
	}

	for _, pip := range data.Pipelines {
		var fromRepo string
		if opts != nil {
			fromRepo = opts.FromRepository
		}
		pipDB, msgList, err := pipeline.ParseAndImport(ctx, tx, store, *proj, &pip, u, pipeline.ImportOptions{Force: true, FromRepository: fromRepo})
		allMsg = append(allMsg, msgList...)
		if err != nil {
			return allMsg, nil, nil, nil, sdk.ErrorWithFallback(err, sdk.ErrWrongRequest, "unable to import pipeline %s/%s", proj.Key, pip.Name)
		}
		proj.SetPipeline(*pipDB)
	}

	isDefaultBranch := true
	if opts != nil {
		isDefaultBranch = opts.IsDefaultBranch
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

	wf, msgList, err := ParseAndImport(ctx, tx, store, *proj, oldWf, data.Workflow, u, importOptions)
	allMsg = append(allMsg, msgList...)
	if err != nil {
		return allMsg, nil, nil, nil, sdk.WrapError(err, "unable to import workflow %s", data.Workflow.GetName())
	}

	// If the workflow is "as-code", it should always be linked to a git repository
	if opts != nil && opts.FromRepository != "" {
		if wf.WorkflowData.Node.Context.ApplicationID == 0 {
			return nil, nil, nil, nil, sdk.WithStack(sdk.ErrApplicationMandatoryOnWorkflowAsCode)
		}
		app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]
		if app.VCSServer == "" || app.RepositoryFullname == "" {
			return nil, nil, nil, nil, sdk.WithStack(sdk.ErrApplicationMandatoryOnWorkflowAsCode)
		}
	}

	// come from run
	if opts != nil && opts.HookUUID != "" {
		// Load secrets from application and environment non-ascode
		for id, app := range wf.Applications {
			if app.FromRepository != "" {
				continue
			}
			appSecrets, err := LoadApplicationSecrets(db, id)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			allSecrets.ApplicationsSecrets[id] = appSecrets
		}
		for id, env := range wf.Environments {
			if env.FromRepository != "" {
				continue
			}
			secrets, err := LoadEnvironmentSecrets(db, id)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			allSecrets.EnvironmentdSecrets[id] = secrets
		}
	}

	if wf.WorkflowData.Node.Context.ApplicationID != 0 {
		app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]
		if err := application.Update(tx, &app); err != nil {
			return nil, nil, nil, nil, sdk.WrapError(err, "Unable to update application vcs datas")
		}
		wf.Applications[wf.WorkflowData.Node.Context.ApplicationID] = app
	}

	if !isDefaultBranch {
		_ = tx.Rollback()
		log.Debug(ctx, "workflow %s rollbacked because it's not coming from the default branch", wf.Name)
	} else {
		if err := tx.Commit(); err != nil {
			return nil, nil, nil, nil, sdk.WithStack(err)
		}

		log.Debug(ctx, "workflow %s updated", wf.Name)
	}

	return allMsg, wf, oldWf, &allSecrets, nil
}

// UpdateFavorite add or delete workflow from user favorites
func UpdateFavorite(db gorp.SqlExecutor, workflowID int64, u string, add bool) error {
	var query string
	if add {
		query = "INSERT INTO workflow_favorite (authentified_user_id, workflow_id) VALUES ($1, $2)"
	} else {
		query = "DELETE FROM workflow_favorite WHERE authentified_user_id = $1 AND workflow_id = $2"
	}

	_, err := db.Exec(query, u, workflowID)
	return sdk.WithStack(err)
}

// IsDeploymentIntegrationUsed checks if a deployment integration is used on any workflow
func IsDeploymentIntegrationUsed(db gorp.SqlExecutor, projectID int64, appID int64, pfName string) (bool, error) {
	query := `
	SELECT count(1)
	FROM w_node_context
	JOIN project_integration ON project_integration.id = w_node_context.project_integration_id
	WHERE w_node_context.application_id = $2
	AND project_integration.project_id = $1
	AND project_integration.name = $3
	`

	nb, err := db.SelectInt(query, projectID, appID, pfName)
	if err != nil {
		return false, sdk.WithStack(err)
	}

	return nb > 0, nil
}
