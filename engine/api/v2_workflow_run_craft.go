package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

type WorkflowRunEntityFinder struct {
	project                sdk.Project
	vcsServerCache         map[string]sdk.VCSProject
	repoCache              map[string]sdk.ProjectRepository
	repoDefaultBranchCache map[string]string
	actionsCache           map[string]sdk.V2Action
	localActionsCache      map[string]sdk.V2Action
	localWorkerModelCache  map[string]sdk.V2WorkerModel
	workerModelCache       map[string]sdk.V2WorkerModel
	plugins                map[string]sdk.GRPCPlugin
	run                    sdk.V2WorkflowRun
	runVcsServer           sdk.VCSProject
	runRepo                sdk.ProjectRepository
	userName               string
}

func (api *API) V2WorkflowRunCraft(ctx context.Context, tick time.Duration) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
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
	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, id)

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
	ctx = context.WithValue(ctx, cdslog.Project, run.ProjectKey)
	ctx = context.WithValue(ctx, cdslog.Workflow, run.WorkflowName)

	if run.Status != sdk.StatusCrafting {
		return nil
	}

	vcsServer, err := vcs.LoadVCSByIDAndProjectKey(ctx, api.mustDB(), run.ProjectKey, run.VCSServerID)
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

	p, err := project.Load(ctx, api.mustDB(), run.ProjectKey, project.LoadOptions.WithVariables)
	if err != nil {
		return err
	}

	// Build run context
	runContext, err := buildRunContext(ctx, api.mustDB(), api.Cache, *p, *run, *vcsServer, *repo, *u)
	if err != nil {
		return stopRun(ctx, api.mustDB(), api.Cache, run, *u, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       fmt.Sprintf("%v", err),
		})
	}
	run.Contexts = *runContext

	wref := WorkflowRunEntityFinder{
		project:                *p,
		run:                    *run,
		runRepo:                *repo,
		runVcsServer:           *vcsServer,
		actionsCache:           make(map[string]sdk.V2Action),
		localActionsCache:      make(map[string]sdk.V2Action),
		workerModelCache:       make(map[string]sdk.V2WorkerModel),
		localWorkerModelCache:  make(map[string]sdk.V2WorkerModel),
		repoCache:              make(map[string]sdk.ProjectRepository),
		vcsServerCache:         make(map[string]sdk.VCSProject),
		repoDefaultBranchCache: make(map[string]string),
		plugins:                make(map[string]sdk.GRPCPlugin),
		userName:               u.Username,
	}

	plugins, err := plugin.LoadAllByType(ctx, api.mustDB(), sdk.GRPCPluginAction)
	if err != nil {
		return err
	}
	for _, p := range plugins {
		wref.plugins[p.Name] = p
	}

	// Retrieve all deps
	for jobID := range run.WorkflowData.Workflow.Jobs {
		j := run.WorkflowData.Workflow.Jobs[jobID]

		// Get actions and sub actions
		msg, err := searchActions(ctx, api.mustDB(), api.Cache, &wref, j.Steps)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return stopRun(ctx, api.mustDB(), api.Cache, run, *u, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("unable to retrieve job[%s] definition. Please contact an administrator", jobID),
			})
		}
		if msg != nil {
			return stopRun(ctx, api.mustDB(), api.Cache, run, *u, *msg)
		}

		if !strings.HasPrefix(j.RunsOn, "${{") {
			completeName, msg, err := wref.checkWorkerModel(ctx, api.mustDB(), api.Cache, jobID, j.RunsOn, j.Region, api.Config.Workflow.JobDefaultRegion)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				return stopRun(ctx, api.mustDB(), api.Cache, run, *u, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelError,
					Message:       fmt.Sprintf("unable to trigger workflow: %v", err),
				})
			}
			if msg != nil {
				return stopRun(ctx, api.mustDB(), api.Cache, run, *u, *msg)
			}
			j.RunsOn = completeName
		}

		run.WorkflowData.Workflow.Jobs[jobID] = j
	}

	run.WorkflowData.Actions = make(map[string]sdk.V2Action)
	for k, v := range wref.actionsCache {
		run.WorkflowData.Actions[k] = v
	}
	for _, v := range wref.localActionsCache {
		completeName := fmt.Sprintf("%s/%s/%s/%s@%s", wref.run.ProjectKey, wref.runVcsServer.Name, wref.runRepo.Name, v.Name, wref.run.WorkflowRef)
		run.WorkflowData.Actions[completeName] = v
	}
	run.WorkflowData.WorkerModels = make(map[string]sdk.V2WorkerModel)
	for k, v := range wref.workerModelCache {
		run.WorkflowData.WorkerModels[k] = v
	}
	for _, v := range wref.localWorkerModelCache {
		completeName := fmt.Sprintf("%s/%s/%s/%s@%s", wref.run.ProjectKey, wref.runVcsServer.Name, wref.runRepo.Name, v.Name, wref.run.WorkflowRef)
		run.WorkflowData.WorkerModels[completeName] = v
	}

	infos, err := wref.checkIntegrations(ctx, api.mustDB())
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return stopRun(ctx, api.mustDB(), api.Cache, run, *u, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       fmt.Sprintf("unable to trigger workflow: %v", err),
		})
	}
	if len(infos) > 0 {
		return stopRun(ctx, api.mustDB(), api.Cache, run, *u, infos...)
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

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(tx.Commit())
	}

	event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunBuilding, *run, *u)

	api.EnqueueWorkflowRun(ctx, run.ID, run.UserID, run.WorkflowName, run.RunNumber)
	return nil
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
			vcsDB, err := vcs.LoadVCSByProject(ctx, db, projKey, vcsName)
			if err != nil {
				if sdk.ErrorIs(err, sdk.ErrNotFound) {
					return "", &sdk.V2WorkflowRunInfo{WorkflowRunID: wref.run.ID, Level: sdk.WorkflowRunInfoLevelError, Message: fmt.Sprintf("vcs %s not found on project %s", vcsName, projKey)}, nil
				}
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
				if sdk.ErrorIs(err, sdk.ErrNotFound) {
					return "", &sdk.V2WorkflowRunInfo{WorkflowRunID: wref.run.ID, Level: sdk.WorkflowRunInfoLevelError, Message: fmt.Sprintf("repository %s not found on vcs %s into project %s", repoName, vcsName, projKey)}, nil
				}
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
					_ = tx.Rollback()
					return "", nil, err
				}
				b, err := client.Branch(ctx, entityRepo.Name, sdk.VCSBranchFilters{Default: true})
				if err != nil {
					_ = tx.Rollback()
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
			return "", &sdk.V2WorkflowRunInfo{WorkflowRunID: wref.run.ID, Level: sdk.WorkflowRunInfoLevelError, Message: fmt.Sprintf("obsolete workflow dependency used: %s", name)}, nil
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
	default:
		return "", nil, sdk.NewErrorFrom(sdk.ErrNotImplemented, "entity %s not implemented", entityType)
	}
	return completePath, nil, nil
}

func searchActions(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, steps []sdk.ActionStep) (*sdk.V2WorkflowRunInfo, error) {
	ctx, end := telemetry.Span(ctx, "searchActions")
	defer end()
	for i := range steps {
		step := &steps[i]
		if step.Uses == "" {
			continue
		}

		if strings.HasPrefix(step.Uses, ".cds/actions/") {
			// Find action from path
			localAct, has := wref.localActionsCache[step.Uses]
			if !has {
				actionEntity, err := entity.LoadEntityByPathAndBranch(ctx, db, wref.runRepo.ID, step.Uses, wref.run.WorkflowRef)
				if err != nil {
					msg := sdk.V2WorkflowRunInfo{
						WorkflowRunID: wref.run.ID,
						Level:         sdk.WorkflowRunInfoLevelError,
						Message:       fmt.Sprintf("Unable to find action %s", step.Uses),
					}
					return &msg, nil
				}
				if err := yaml.Unmarshal([]byte(actionEntity.Data), &localAct); err != nil {
					return nil, err
				}
				wref.localActionsCache[step.Uses] = localAct
				msg, err := searchActions(ctx, db, store, wref, localAct.Runs.Steps)
				if msg != nil || err != nil {
					return msg, err
				}
			}
			completeName := fmt.Sprintf("%s/%s/%s/%s@%s", wref.run.ProjectKey, wref.runVcsServer.Name, wref.runRepo.Name, localAct.Name, wref.run.WorkflowRef)
			step.Uses = "actions/" + completeName
		} else {
			actionName := strings.TrimPrefix(step.Uses, "actions/")
			actionSplit := strings.Split(actionName, "/")
			// If plugins
			if strings.HasPrefix(step.Uses, "actions/") && len(actionSplit) == 1 {
				// Check plugins
				if _, has := wref.plugins[actionSplit[0]]; !has {
					msg := sdk.V2WorkflowRunInfo{
						WorkflowRunID: wref.run.ID,
						Level:         sdk.WorkflowRunInfoLevelError,
						Message:       fmt.Sprintf("Action %s doesn't exist", actionSplit[0]),
					}
					return &msg, nil
				}
				continue
			} else {
				completeName, msg, err := wref.searchEntity(ctx, db, store, actionName, sdk.EntityTypeAction)
				if msg != nil || err != nil {
					return msg, err
				}
				// rewrite step with full path
				step.Uses = "actions/" + completeName
				act := wref.actionsCache[completeName]
				msg, err = searchActions(ctx, db, store, wref, act.Runs.Steps)
				if msg != nil || err != nil {
					return msg, err
				}
			}
		}

	}
	return nil, nil
}

func stopRun(ctx context.Context, db *gorp.DbMap, store cache.Store, run *sdk.V2WorkflowRun, u sdk.AuthentifiedUser, messages ...sdk.V2WorkflowRunInfo) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	var status = sdk.StatusSkipped

	for _, msg := range messages {
		if err := workflow_v2.InsertRunInfo(ctx, tx, &msg); err != nil {
			return err
		}
		if msg.Level != sdk.WorkflowRunInfoLevelWarning && status == sdk.StatusSkipped {
			status = sdk.StatusFail
		}
	}

	run.Status = status

	if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	event_v2.PublishRunEvent(ctx, store, sdk.EventRunEnded, *run, u)

	return nil
}

func buildRunContext(ctx context.Context, db *gorp.DbMap, store cache.Store, p sdk.Project, wr sdk.V2WorkflowRun, vcsServer sdk.VCSProject, repo sdk.ProjectRepository, u sdk.AuthentifiedUser) (*sdk.WorkflowRunContext, error) {
	var runContext sdk.WorkflowRunContext

	var ref, commit, semverNext, semverCurrent string

	cdsContext := sdk.CDSContext{
		ProjectKey:         wr.ProjectKey,
		RunID:              wr.ID,
		RunNumber:          wr.RunNumber,
		RunAttempt:         1,
		Workflow:           wr.WorkflowName,
		WorkflowRef:        wr.WorkflowRef,
		WorkflowSha:        wr.WorkflowSha,
		WorkflowVCSServer:  vcsServer.Name,
		WorkflowRepository: repo.Name,
		Event:              nil,
		TriggeringActor:    u.Username,
		EventName:          "",
	}
	switch {
	case wr.RunEvent.Manual != nil:
		cdsContext.EventName = "manual"
		cdsContext.Event = wr.RunEvent.Manual.Payload
		if r, has := wr.RunEvent.Manual.Payload[sdk.GitRefManualPayload]; has {
			if refString, ok := r.(string); ok {
				ref = refString
			}
		}
		if c, has := wr.RunEvent.Manual.Payload[sdk.GitCommitManualPayload]; has {
			if commitString, ok := c.(string); ok {
				commit = commitString
			}
		}
	case wr.RunEvent.GitTrigger != nil:
		cdsContext.EventName = wr.RunEvent.GitTrigger.EventName
		cdsContext.Event = wr.RunEvent.GitTrigger.Payload
		ref = wr.RunEvent.GitTrigger.Ref
		commit = wr.RunEvent.GitTrigger.Sha

		semverNext = wr.RunEvent.GitTrigger.SemverNext

		// Compute current semver version with CDS metadata
		currentVersion, _ := semver.NewVersion(wr.RunEvent.GitTrigger.SemverCurrent)
		if currentVersion != nil {
			if currentVersion.Metadata() == "" { // Tags doesn't have metadata
				semverCurrent = wr.RunEvent.GitTrigger.SemverCurrent
			} else {
				suffix := currentVersion.Metadata()
				splittedSuffix := strings.Split(suffix, ".")
				var metadataStr string
				if len(splittedSuffix) >= 2 {
					metadataStr += splittedSuffix[0] + ".sha." + splittedSuffix[1]
				}
				for i := 2; i < len(splittedSuffix); i++ {
					metadataStr += "." + splittedSuffix[i]
				}
				preRelease := currentVersion.Prerelease()
				var v = *currentVersion
				if preRelease != "" {
					v, _ = currentVersion.SetPrerelease(preRelease + "-" + strconv.FormatInt(wr.RunNumber, 10))
				} else {
					v, _ = currentVersion.SetPrerelease(strconv.FormatInt(wr.RunNumber, 10))
				}
				v, _ = v.SetMetadata(metadataStr)
				semverCurrent = v.String()
			}
		} else {
			// If no semver found, compute it from 0.1.0
			semverCurrent = "0.1.0+" + strconv.FormatInt(wr.RunNumber, 10) + ".sha." + commit
		}

	case wr.RunEvent.ModelUpdateTrigger != nil:
		ref = wr.RunEvent.ModelUpdateTrigger.Ref
	case wr.RunEvent.WorkflowUpdateTrigger != nil:
		ref = wr.RunEvent.WorkflowUpdateTrigger.Ref
	default:
		// TODO implement scheduler and webhooks
		return nil, sdk.NewErrorFrom(sdk.ErrNotImplemented, "RunEvent not implemented: %+v", wr.RunEvent)
	}

	// Reload VCS with decryption to have sshkey / git username
	vcsName := vcsServer.Name
	if wr.WorkflowData.Workflow.Repository != nil && wr.WorkflowData.Workflow.Repository.VCSServer != vcsServer.Name {
		vcsName = wr.WorkflowData.Workflow.Repository.VCSServer
	}
	vcsTmp, err := vcs.LoadVCSByProject(ctx, db, wr.ProjectKey, vcsName, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return nil, err
	}
	workflowVCSServer := *vcsTmp

	gitContext := sdk.GitContext{
		Server:        workflowVCSServer.Name,
		SSHKey:        workflowVCSServer.Auth.SSHKeyName,
		Username:      workflowVCSServer.Auth.Username,
		Ref:           ref,
		Sha:           commit,
		SemverCurrent: semverCurrent,
		SemverNext:    semverNext,
	}

	if gitContext.SSHKey != "" {
		gitContext.Connection = "ssh"
	} else {
		gitContext.Connection = "https"
	}

	if wr.WorkflowData.Workflow.Repository != nil {
		gitContext.Repository = wr.WorkflowData.Workflow.Repository.Name
	} else {
		gitContext.Repository = repo.Name
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer tx.Rollback()
	vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, store, wr.ProjectKey, workflowVCSServer.Name)
	if err != nil {
		return nil, err
	}

	if gitContext.Repository == repo.Name {
		gitContext.RepositoryURL = repo.CloneURL
	} else {
		vcsRepo, err := vcsClient.RepoByFullname(ctx, gitContext.Repository)
		if err != nil {
			return nil, err
		}
		if gitContext.SSHKey != "" {
			gitContext.RepositoryURL = vcsRepo.SSHCloneURL
		} else {
			gitContext.RepositoryURL = vcsRepo.HTTPCloneURL
		}
	}
	if gitContext.Ref == "" {
		defaultBranch, err := vcsClient.Branch(ctx, gitContext.Repository, sdk.VCSBranchFilters{Default: true})
		if err != nil {
			return nil, err
		}
		gitContext.Ref = defaultBranch.DisplayID
		if gitContext.Sha == "" {
			gitContext.Sha = defaultBranch.LatestCommit
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, sdk.WithStack(err)
	}

	// VARS context
	vars := make(map[string]string, 0)
	for _, v := range p.Variables {
		if sdk.NeedPlaceholder(v.Type) {
			continue
		}
		vName := strings.Replace(v.Name, ".", "_", -1)
		vars[vName] = v.Value
	}

	// Env context
	envs := make(map[string]string)
	for k, v := range wr.WorkflowData.Workflow.Env {
		envs[k] = v
	}

	runContext.CDS = cdsContext
	runContext.Git = gitContext
	runContext.Vars = vars
	runContext.Env = envs
	return &runContext, nil
}

func (wref *WorkflowRunEntityFinder) checkWorkerModel(ctx context.Context, db *gorp.DbMap, store cache.Store, jobName, workerModel, reg, defaultRegion string) (string, *sdk.V2WorkflowRunInfo, error) {
	ctx, next := telemetry.Span(ctx, "wref.checkWorkerModel", trace.StringAttribute(telemetry.TagWorkerModel, workerModel))
	defer next()

	hatcheries, err := hatchery.LoadHatcheries(ctx, db)
	if err != nil {
		return "", nil, err
	}
	if reg == "" {
		reg = defaultRegion
	}

	currentRegion, err := region.LoadRegionByName(ctx, db, reg)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			msg := &sdk.V2WorkflowRunInfo{
				WorkflowRunID: wref.run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("region %s not found", reg),
			}
			return "", msg, nil
		}
		return "", nil, err

	}

	modelCompleteName := ""
	modelType := ""

	if workerModel != "" {
		if strings.HasPrefix(workerModel, ".cds/worker-models/") {
			// Find action from path
			localWM, has := wref.localWorkerModelCache[workerModel]
			if !has {
				wmEntity, err := entity.LoadEntityByPathAndBranch(ctx, db, wref.runRepo.ID, workerModel, wref.run.WorkflowRef)
				if err != nil {
					msg := sdk.V2WorkflowRunInfo{
						WorkflowRunID: wref.run.ID,
						Level:         sdk.WorkflowRunInfoLevelError,
						Message:       fmt.Sprintf("Unable to find worker model %s", workerModel),
					}
					return "", &msg, nil
				}
				if err := yaml.Unmarshal([]byte(wmEntity.Data), &localWM); err != nil {
					return "", nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to read worker model %s: %v", workerModel, err)
				}
				wref.localWorkerModelCache[workerModel] = localWM
			}
			modelCompleteName = fmt.Sprintf("%s/%s/%s/%s@%s", wref.run.ProjectKey, wref.runVcsServer.Name, wref.runRepo.Name, localWM.Name, wref.run.WorkflowRef)
			modelType = localWM.Type
		} else {
			completeName, msg, err := wref.searchEntity(ctx, db, store, workerModel, sdk.EntityTypeWorkerModel)
			if err != nil {
				return "", nil, err
			}
			if msg != nil {
				return "", msg, nil
			}
			modelType = wref.workerModelCache[completeName].Type
			modelCompleteName = completeName
		}

	}

	for _, h := range hatcheries {
		if h.ModelType == modelType {
			// check permission
			rbacHatchery, err := rbac.LoadRBACHatcheryByHatcheryID(ctx, db, h.ID)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return "", nil, err
			}
			if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
				continue
			}
			if rbacHatchery.RegionID == currentRegion.ID {
				return modelCompleteName, nil, nil
			}
		}
	}

	return modelCompleteName, &sdk.V2WorkflowRunInfo{
		WorkflowRunID: wref.run.ID,
		Level:         sdk.WorkflowRunInfoLevelError,
		IssuedAt:      time.Now(),
		Message:       fmt.Sprintf("wrong configuration on job %q. No hatchery can run it", jobName),
	}, nil
}

func (wref *WorkflowRunEntityFinder) checkIntegrations(ctx context.Context, db *gorp.DbMap) ([]sdk.V2WorkflowRunInfo, error) {
	availableIntegrations, err := integration.LoadIntegrationsByProjectID(ctx, db, wref.project.ID)
	if err != nil {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to load integration")
	}

	var infos []sdk.V2WorkflowRunInfo

	for i := range wref.run.WorkflowData.Workflow.Integrations {
		var found bool
		for j := range availableIntegrations {
			if wref.run.WorkflowData.Workflow.Integrations[i] == availableIntegrations[j].Name {
				found = true
				break
			}
		}
		if !found {
			infos = append(infos, sdk.V2WorkflowRunInfo{
				WorkflowRunID: wref.run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				IssuedAt:      time.Now(),
				Message:       fmt.Sprintf("wrong workflow configuration. Integration %s does not exist", wref.run.WorkflowData.Workflow.Integrations[i]),
			})
		}
	}

	for jobID, job := range wref.run.WorkflowData.Workflow.Jobs {
		for _, integ := range job.Integrations {
			var found bool
			for j := range availableIntegrations {
				if integ == availableIntegrations[j].Name {
					found = true
					if availableIntegrations[j].Model.ArtifactManager {
						infos = append(infos, sdk.V2WorkflowRunInfo{
							WorkflowRunID: wref.run.ID,
							Level:         sdk.WorkflowRunInfoLevelError,
							IssuedAt:      time.Now(),
							Message:       fmt.Sprintf("wrong configuration on job %q. Integration %q cannot be used at the job level", jobID, integ),
						})
					}
					break
				}
			}
			if !found {
				infos = append(infos, sdk.V2WorkflowRunInfo{
					WorkflowRunID: wref.run.ID,
					Level:         sdk.WorkflowRunInfoLevelError,
					IssuedAt:      time.Now(),
					Message:       fmt.Sprintf("wrong configuration on job %q. Integration %q does not exist", jobID, integ),
				})
			}
		}
	}

	return infos, nil
}
