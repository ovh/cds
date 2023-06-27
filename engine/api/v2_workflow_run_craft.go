package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

type WorkflowRunEntityFinder struct {
	vcsServerCache         map[string]sdk.VCSProject
	repoCache              map[string]sdk.ProjectRepository
	repoDefaultBranchCache map[string]string
	actionsCache           map[string]sdk.V2Action
	workerModelCache       map[string]sdk.V2WorkerModel
	run                    sdk.V2WorkflowRun
	runVcsServer           sdk.VCSProject
	runRepo                sdk.ProjectRepository
	userName               string
}

func (api *API) V2WorkflowRunCraft(ctx context.Context, tick time.Duration) error {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case id := <-api.workflowRunCraftChan:
			api.GoRoutines.Exec(
				ctx,
				"V2WorkflowRunCraft-"+id,
				func(ctx context.Context) {
					if err := api.craftWorkflowRunV2(ctx, id); err != nil {
						log.Error(ctx, "V2WorkflowRunCraft> error on workflow run %s: %v", id, err)
					}
				},
			)
		case <-ticker.C:
			ids, err := workflow_v2.LoadCratingWorkflowRunIDs(api.mustDB())
			if err != nil {
				log.Error(ctx, "V2WorkflowRunCraft> unable to start tx: %v", err)
				continue
			}
			for _, id := range ids {
				api.GoRoutines.Exec(
					ctx,
					"V2WorkflowRunCraft-"+id,
					func(ctx context.Context) {
						if err := api.craftWorkflowRunV2(ctx, id); err != nil {
							log.Error(ctx, "V2WorkflowRunCraft> error on workflow run %s: %v", id, err)
						}
					},
				)
			}
		}
	}
}

func (api *API) craftWorkflowRunV2(ctx context.Context, id string) error {
	ctx, next := telemetry.Span(ctx, "api.craftWorkflowRunV2")
	defer next()

	_, next = telemetry.Span(ctx, "api.craftWorkflowRunV2.lock")
	lockKey := cache.Key("api:craftWorkflowRunV2", id)
	b, err := api.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		next()
		return err
	}
	if !b {
		log.Debug(ctx, "api.craftWorkflowRunV2> run %d is locked in cache", id)
		next()
		return nil
	}
	next()
	defer func() {
		_ = api.Cache.Unlock(lockKey)
	}()

	run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), id)
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil
	}
	if err != nil {
		return sdk.WrapError(err, "unable to load workflow run %s", id)
	}

	telemetry.Current(ctx).AddAttributes(
		trace.StringAttribute(telemetry.TagProjectKey, run.ProjectKey),
		trace.StringAttribute(telemetry.TagWorkflow, run.WorkflowName),
		trace.StringAttribute(telemetry.TagWorkflowRunNumber, strconv.FormatInt(run.RunNumber, 10)))

	if run.Status != sdk.StatusCrafting {
		return nil
	}

	vcsServer, err := vcs.LoadVCSByID(ctx, api.mustDB(), run.ProjectKey, run.VCSServerID)
	if err != nil {
		return err
	}
	repo, err := repository.LoadRepositoryByVCSAndID(ctx, api.mustDB(), vcsServer.ID, run.RepositoryID)
	if err != nil {
		return err
	}

	u, err := user.LoadByID(ctx, api.mustDB(), run.UserID)
	if err != nil {
		return err
	}

	// Build run context
	runContext := buildRunContext(*run, *vcsServer, *repo, *u)
	run.Contexts = runContext

	wref := WorkflowRunEntityFinder{
		run:                    *run,
		runRepo:                *repo,
		runVcsServer:           *vcsServer,
		actionsCache:           make(map[string]sdk.V2Action),
		workerModelCache:       make(map[string]sdk.V2WorkerModel),
		repoCache:              make(map[string]sdk.ProjectRepository),
		vcsServerCache:         make(map[string]sdk.VCSProject),
		repoDefaultBranchCache: make(map[string]string),
		userName:               u.Username,
	}

	// Retrieve all deps
	for jobID := range run.WorkflowData.Workflow.Jobs {
		j := run.WorkflowData.Workflow.Jobs[jobID]

		// Get worker model
		completeName, msg, err := wref.searchEntity(ctx, api.mustDB(), api.Cache, j.WorkerModel, sdk.EntityTypeWorkerModel)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return stopRun(ctx, api.mustDB(), run, &sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       "unable to trigger workflow. Please contact an administrator",
			})
		}
		if msg != nil {
			return stopRun(ctx, api.mustDB(), run, msg)
		}
		j.WorkerModel = completeName

		// Get actions and sub actions
		msg, err = searchActions(ctx, api.mustDB(), api.Cache, &wref, j.Steps)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return stopRun(ctx, api.mustDB(), run, &sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("unable to retrieve job[%s] definition. Please contact an administrator", jobID),
			})
		}
		if msg != nil {
			return stopRun(ctx, api.mustDB(), run, msg)
		}

	}

	for k, v := range wref.actionsCache {
		run.WorkflowData.Actions = make(map[string]sdk.V2Action)
		run.WorkflowData.Actions[k] = v
	}
	for k, v := range wref.workerModelCache {
		run.WorkflowData.WorkerModels = make(map[string]sdk.V2WorkerModel)
		run.WorkflowData.WorkerModels[k] = v
	}

	tx, err := api.mustDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	run.Status = sdk.StatusBuilding
	if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
		return err
	}

	enqueueRequest := sdk.V2WorkflowRunEnqueue{
		RunID:  run.ID,
		UserID: run.UserID,
	}

	select {
	case api.workflowRunTriggerChan <- enqueueRequest:
		log.Debug(ctx, "workflow run %s %d trigger in chan", run.WorkflowName, run.RunNumber)
	default:
		if err := api.Cache.Enqueue(workflow_v2.WorkflowEngineKey, enqueueRequest); err != nil {
			return err
		}
	}
	return sdk.WithStack(tx.Commit())
}

// Return the complete path of the entity
func (wref *WorkflowRunEntityFinder) searchEntity(ctx context.Context, db *gorp.DbMap, store cache.Store, name string, entityType string) (string, *sdk.V2WorkflowRunInfo, error) {
	ctx, end := telemetry.Span(ctx, "WorkflowRunEntityFinder.searchEntity", trace.StringAttribute("entity-type", entityType), trace.StringAttribute("entity-name", name))
	defer end()

	var branch, entityName, repoName, vcsName, projKey string

	if name == "" {
		return "", &sdk.V2WorkflowRunInfo{WorkflowRunID: wref.run.ID, Level: sdk.WorkflowRunInfoLevelError, Message: entityType + " cannot be empty"}, nil
	}

	// Get branch if present
	splitBranch := strings.Split(name, "@")
	if len(splitBranch) == 2 {
		branch = splitBranch[1]
	}
	entityFullPath := splitBranch[0]

	entityPathSplit := strings.Split(entityFullPath, "/")
	embeddedEntity := false
	switch len(entityPathSplit) {
	case 1:
		entityName = entityFullPath
		embeddedEntity = true
	case 2:
		return "", &sdk.V2WorkflowRunInfo{WorkflowRunID: wref.run.ID, Level: sdk.WorkflowRunInfoLevelError, Message: fmt.Sprintf("invalid workflow: unable to get repository from %s", entityFullPath)}, nil
	case 3:
		repoName = fmt.Sprintf("%s/%s", entityPathSplit[0], entityPathSplit[1])
		entityName = entityPathSplit[2]
	case 4:
		vcsName = entityPathSplit[0]
		repoName = fmt.Sprintf("%s/%s", entityPathSplit[1], entityPathSplit[2])
		entityName = entityPathSplit[3]
	case 5:
		projKey = entityPathSplit[0]
		vcsName = entityPathSplit[1]
		repoName = fmt.Sprintf("%s/%s", entityPathSplit[2], entityPathSplit[3])
		entityName = entityPathSplit[4]
	default:
		return "", &sdk.V2WorkflowRunInfo{WorkflowRunID: wref.run.ID, Level: sdk.WorkflowRunInfoLevelError, Message: fmt.Sprintf("unable to parse the %s: %s", entityType, name)}, nil
	}

	var entityVCS sdk.VCSProject
	var entityRepo sdk.ProjectRepository

	// If no project key in path, get it from workflow run
	if projKey == "" || projKey == wref.run.ProjectKey {
		projKey = wref.run.ProjectKey
	} else {
		// Verify project read permission
		can, err := rbac.HasRoleOnProjectAndUserID(ctx, db, sdk.ProjectRoleRead, wref.run.UserID, projKey)
		if err != nil {
			return "", nil, err
		}
		if !can {
			return "", &sdk.V2WorkflowRunInfo{WorkflowRunID: wref.run.ID, Level: sdk.WorkflowRunInfoLevelError, Message: fmt.Sprintf("user %s do not have the permission to access %s", wref.userName, name)}, nil
		}
	}

	// If no vcs in path, get it from workflow run
	if vcsName == "" || vcsName == wref.runVcsServer.Name {
		vcsName = wref.runVcsServer.Name
		entityVCS = wref.runVcsServer
	} else {
		vcsFromCache, has := wref.vcsServerCache[vcsName]
		if has {
			entityVCS = vcsFromCache
		} else {
			vcsDB, err := vcs.LoadVCSByName(ctx, db, projKey, vcsName)
			if err != nil {
				return "", nil, err
			}
			entityVCS = *vcsDB
			wref.vcsServerCache[vcsName] = *vcsDB
		}
	}
	// If no repo in path, get it from workflow run
	if repoName == "" || (vcsName == wref.runVcsServer.Name && repoName == wref.runRepo.Name) {
		repoName = wref.runRepo.Name
		entityRepo = wref.runRepo
	} else {
		entityFromCache, has := wref.repoCache[vcsName+"/"+repoName]
		if has {
			entityRepo = entityFromCache
		} else {
			repoDB, err := repository.LoadRepositoryByName(ctx, db, entityVCS.ID, repoName)
			if err != nil {
				return "", nil, err
			}
			entityRepo = *repoDB
			wref.repoCache[vcsName+"/"+repoName] = *repoDB
		}
	}
	if branch == "" {
		if embeddedEntity || (projKey == wref.run.ProjectKey && entityVCS.ID == wref.runVcsServer.ID && entityRepo.ID == wref.runRepo.ID) {
			// Get current git.branch parameters
			branch = wref.run.WorkflowRef
		} else {
			defaultCache, has := wref.repoDefaultBranchCache[entityVCS.Name+"/"+entityRepo.Name]
			if has {
				branch = defaultCache
			} else {
				// Get default branch
				tx, err := db.Begin()
				if err != nil {
					return "", nil, sdk.WithStack(err)
				}
				client, err := repositoriesmanager.AuthorizedClient(ctx, tx, store, projKey, entityVCS.Name)
				if err != nil {
					return "", nil, err
				}
				b, err := client.Branch(ctx, entityRepo.Name, sdk.VCSBranchFilters{Default: true})
				if err != nil {
					return "", nil, err
				}
				if err := tx.Commit(); err != nil {
					return "", nil, sdk.WithStack(err)
				}
				branch = b.DisplayID
				wref.repoDefaultBranchCache[entityVCS.Name+"/"+entityRepo.Name] = branch
			}
		}
	}

	completePath := fmt.Sprintf("%s/%s/%s/%s", projKey, vcsName, repoName, entityName)
	if branch != "" {
		completePath += "@" + branch
	}

	switch entityType {
	case sdk.EntityTypeAction:
		if _, has := wref.actionsCache[completePath]; has {
			return completePath, nil, nil
		}
	case sdk.EntityTypeWorkerModel:
		if _, has := wref.workerModelCache[completePath]; has {
			return completePath, nil, nil
		}
	}

	var entityDB *sdk.Entity
	var err error
	if projKey != wref.run.ProjectKey || entityVCS.Name != wref.runVcsServer.Name || entityRepo.Name != wref.runRepo.Name || branch != wref.run.WorkflowRef {
		entityDB, err = entity.LoadByBranchTypeName(ctx, db, entityRepo.ID, branch, entityType, entityName)
	} else {
		entityDB, err = entity.LoadByBranchTypeNameCommit(ctx, db, entityRepo.ID, branch, entityType, entityName, wref.run.WorkflowSha)
	}
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return "", &sdk.V2WorkflowRunInfo{WorkflowRunID: wref.run.ID, Level: sdk.WorkflowRunInfoLevelWarning, Message: fmt.Sprintf("obsolete workflow dependency used: %s", name)}, nil
		}
		return "", nil, err
	}

	switch entityType {
	case sdk.EntityTypeAction:
		var act sdk.V2Action
		if err := yaml.Unmarshal([]byte(entityDB.Data), &act); err != nil {
			return "", nil, err
		}
		wref.actionsCache[completePath] = act
	case sdk.EntityTypeWorkerModel:
		var wm sdk.V2WorkerModel
		if err := yaml.Unmarshal([]byte(entityDB.Data), &wm); err != nil {
			return "", nil, err
		}
		wref.workerModelCache[completePath] = wm
	}
	return completePath, nil, nil
}

func searchActions(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, steps []sdk.ActionStep) (*sdk.V2WorkflowRunInfo, error) {
	ctx, end := telemetry.Span(ctx, "searchActions")
	defer end()
	for i := range steps {
		step := &steps[i]
		if step.Uses == "" || !strings.HasPrefix(step.Uses, "actions/") {
			continue
		}
		actionName := strings.TrimPrefix(step.Uses, "actions/")
		completeName, msg, err := wref.searchEntity(ctx, db, store, actionName, sdk.EntityTypeAction)
		if msg != nil || err != nil {
			return msg, err
		}
		step.Uses = "actions/" + completeName
		act := wref.actionsCache[completeName]
		msg, err = searchActions(ctx, db, store, wref, act.Runs.Steps)
		if msg != nil || err != nil {
			return msg, err
		}
	}
	return nil, nil
}

func stopRun(ctx context.Context, db *gorp.DbMap, run *sdk.V2WorkflowRun, msg *sdk.V2WorkflowRunInfo) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	if err := workflow_v2.InsertRunInfo(ctx, tx, msg); err != nil {
		return err
	}
	if msg.Level == sdk.WorkflowRunInfoLevelWarning {
		run.Status = sdk.StatusSkipped
	} else {
		run.Status = sdk.StatusFail
	}
	if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
		return err
	}

	return sdk.WithStack(tx.Commit())
}

// TODO Manage run attempt + git context + vars context
func buildRunContext(wr sdk.V2WorkflowRun, vcsServer sdk.VCSProject, repo sdk.ProjectRepository, u sdk.AuthentifiedUser) sdk.WorkflowRunContext {
	var runContext sdk.WorkflowRunContext

	cdsContext := sdk.CDSContext{
		ProjectKey:         wr.ProjectKey,
		RunID:              wr.ID,
		RunNumber:          wr.RunNumber,
		RunAttempt:         0, // TODO manage run attempt
		Workflow:           wr.WorkflowName,
		WorkflowRef:        wr.WorkflowRef,
		WorkflowSha:        wr.WorkflowSha,
		WorkflowVCSServer:  vcsServer.Name,
		WorkflowRepository: repo.Name,
		Event:              nil,
		TriggeringActor:    u.Username,
	}

	// TODO manage git context
	var gitContext sdk.GitContext
	if wr.WorkflowData.Workflow.Repository.Name != "" {
		gitContext = sdk.GitContext{
			Hash:       "",
			HashShort:  "",
			Repository: "",
			Branch:     "",
			Tag:        "",
			Author:     "",
			Message:    "",
			URL:        "",
			Server:     "",
			EventName:  "",
			Connection: "",
			SSHKey:     "",
			PGPKey:     "",
			HttpUser:   "",
		}
	}

	runContext.CDS = cdsContext
	runContext.Git = gitContext
	runContext.Vars = nil
	return runContext
}
