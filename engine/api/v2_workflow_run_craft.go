package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-gorp/gorp"
	"github.com/pelletier/go-toml"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

type WorkflowRunEntityFinder struct {
	run     sdk.V2WorkflowRun
	ef      *EntityFinder
	project sdk.Project
}

func NewWorkflowRunEntityFinder(ctx context.Context, db *gorp.DbMap, proj sdk.Project, run sdk.V2WorkflowRun, repo sdk.ProjectRepository, vcsServer sdk.VCSProject, ref, sha string, libraryProjectKey string, initiator *sdk.V2Initiator) (*WorkflowRunEntityFinder, error) {
	if initiator == nil {
		initiator = run.Initiator
	}
	ef, err := NewEntityFinder(ctx, db, proj.Key, ref, sha, repo, vcsServer, *initiator, libraryProjectKey)
	if err != nil {
		return nil, err
	}
	return &WorkflowRunEntityFinder{
		ef:      ef,
		run:     run,
		project: proj,
	}, nil
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
							ctx := log.ContextWithStackTrace(ctx, err)
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

	if run.Status != sdk.V2WorkflowRunStatusCrafting {
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

	p, err := project.Load(ctx, api.mustDB(), run.ProjectKey)
	if err != nil {
		return err
	}

	// Build run context
	runContext, mustSaveVersion, err := buildRunContext(ctx, api.mustDB(), api.Cache, *run, *vcsServer, *repo, api.Config.URL.UI)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       fmt.Sprintf("%v", err),
		})
	}
	run.Contexts = *runContext

	wref, err := NewWorkflowRunEntityFinder(ctx, api.mustDB(), *p, *run, *repo, *vcsServer, run.WorkflowRef, run.WorkflowSha, api.Config.WorkflowV2.LibraryProjectKey, nil)
	if err != nil {
		return err
	}

	// Resolve workflow template if applicable
	if run.WorkflowData.Workflow.From != "" {
		e, msg, err := wref.checkWorkflowTemplate(ctx, api.mustDB(), api.Cache, run.WorkflowData.Workflow.From)
		if err != nil {
			return err
		}
		if msg != nil {
			return stopRun(ctx, api.mustDB(), api.Cache, run, nil, *msg)
		}

		if _, err := e.Template.Resolve(ctx, &run.WorkflowData.Workflow); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("Unable to resolve workflow template %s: %s", run.WorkflowData.Workflow.From, err),
			})
		}

		// Build context for workflow
		repo, err := repository.LoadRepositoryByID(ctx, api.mustDB(), e.Entity.ProjectRepositoryID)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("Unable to resolve workflow template repository: %s", err),
			})
		}

		vcsProj, err := vcs.LoadVCSByIDAndProjectKey(ctx, api.mustDB(), repo.ProjectKey, repo.VCSProjectID)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("Unable to resolve workflow template vcs: %s", err),
			})
		}

		vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, e.Entity.ProjectKey, vcsProj.Name)
		if err != nil {
			return err
		}
		vcsRepo, err := vcsClient.RepoByFullname(ctx, repo.Name)
		if err != nil {
			return err
		}

		run.Contexts.CDS.WorkflowTemplate = e.Entity.Name
		run.Contexts.CDS.WorkflowTemplateRef = e.Entity.Ref
		run.Contexts.CDS.WorkflowTemplateSha = e.Entity.Commit
		run.Contexts.CDS.WorkflowTemplateVCSServer = vcsProj.Name
		run.Contexts.CDS.WorkflowTemplateRepository = repo.Name
		run.Contexts.CDS.WorkflowTemplateProjectKey = e.Entity.ProjectKey
		run.Contexts.CDS.WorkflowTemplateParams = run.WorkflowData.Workflow.Parameters
		run.Contexts.CDS.WorkflowTemplateCommitWebURL = fmt.Sprintf(vcsRepo.URLCommitFormat, e.Entity.Commit)
		run.Contexts.CDS.WorkflowTemplateRepositoryWebURL = vcsRepo.URL
		if strings.HasPrefix(e.Entity.Ref, sdk.GitRefTagPrefix) {
			run.Contexts.CDS.WorkflowTemplateRefWebURL = fmt.Sprintf(vcsRepo.URLTagFormat, strings.TrimPrefix(e.Entity.Ref, sdk.GitRefTagPrefix))
		} else if strings.HasPrefix(e.Entity.Ref, sdk.GitRefBranchPrefix) {
			run.Contexts.CDS.WorkflowTemplateRefWebURL = fmt.Sprintf(vcsRepo.URLBranchFormat, strings.TrimPrefix(e.Entity.Ref, sdk.GitRefBranchPrefix))
		}

		// Recreate wref with vcs/repo = template's repo and not workflow's repo
		wref, err = NewWorkflowRunEntityFinder(ctx, api.mustDB(), *p, *run, *repo, *vcsProj, e.Ref, e.Commit, api.Config.WorkflowV2.LibraryProjectKey, &wref.ef.initiator)
		if err != nil {
			return err
		}

		// Lint workflow. Create tmp workflow without FROM to check workflow structure
		tmpWkf := run.WorkflowData.Workflow
		tmpWkf.From = ""
		if errsLint := Lint(ctx, api.mustDB(), api.Cache, tmpWkf, wref.ef, api.WorkerModelDockerImageWhiteList); errsLint != nil {
			//run.Status = sdk.V2WorkflowRunStatusFail
			msgs := make([]sdk.V2WorkflowRunInfo, 0, len(errsLint))
			for _, e := range errsLint {
				msgs = append(msgs, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					IssuedAt:      time.Now(),
					Level:         sdk.WorkflowRunInfoLevelError,
					Message:       e.Error(),
				})
			}
			return stopRun(ctx, api.mustDB(), api.Cache, run, &wref.ef.initiator, msgs...)
		}
	}

	allVariableSets, err := project.LoadVariableSetsByProject(ctx, api.mustDB(), p.Key)
	if err != nil {
		return err
	}

	// Apply all job templates
	for jobID := range run.WorkflowData.Workflow.Jobs {
		j := run.WorkflowData.Workflow.Jobs[jobID]
		if j.From != "" {
			msgs, err := api.computeJobFromTemplate(ctx, api.mustDB(), api.Cache, wref, jobID, j, run, allVariableSets, api.Config.Workflow.JobDefaultRegion)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelError,
					Message:       fmt.Sprintf("unable to compute job[%s] with template %s: %v", jobID, j.From, err),
				})
			}
			if len(msgs) > 0 {
				return stopRun(ctx, api.mustDB(), api.Cache, run, nil, msgs...)
			}
		}
	}

	// Check workflow lint in case of modification through job template
	msgs := make([]sdk.V2WorkflowRunInfo, 0)
	errs := run.WorkflowData.Workflow.Lint()
	for _, e := range errs {
		msgs = append(msgs, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			IssuedAt:      time.Now(),
			Message:       e.Error(),
		})
	}
	if len(msgs) > 0 {
		return stopRun(ctx, api.mustDB(), api.Cache, run, nil, msgs...)
	}

	// Reload integration regarding jobs
	integrations, infos, err := wref.checkIntegrations(ctx, api.mustDB(), run.WorkflowData.Workflow.Jobs)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       fmt.Sprintf("unable to trigger workflow: %v", err),
		})
	}
	if len(infos) > 0 {
		return stopRun(ctx, api.mustDB(), api.Cache, run, nil, infos...)
	}

	// Retrieve all deps
	for jobID := range run.WorkflowData.Workflow.Jobs {
		msg := retrieveAndUpdateAllJobDependencies(ctx, api.mustDB(), api.Cache, run, jobID, run.WorkflowData.Workflow.Jobs[jobID], wref, integrations, allVariableSets, api.Config.Workflow.JobDefaultRegion)
		if msg != nil {
			return stopRun(ctx, api.mustDB(), api.Cache, run, nil, *msg)
		}
	}

	// Interpolate workflow.concurrencies
	bts, err := json.Marshal(run.Contexts)
	if err != nil {
		return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			IssuedAt:      time.Now(),
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       "unable to read run context. Please contact an administrator",
		})
	}
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			IssuedAt:      time.Now(),
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       "unable to read run context. Please contact an administrator",
		})
	}
	ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
	for i := range run.WorkflowData.Workflow.Concurrencies {
		c := &run.WorkflowData.Workflow.Concurrencies[i]
		if strings.Contains(c.Name, "${{") {
			interpolatedString, err := ap.InterpolateToString(ctx, c.Name)
			if err != nil {
				return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					IssuedAt:      time.Now(),
					Level:         sdk.WorkflowRunInfoLevelError,
					Message:       "unable to read run context. Please contact an administrator",
				})
			}
			c.Name = interpolatedString
		}
		if c.Order == "" {
			c.Order = sdk.ConcurrencyOrderOldestFirst
		}
		if c.Pool == 0 {
			c.Pool = 1
		}
	}
	// Compute workflow concurrency
	if strings.Contains(run.WorkflowData.Workflow.Concurrency, "${{") {
		interpolatedString, err := ap.InterpolateToString(ctx, run.WorkflowData.Workflow.Concurrency)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       err.Error(),
			})
		}
		run.WorkflowData.Workflow.Concurrency = interpolatedString
	}

	run.WorkflowData.Actions = make(map[string]sdk.V2Action)
	for k, v := range wref.ef.actionsCache {
		run.WorkflowData.Actions[k] = v.Action
	}
	for _, v := range wref.ef.localActionsCache {
		completeName := fmt.Sprintf("%s/%s/%s/%s@%s", wref.run.ProjectKey, wref.ef.currentVCS.Name, wref.ef.currentRepo.Name, v.Name, wref.ef.currentRef)
		run.WorkflowData.Actions[completeName] = v.Action
	}
	run.WorkflowData.WorkerModels = make(map[string]sdk.V2WorkerModel)
	for k, v := range wref.ef.workerModelCache {
		run.WorkflowData.WorkerModels[k] = v.Model
	}
	for _, v := range wref.ef.localWorkerModelCache {
		completeName := fmt.Sprintf("%s/%s/%s/%s@%s", wref.run.ProjectKey, wref.ef.currentVCS.Name, wref.ef.currentRepo.Name, v.Model.Name, wref.ef.currentRef)
		run.WorkflowData.WorkerModels[completeName] = v.Model
	}

	// check concurrency
	runInfos := make([]sdk.V2WorkflowRunInfo, 0)
	if run.WorkflowData.Workflow.Concurrency != "" {
		concurrencyDef, err := retrieveConcurrencyDefinition(ctx, api.mustDB(), *run, run.WorkflowData.Workflow.Concurrency)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("unable to retrieve concurrency %q: %v", run.WorkflowData.Workflow.Concurrency, err),
			})
		}
		if concurrencyDef == nil {
			return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("concurrency %q not found on workflow nor on project", run.WorkflowData.Workflow.Concurrency),
			})
		}
		run.Concurrency = concurrencyDef

		// Check concurrency condition
		if run.Concurrency.If != "" {
			if !strings.HasPrefix(run.Concurrency.If, "${{") {
				run.Concurrency.If = fmt.Sprintf("${{ %s }}", run.Concurrency.If)
			}
			useConcurrency, err := ap.InterpolateToBool(ctx, run.Concurrency.If)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					IssuedAt:      time.Now(),
					Level:         sdk.WorkflowRunInfoLevelError,
					Message:       fmt.Sprintf("unable to interpolate concurrency %q condition %q: %v", run.Concurrency.Name, run.Concurrency.If, err),
				})
			}
			// If we don't have to use the concurrency, remove it from the workflow run
			if !useConcurrency {
				run.Concurrency = nil
				runInfos = append(runInfos, sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					IssuedAt:      time.Now(),
					Level:         sdk.WorkflowRunInfoLevelInfo,
					Message:       fmt.Sprintf("Concurrency %q skipped", run.WorkflowData.Workflow.Concurrency),
				})
			}
		}

		// Lock concurrency if needed
		if run.Concurrency != nil {
			concurrencyKey := getConcurrencyUniqueKey(*run.Concurrency, run.ProjectKey, run.VCSServer, run.Repository, run.WorkflowName)
			concurrencyLockKey := cache.Key("api:workflow:concurrency:enqueue", concurrencyKey)
			locked, err := api.Cache.Lock(concurrencyLockKey, 1*time.Minute, 0, 1)
			if err != nil {
				return err
			}
			if !locked {
				log.Info(ctx, "concurrency %q already locked", concurrencyKey)
				time.Sleep(1 * time.Second)
				return nil
			}
			defer api.Cache.Unlock(concurrencyLockKey)
		}

	}

	runObjectToCancel := make(map[string]workflow_v2.ConcurrencyObject)
	runInfo, err := manageWorkflowConcurrency(ctx, api.mustDB(), run, map[string]int64{}, runObjectToCancel)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return stopRun(ctx, api.mustDB(), api.Cache, run, nil, sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			IssuedAt:      time.Now(),
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       "unable to start the workflow. Please contact an administrator",
		})
	}
	if runInfo != nil {
		runInfos = append(runInfos, *runInfo)
	}

	tx, err := api.mustDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
		return err
	}

	for _, runInfo := range runInfos {
		if err := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); err != nil {
			return err
		}
	}
	if mustSaveVersion {
		wkfVersion := sdk.V2WorkflowVersion{
			Version:            run.Contexts.CDS.Version,
			ProjectKey:         run.Contexts.CDS.ProjectKey,
			WorkflowVCS:        run.Contexts.CDS.WorkflowVCSServer,
			WorkflowRepository: run.Contexts.CDS.WorkflowRepository,
			WorkflowRef:        run.Contexts.CDS.WorkflowRef,
			WorkflowSha:        run.Contexts.CDS.WorkflowSha,
			VCSServer:          run.Contexts.Git.Server,
			Repository:         run.Contexts.Git.Repository,
			WorkflowName:       run.Contexts.CDS.Workflow,
			WorkflowRunID:      run.ID,
			Sha:                run.Contexts.Git.Sha,
			Ref:                run.Contexts.Git.Ref,
			Type:               string(run.WorkflowData.Workflow.Semver.From),
			File:               run.WorkflowData.Workflow.Semver.Path,
			Username:           run.Initiator.Username(),
			UserID:             run.Initiator.UserID,
		}
		if err := workflow_v2.InsertWorkflowVersion(ctx, tx, &wkfVersion); err != nil {
			return err
		}
	}

	if err := api.cancelRunObjects(ctx, tx, runObjectToCancel); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(tx.Commit())
	}

	event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunBuilding, *run, nil, nil, nil)
	api.EnqueueWorkflowRun(ctx, run.ID, *run.Initiator, run.WorkflowName, run.RunNumber)
	return nil
}

func retrieveAndUpdateAllJobDependencies(ctx context.Context, db *gorp.DbMap, store cache.Store, run *sdk.V2WorkflowRun, jobID string, j sdk.V2Job, wref *WorkflowRunEntityFinder, integrations map[string]sdk.ProjectIntegration, allVariableSets []sdk.ProjectVariableSet, defaultRegion string) *sdk.V2WorkflowRunInfo {
	if len(j.Steps) == 0 && j.From == "" {
		return nil
	}

	// Check integration region
	if len(j.Integrations) > 0 && j.Region == "" {
	regionLoop:
		for _, jobInt := range j.Integrations {
			if !strings.Contains(jobInt, "${{") {
				for _, integ := range integrations {
					if integ.Name != jobInt {
						continue
					}
					for _, v := range integ.Config {
						if v.Type == sdk.IntegrationConfigTypeRegion {
							j.Region = v.Value
							break regionLoop
						}
					}
				}
			}
		}
	}

	// Get actions and sub actions

	msg, err := searchActions(ctx, db, store, wref, j.Steps)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return &sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       fmt.Sprintf("unable to retrieve job[%s] definition. Please contact an administrator", jobID),
		}
	}
	if msg != nil {
		return msg
	}

	// Check worker model
	if !strings.Contains(j.RunsOn.Model, "${{") && j.From == "" && !strings.Contains(j.Region, "${{") {
		completeName, msg, err := wref.checkWorkerModel(ctx, db, store, jobID, j.RunsOn.Model, j.Region, defaultRegion)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return &sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("unable to compute worker model %s: %v", j.RunsOn.Model, err),
			}
		}
		if msg != nil {
			return msg
		}
		j.RunsOn.Model = completeName
	}

	// Check variable set
	if err := checkWorkflowVariableSets(*run, j, allVariableSets); err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return &sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			IssuedAt:      time.Now(),
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       err.Error(),
		}
	}
	vss := sdk.StringSlice{}
	vss = append(vss, run.WorkflowData.Workflow.VariableSets...)
	vss = append(vss, j.VariableSets...)
	vss.Unique()
	j.VariableSets = vss

	// Check concurrency
	if strings.Contains(j.Concurrency, "${{") {
		bts, err := json.Marshal(run.Contexts)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return &sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       "unable to read run context. Please contact an administrator",
			}
		}
		var mapContexts map[string]interface{}
		if err := json.Unmarshal(bts, &mapContexts); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return &sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       "unable to read run context. Please contact an administrator",
			}
		}
		ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
		interpolatedString, err := ap.InterpolateToString(ctx, j.Concurrency)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return &sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       err.Error(),
			}
		}
		j.Concurrency = interpolatedString
	}

	run.WorkflowData.Workflow.Jobs[jobID] = j
	return nil
}

func checkWorkflowVariableSets(run sdk.V2WorkflowRun, currentJob sdk.V2Job, projectVariableSets []sdk.ProjectVariableSet) error {
	for _, vsName := range run.WorkflowData.Workflow.VariableSets {
		vsFound := false
		for _, projVarset := range projectVariableSets {
			if projVarset.Name == vsName {
				vsFound = true
			}
		}
		if !vsFound {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "variable set %s not found on project", vsName)
		}
	}
	for _, vsName := range currentJob.VariableSets {
		vsFound := false
		for _, projVarset := range projectVariableSets {
			if projVarset.Name == vsName {
				vsFound = true
			}
		}
		if !vsFound {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "variable set %s not found on project", vsName)
		}
	}
	return nil
}

func checkJobTemplate(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, j sdk.V2Job, run *sdk.V2WorkflowRun, params map[string]string) (*sdk.EntityWithObject, *sdk.V2Workflow, []sdk.V2WorkflowRunInfo, error) {
	// Check is template exist
	e, msg, err := wref.checkWorkflowTemplate(ctx, db, store, j.From)
	if err != nil {
		return nil, nil, nil, err
	}
	if msg != nil {
		return &e, nil, []sdk.V2WorkflowRunInfo{*msg}, nil
	}

	tmpWorkflow := sdk.V2Workflow{
		Name:       "tmp",
		Parameters: params,
	}

	if _, err := e.Template.Resolve(ctx, &tmpWorkflow); err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return &e, nil, nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to resolve workflow template %s: %s", j.From, err)
	}
	tmpWorkflow.Name = e.Name

	// Add stage from parent workflow to lint jobs
	if tmpWorkflow.Stages == nil {
		tmpWorkflow.Stages = make(map[string]sdk.WorkflowStage)
	}
	for k, v := range run.WorkflowData.Workflow.Stages {
		tmpWorkflow.Stages[k] = v
	}
	if tmpWorkflow.Gates == nil {
		tmpWorkflow.Gates = make(map[string]sdk.V2JobGate)
	}
	for k, v := range run.WorkflowData.Workflow.Gates {
		tmpWorkflow.Gates[k] = v
	}
	tmpWorkflow.Integrations = append(tmpWorkflow.Integrations, run.WorkflowData.Workflow.Integrations...)
	tmpWorkflow.VariableSets = append(tmpWorkflow.VariableSets, run.WorkflowData.Workflow.VariableSets...)

	repoTemplate, err := repository.LoadRepositoryByID(ctx, db, e.ProjectRepositoryID)
	if err != nil {
		return &e, nil, nil, err
	}
	vcsTemplate, err := vcs.LoadVCSByIDAndProjectKey(ctx, db, e.ProjectKey, repoTemplate.VCSProjectID)
	if err != nil {
		return &e, nil, nil, err
	}
	wrefTemplate, err := NewWorkflowRunEntityFinder(ctx, db, wref.project, *run, *repoTemplate, *vcsTemplate, e.Ref, e.Commit, wref.ef.libraryProject, &wref.ef.initiator)
	if err != nil {
		return &e, nil, nil, err
	}

	// Lint generated workflow
	if errsLint := Lint(ctx, db, store, tmpWorkflow, wrefTemplate.ef, nil); errsLint != nil {
		//run.Status = sdk.V2WorkflowRunStatusFail
		msgs := make([]sdk.V2WorkflowRunInfo, 0, len(errsLint))
		for _, e := range errsLint {
			msgs = append(msgs, sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       e.Error(),
			})
		}
		return &e, nil, msgs, nil
	}

	return &e, &tmpWorkflow, nil, nil
}

func (a *API) computeJobFromTemplate(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, jobID string, j sdk.V2Job, run *sdk.V2WorkflowRun, allVariableSets []sdk.ProjectVariableSet, defaultRegion string) ([]sdk.V2WorkflowRunInfo, error) {
	ctx, end := telemetry.Span(ctx, "computeJobFromTemplate")
	defer end()

	// Retrieve Template and Lint
	entityTemplate, tmpWorkflow, msgs, err := checkJobTemplate(ctx, a.mustDB(), a.Cache, wref, j, run, j.Parameters)
	if err != nil {
		return nil, err
	}
	if len(msgs) > 0 {
		return msgs, nil
	}

	// If there is no matrix, compute the final workflow now. Else it will be done at runtime
	if j.Strategy == nil || len(j.Strategy.Matrix) == 0 {
		msgs, err := handleTemplatedJobInWorkflow(ctx, db, store, wref, entityTemplate, run, tmpWorkflow.Jobs, tmpWorkflow.Stages, tmpWorkflow.Gates, tmpWorkflow.Annotations, jobID, j, allVariableSets, defaultRegion)
		if err != nil {
			return nil, err
		}
		if len(msgs) > 0 {
			return msgs, nil
		}

		// Remove templated job
		delete(run.WorkflowData.Workflow.Jobs, jobID)
	}
	return nil, nil
}

func handleTemplatedJobInWorkflow(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, templateEntity *sdk.EntityWithObject, run *sdk.V2WorkflowRun, newJobs map[string]sdk.V2Job, newStages map[string]sdk.WorkflowStage, newGates map[string]sdk.V2JobGate, newAnnotations map[string]string, jobID string, j sdk.V2Job, allVariableSets []sdk.ProjectVariableSet, defaultRegion string) ([]sdk.V2WorkflowRunInfo, error) {
	// Check duplication of jobID
	// Retrieve root job
	rootJobs := make([]string, 0)
	for subJobID, subJob := range newJobs {
		if _, exist := run.WorkflowData.Workflow.Jobs[subJobID]; exist {
			msgs := []sdk.V2WorkflowRunInfo{{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("job %s: job %s defined in template %s already exist in the parent workflow", jobID, subJobID, j.From),
			}}
			return msgs, nil
		}
		if len(subJob.Needs) == 0 {
			rootJobs = append(rootJobs, subJobID)
		}
	}

	// Retrieve final jobs
	finalJobs := make([]string, 0)
loop:
	for subJobID := range newJobs {
		for _, jobDef := range newJobs {
			if slices.Contains(jobDef.Needs, subJobID) {
				continue loop
			}
		}
		finalJobs = append(finalJobs, subJobID)
	}

	// Set needs on templated root job
	if len(j.Needs) > 0 {
		for _, id := range rootJobs {
			jobDef := newJobs[id]

			// If new root templated job has a diffent stage from the old one
			if jobDef.Stage == j.Stage {
				jobDef.Needs = append(jobDef.Needs, j.Needs...)
				newJobs[id] = jobDef
			}
		}
	}

	// Update needs in parent workflow with job ID from template
	for id := range run.WorkflowData.Workflow.Jobs {
		jobDef := run.WorkflowData.Workflow.Jobs[id]
		for i := range jobDef.Needs {
			if jobDef.Needs[i] == jobID {
				jobDef.Needs = slices.Delete(jobDef.Needs, i, i+1)
				jobDef.Needs = append(jobDef.Needs, finalJobs...)
				break
			}
		}
		run.WorkflowData.Workflow.Jobs[id] = jobDef
	}

	if run.WorkflowData.Workflow.Stages == nil {
		run.WorkflowData.Workflow.Stages = make(map[string]sdk.WorkflowStage)
	}
	for k, v := range newStages {
		if _, has := run.WorkflowData.Workflow.Stages[k]; !has {
			run.WorkflowData.Workflow.Stages[k] = v
		}
	}

	repoTemplate, err := repository.LoadRepositoryByID(ctx, db, templateEntity.ProjectRepositoryID)
	if err != nil {
		return nil, err
	}
	vcsTemplate, err := vcs.LoadVCSByIDAndProjectKey(ctx, db, templateEntity.ProjectKey, repoTemplate.VCSProjectID)
	if err != nil {
		return nil, err
	}
	wrefTemplate, err := NewWorkflowRunEntityFinder(ctx, db, wref.project, *run, *repoTemplate, *vcsTemplate, templateEntity.Ref, templateEntity.Commit, wref.ef.libraryProject, &wref.ef.initiator)
	if err != nil {
		return nil, err
	}
	integrations, msgs, err := wrefTemplate.checkIntegrations(ctx, db, newJobs)
	if err != nil {
		return nil, err
	}
	if len(msgs) > 0 {
		return msgs, nil
	}

	// Set job on workflow
	for k, v := range newJobs {
		run.WorkflowData.Workflow.Jobs[k] = v
		msg := retrieveAndUpdateAllJobDependencies(ctx, db, store, run, k, v, wrefTemplate, integrations, allVariableSets, defaultRegion)
		if msg != nil {
			return []sdk.V2WorkflowRunInfo{*msg}, nil
		}
	}
	// Set gate on workflow
	if run.WorkflowData.Workflow.Gates == nil {
		run.WorkflowData.Workflow.Gates = make(map[string]sdk.V2JobGate)
	}
	for k, v := range newGates {
		if _, has := run.WorkflowData.Workflow.Gates[k]; !has {
			run.WorkflowData.Workflow.Gates[k] = v
		}
	}
	// Set annotations on workflow
	if run.WorkflowData.Workflow.Annotations == nil {
		run.WorkflowData.Workflow.Annotations = make(map[string]string)
	}
	for k, v := range newAnnotations {
		if _, has := run.WorkflowData.Workflow.Annotations[k]; !has {
			run.WorkflowData.Workflow.Annotations[k] = v
		}
	}

	if run.WorkflowData.Actions == nil {
		run.WorkflowData.Actions = make(map[string]sdk.V2Action)
	}

	for k, v := range wrefTemplate.ef.actionsCache {
		if _, has := run.WorkflowData.Actions[k]; !has {
			run.WorkflowData.Actions[k] = v.Action
		}
		wref.ef.actionsCache[k] = v
	}
	for _, v := range wrefTemplate.ef.localActionsCache {
		completeName := fmt.Sprintf("%s/%s/%s/%s@%s", wrefTemplate.run.ProjectKey, wrefTemplate.ef.currentVCS.Name, wrefTemplate.ef.currentRepo.Name, v.Name, templateEntity.Ref)
		if _, has := run.WorkflowData.Actions[completeName]; !has {
			run.WorkflowData.Actions[completeName] = v.Action
		}
		wref.ef.actionsCache[completeName] = v
	}

	if run.WorkflowData.WorkerModels == nil {
		run.WorkflowData.WorkerModels = make(map[string]sdk.V2WorkerModel)
	}

	for k, v := range wrefTemplate.ef.workerModelCache {
		if _, has := run.WorkflowData.WorkerModels[k]; !has {
			run.WorkflowData.WorkerModels[k] = v.Model
		}
		wref.ef.workerModelCache[k] = v
	}
	for _, v := range wrefTemplate.ef.localWorkerModelCache {
		completeName := fmt.Sprintf("%s/%s/%s/%s@%s", wrefTemplate.run.ProjectKey, wrefTemplate.ef.currentVCS.Name, wrefTemplate.ef.currentRepo.Name, v.Model.Name, templateEntity.Ref)
		if _, has := run.WorkflowData.WorkerModels[completeName]; !has {
			run.WorkflowData.WorkerModels[completeName] = v.Model
		}
		wref.ef.workerModelCache[completeName] = v
	}
	return nil, nil
}

func searchActions(ctx context.Context, db *gorp.DbMap, store cache.Store, wref *WorkflowRunEntityFinder, steps []sdk.ActionStep) (*sdk.V2WorkflowRunInfo, error) {
	ctx, end := telemetry.Span(ctx, "searchActions")
	defer end()
	for i := range steps {
		step := &steps[i]
		if step.Uses == "" {
			continue
		}

		entityWithAction, completePath, msg, err := wref.ef.searchAction(ctx, db, store, step.Uses)
		if err != nil {
			return nil, err
		}
		if msg != "" {
			runMsg := sdk.V2WorkflowRunInfo{
				WorkflowRunID: wref.run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       msg,
			}
			return &runMsg, nil
		}
		if entityWithAction != nil {
			step.Uses = "actions/" + completePath

			// Recreate wref with vcs/repo = action's repo and not workflow's repo
			actionProject := wref.project
			actionVCS := wref.ef.currentVCS
			actionRepo := wref.ef.currentRepo
			if entityWithAction.ProjectKey != wref.project.Key {
				p, err := project.Load(ctx, db, entityWithAction.ProjectKey)
				if err != nil {
					return nil, sdk.WrapError(err, "unable to find project %s: %+v", entityWithAction.ProjectKey, *entityWithAction)
				}
				actionProject = *p
			}
			if entityWithAction.ProjectKey != wref.project.Key || entityWithAction.ProjectRepositoryID != wref.ef.currentRepo.ID {
				repo, err := repository.LoadRepositoryByID(ctx, db, entityWithAction.ProjectRepositoryID)
				if err != nil {
					return nil, err
				}
				actionRepo = *repo

				if repo.VCSProjectID != wref.ef.currentVCS.ID {
					v, err := vcs.LoadVCSByIDAndProjectKey(ctx, db, entityWithAction.ProjectKey, repo.VCSProjectID)
					if err != nil {
						return nil, err
					}
					actionVCS = *v
				}
			}

			wrefAction, err := NewWorkflowRunEntityFinder(ctx, db, actionProject, wref.run, actionRepo, actionVCS, entityWithAction.Ref, entityWithAction.Commit, wref.ef.libraryProject, &wref.ef.initiator)
			if err != nil {
				return nil, err
			}

			msgAction, err := searchActions(ctx, db, store, wrefAction, entityWithAction.Action.Runs.Steps)
			if msgAction != nil || err != nil {
				return msgAction, err
			}

			// Insert sub action in the main entity finder
			for k, v := range wrefAction.ef.localActionsCache {
				wref.ef.localActionsCache[k] = v
			}
			for k, v := range wrefAction.ef.actionsCache {
				wref.ef.actionsCache[k] = v
			}
		}
	}
	return nil, nil
}

func stopRun(ctx context.Context, db *gorp.DbMap, store cache.Store, run *sdk.V2WorkflowRun, initiator *sdk.V2Initiator, messages ...sdk.V2WorkflowRunInfo) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	status := sdk.V2WorkflowRunStatusSkipped

	for _, msg := range messages {
		if err := workflow_v2.InsertRunInfo(ctx, tx, &msg); err != nil {
			return err
		}
		if msg.Level != sdk.WorkflowRunInfoLevelWarning && status == sdk.V2WorkflowRunStatusSkipped {
			status = sdk.V2WorkflowRunStatusFail
		}
	}

	run.Status = status

	if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	event_v2.PublishRunEvent(ctx, store, sdk.EventRunEnded, *run, nil, nil, initiator)

	return nil
}

func buildRunContext(ctx context.Context, db *gorp.DbMap, store cache.Store, wr sdk.V2WorkflowRun, vcsServer sdk.VCSProject, repo sdk.ProjectRepository, uiURL string) (*sdk.WorkflowRunContext, bool, error) {
	var runContext sdk.WorkflowRunContext

	cdsContext := sdk.CDSContext{
		ProjectKey:         wr.ProjectKey,
		RunID:              wr.ID,
		RunNumber:          wr.RunNumber,
		RunAttempt:         1,
		RunURL:             fmt.Sprintf("%s/project/%s/run/%s", uiURL, wr.ProjectKey, wr.ID),
		Workflow:           wr.WorkflowName,
		WorkflowRef:        wr.WorkflowRef,
		WorkflowSha:        wr.WorkflowSha,
		WorkflowVCSServer:  vcsServer.Name,
		WorkflowRepository: repo.Name,
		Event:              wr.RunEvent.Payload,
		TriggeringActor:    wr.Initiator.Username(),
		EventName:          wr.RunEvent.EventName,
	}

	ref := wr.RunEvent.Ref
	var refType string
	if strings.HasPrefix(ref, sdk.GitRefTagPrefix) {
		refType = sdk.GitRefTypeTag
	} else {
		refType = sdk.GitRefTypeBranch
	}
	commit := wr.RunEvent.Sha
	var semverCurrent string
	semverNext := wr.RunEvent.SemverNext
	currentVersion, _ := semver.NewVersion(wr.RunEvent.SemverCurrent)

	if currentVersion != nil {
		if currentVersion.Metadata() == "" { // Tags doesn't have metadata
			semverCurrent = wr.RunEvent.SemverCurrent
		} else {
			suffix := currentVersion.Metadata()
			splittedSuffix := strings.Split(suffix, ".")
			var metadataStr string
			if len(splittedSuffix) >= 2 {
				metadataStr += splittedSuffix[0] + ".sha." + sdk.StringFirstN(splittedSuffix[1], 8)
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
		semverCurrent = "0.1.0+" + strconv.FormatInt(wr.RunNumber, 10) + ".sha.g" + sdk.StringFirstN(commit, 7)
	}

	semverNext = strings.ReplaceAll(semverNext, "+", "-")
	semverCurrent = strings.ReplaceAll(semverCurrent, "+", "-")

	// Reload VCS with decryption to have sshkey / git username
	vcsName := vcsServer.Name
	if wr.WorkflowData.Workflow.Repository != nil && wr.WorkflowData.Workflow.Repository.VCSServer != vcsServer.Name {
		vcsName = wr.WorkflowData.Workflow.Repository.VCSServer
	}
	vcsTmp, err := vcs.LoadVCSByProject(ctx, db, wr.ProjectKey, vcsName, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return nil, false, err
	}
	workflowVCSServer := *vcsTmp

	gitContext := sdk.GitContext{
		Server:               workflowVCSServer.Name,
		RepositoryOrigin:     wr.RunEvent.RepositoryOrigin,
		SSHKey:               workflowVCSServer.Auth.SSHKeyName,
		GPGKey:               workflowVCSServer.Auth.GPGKeyName,
		Username:             workflowVCSServer.Auth.Username,
		Email:                workflowVCSServer.Auth.EmailAddress,
		Ref:                  ref,
		RefName:              strings.TrimPrefix(strings.TrimPrefix(ref, sdk.GitRefBranchPrefix), sdk.GitRefTagPrefix),
		RefType:              refType,
		Sha:                  commit,
		CommitMessage:        wr.RunEvent.CommitMessage,
		Author:               wr.RunEvent.CommitAuthor,
		AuthorEmail:          wr.RunEvent.CommitAuthorEmail,
		SemverCurrent:        semverCurrent,
		SemverNext:           semverNext,
		ChangeSets:           wr.RunEvent.ChangeSets,
		PullRequestID:        wr.RunEvent.PullRequestID,
		PullRequestToRef:     wr.RunEvent.PullRequestToRef,
		PullRequestToRefName: strings.TrimPrefix(strings.TrimPrefix(wr.RunEvent.PullRequestToRef, sdk.GitRefBranchPrefix), sdk.GitRefTagPrefix),
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

	vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, store, wr.ProjectKey, workflowVCSServer.Name)
	if err != nil {
		return nil, false, err
	}
	vcsRepo, err := vcsClient.RepoByFullname(ctx, gitContext.Repository)
	if err != nil {
		return nil, false, err
	}
	gitContext.RepositoryWebURL = vcsRepo.URL

	if gitContext.SSHKey != "" {
		gitContext.RepositoryURL = vcsRepo.SSHCloneURL
	} else {
		gitContext.RepositoryURL = vcsRepo.HTTPCloneURL
	}

	if gitContext.Ref == "" {
		defaultBranch, err := vcsClient.Branch(ctx, gitContext.Repository, sdk.VCSBranchFilters{Default: true})
		if err != nil {
			return nil, false, err
		}
		gitContext.Ref = defaultBranch.ID
		gitContext.RefName = defaultBranch.DisplayID
		gitContext.RefType = sdk.GitRefTypeBranch
		if gitContext.Sha == "" {
			gitContext.Sha = defaultBranch.LatestCommit
		}
	}

	if gitContext.Sha == "" {
		switch {
		case strings.HasPrefix(gitContext.Ref, sdk.GitRefBranchPrefix):
			b, err := vcsClient.Branch(ctx, gitContext.Repository, sdk.VCSBranchFilters{BranchName: strings.TrimPrefix(gitContext.Ref, sdk.GitRefBranchPrefix)})
			if err != nil {
				return nil, false, err
			}
			gitContext.Sha = b.LatestCommit
		case strings.HasPrefix(gitContext.Ref, sdk.GitRefTagPrefix):
			t, err := vcsClient.Tag(ctx, gitContext.Repository, strings.TrimPrefix(gitContext.Ref, sdk.GitRefTagPrefix))
			if err != nil {
				return nil, false, err
			}
			gitContext.Sha = t.Hash
		}
	}
	if len(gitContext.Sha) > 7 {
		gitContext.ShaShort = gitContext.Sha[0:7]
	}

	gitContext.CommitWebURL = fmt.Sprintf(vcsRepo.URLCommitFormat, gitContext.Sha)
	switch gitContext.RefType {
	case sdk.GitRefTypeBranch:
		gitContext.RefWebURL = fmt.Sprintf(vcsRepo.URLBranchFormat, gitContext.RefName)
	case sdk.GitRefTypeTag:
		gitContext.RefWebURL = fmt.Sprintf(vcsRepo.URLTagFormat, gitContext.RefName)
	}

	// Env context
	envs := make(map[string]string)
	for k, v := range wr.WorkflowData.Workflow.Env {
		envs[k] = v
	}

	runContext.CDS = cdsContext
	runContext.Git = gitContext
	runContext.Env = envs

	mustSaveVersion := false
	if wr.WorkflowData.Workflow.Semver != nil {
		var cdsVersion *semver.Version
		cdsVersion, mustSaveVersion, err = getCDSversion(ctx, db, vcsClient, workflowVCSServer.Type, runContext, wr.WorkflowData.Workflow)
		if err != nil {
			return nil, false, err
		}
		runContext.CDS.Version = cdsVersion.String()
		runContext.CDS.VersionNext = cdsVersion.IncMinor().String()
	} else {
		runContext.CDS.Version = gitContext.SemverCurrent
		runContext.CDS.VersionNext = gitContext.SemverNext
	}

	return &runContext, mustSaveVersion, nil
}

func getCDSversion(ctx context.Context, db gorp.SqlExecutor, vcsClient sdk.VCSAuthorizedClientService, typeVCS string, runContext sdk.WorkflowRunContext, workflowDef sdk.V2Workflow) (*semver.Version, bool, error) {
	// If not git, retrieve file
	var content sdk.VCSContent
	if workflowDef.Semver.From != sdk.SemverTypeGit {
		filePath := strings.TrimPrefix(filepath.Clean(workflowDef.Semver.Path), "/")
		var err error
		content, err = vcsClient.GetContent(ctx, runContext.Git.Repository, runContext.Git.Sha, filePath)
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, false, sdk.NewErrorFrom(sdk.ErrInvalidData, "file %s doesn't not exist on commit %s", filePath, runContext.Git.Sha)
			}
			return nil, false, err
		}
		if !content.IsFile {
			return nil, false, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to compute cds version: file not found wih path %s", workflowDef.Semver.Path)
		}
		if content.Content == "" {
			return nil, false, sdk.NewErrorFrom(sdk.ErrInvalidData, "the file found %s is empty", workflowDef.Semver.Path)
		}
	}

	var fileContent string
	switch typeVCS {
	case sdk.VCSTypeGitlab, sdk.VCSTypeGithub, sdk.VCSTypeGitea:
		contentBts, err := base64.StdEncoding.DecodeString(content.Content)
		if err != nil {
			return nil, false, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to decode file at path %s", workflowDef.Semver.Path)
		}
		fileContent = string(contentBts)
	default:
		fileContent = content.Content
	}

	// Retrieve version from file
	var fileVersion string
	var cdsVersion string
	switch workflowDef.Semver.From {
	case sdk.SemverTypeGit:
		v, _ := semver.NewVersion(runContext.Git.SemverCurrent)
		fileVersion = fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
		if runContext.Git.RefType == sdk.GitRefTypeTag {
			cdsVersion = runContext.Git.SemverCurrent
		}
	case sdk.SemverTypeHelm:
		var file sdk.SemverHelmChart
		if err := yaml.Unmarshal([]byte(fileContent), &file); err != nil {
			return nil, false, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read helm chart file: %v", err)
		}
		fileVersion = file.Version
	case sdk.SemverTypeCargo:
		var file sdk.SemverCargoFile
		if err := toml.Unmarshal([]byte(fileContent), &file); err != nil {
			return nil, false, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read cargo file: %v", err)
		}
		fileVersion = file.Package.Version
	case sdk.SemverTypeNpm, sdk.SemverTypeYarn:
		var file sdk.SemverNpmYarnPackage
		if err := json.Unmarshal([]byte(fileContent), &file); err != nil {
			return nil, false, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read npm/yarn file: %v", err)
		}
		fileVersion = file.Version
	case sdk.SemverTypeFile:
		fileVersion = strings.Split(fileContent, "\n")[0]
	case sdk.SemverTypePoetry:
		var file sdk.SemverPoetry
		if err := toml.Unmarshal([]byte(fileContent), &file); err != nil {
			return nil, false, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read poetry file: %v", err)
		}
		fileVersion = file.Tool.Poetry.Version
	case sdk.SemverTypeDebian:
		firsLine := strings.Split(fileContent, "\n")[0]
		r, _ := regexp.Compile(`.*\((.*)\).*`) // format: package (version) distribution; urgency=low
		result := r.FindStringSubmatch(firsLine)
		if r.NumSubexp() == 0 {
			return nil, false, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to extract version from [%s]", firsLine)
		}
		fileVersion = result[1]
	default:
		return nil, false, sdk.NewErrorFrom(sdk.ErrInvalidData, "the semver type %s not managed", workflowDef.Semver.From)
	}

	isReleaseRef := false
	if workflowDef.Semver.From != sdk.SemverTypeGit {
		// Check defaultBranch
		if len(workflowDef.Semver.ReleaseRefs) == 0 {
			defaultBranch, err := vcsClient.Branch(ctx, runContext.Git.Repository, sdk.VCSBranchFilters{Default: true})
			if err != nil {
				return nil, false, err
			}
			isReleaseRef = defaultBranch.ID == runContext.Git.Ref
		} else {
			for _, r := range workflowDef.Semver.ReleaseRefs {
				g := glob.New(r)
				result, err := g.MatchString(runContext.Git.Ref)
				if err != nil {
					return nil, false, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to check release ref with pattern %s: %v", r, err)
				}
				if result == nil {
					continue
				}
				isReleaseRef = true
			}
		}
	}

	mustSaveVersion := false
	if isReleaseRef {
		// Check if the release exists
		_, err := workflow_v2.LoadWorkflowVersion(ctx, db, runContext.CDS.ProjectKey, runContext.CDS.WorkflowVCSServer, runContext.CDS.WorkflowRepository, runContext.CDS.Workflow, fileVersion)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, false, err
		}
		if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
			cdsVersion = fileVersion
			mustSaveVersion = true
		}
	}

	// If not a release ref OR release version already exists
	// Compute cds version from a pattern
	if cdsVersion == "" {
		// Retrieve version pattern
		cdsDefaultPattern := fmt.Sprintf(sdk.DefaultVersionPattern, workflowDef.Semver.From)
		var pattern string
		if workflowDef.Semver.Schema != nil {
			// Check non default pattern
			for k, v := range workflowDef.Semver.Schema {
				if k != "**/*" {
					g := glob.New(k)
					result, err := g.MatchString(runContext.Git.Ref)
					if err != nil {
						return nil, false, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to check branch with pattern %s: %v", k, err)
					}
					if result != nil {
						pattern = v
						break
					}
				}
			}
		}
		// If no pattern found, check default pattern
		if pattern == "" {
			if p, has := workflowDef.Semver.Schema["**/*"]; has {
				pattern = p
			} else {
				pattern = cdsDefaultPattern
			}

		}

		// Compute new dev version
		bts, err := json.Marshal(runContext)
		if err != nil {
			return nil, false, sdk.WithStack(err)
		}
		var mapContexts map[string]interface{}
		if err := json.Unmarshal(bts, &mapContexts); err != nil {
			return nil, false, sdk.WithStack(err)
		}
		if workflowDef.Semver.From != sdk.SemverTypeGit {
			versionContext := map[string]interface{}{
				"version": fileVersion,
			}
			mapContexts[string(workflowDef.Semver.From)] = versionContext
		} else {
			gitMap := mapContexts["git"].(map[string]interface{})
			gitMap["version"] = fileVersion
		}

		ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
		cdsVersion, err = ap.InterpolateToString(ctx, pattern)
		if err != nil {
			return nil, false, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to compute version from pattern %s: %v", pattern, err)
		}
	}

	// Check semver
	semverVersion, err := semver.NewVersion(cdsVersion)
	if err != nil {
		return nil, false, sdk.NewErrorFrom(sdk.ErrInvalidData, "the computed version %s is not semver compatible: %v", cdsVersion, err)
	}
	return semverVersion, mustSaveVersion, nil
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
		wm, fullName, msg, err := wref.ef.searchWorkerModel(ctx, db, store, workerModel)
		if err != nil {
			msg := sdk.V2WorkflowRunInfo{
				WorkflowRunID: wref.run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("Unable to find worker model %s", workerModel),
			}
			return "", &msg, nil
		}
		if msg != "" {
			runMsg := sdk.V2WorkflowRunInfo{
				WorkflowRunID: wref.run.ID,
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       msg,
			}
			return "", &runMsg, nil
		}
		if err := wm.Interpolate(ctx); err != nil {
			return "", nil, err
		}
		if strings.HasPrefix(workerModel, ".cds/worker-models/") {
			wref.ef.localWorkerModelCache[workerModel] = *wm
		} else {
			wref.ef.workerModelCache[fullName] = *wm
		}
		modelType = wm.Model.Type
		modelCompleteName = fullName

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
		Message:       fmt.Sprintf("wrong configuration on job %q. No hatchery can run it with model [%s]", jobName, modelCompleteName),
	}, nil
}

func (wref *WorkflowRunEntityFinder) checkIntegrations(ctx context.Context, db *gorp.DbMap, jobs map[string]sdk.V2Job) (map[string]sdk.ProjectIntegration, []sdk.V2WorkflowRunInfo, error) {
	availableIntegrations, err := integration.LoadIntegrationsByProjectID(ctx, db, wref.project.ID)
	if err != nil {
		return nil, nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to load integration")
	}

	var infos []sdk.V2WorkflowRunInfo
	var integrations = map[string]sdk.ProjectIntegration{}

	for i := range wref.run.WorkflowData.Workflow.Integrations {
		var found bool
		for j := range availableIntegrations {
			if wref.run.WorkflowData.Workflow.Integrations[i] == availableIntegrations[j].Name {
				if availableIntegrations[j].Model.ArtifactManager {
					if exiting, has := integrations[wref.run.WorkflowData.Workflow.Integrations[i]]; has && wref.run.WorkflowData.Workflow.Integrations[i] != exiting.Name {
						infos = append(infos, sdk.V2WorkflowRunInfo{
							WorkflowRunID: wref.run.ID,
							Level:         sdk.WorkflowRunInfoLevelError,
							IssuedAt:      time.Now(),
							Message:       fmt.Sprintf("wrong workflow configuration. Only one artifact manager Integration %s is allowed", wref.run.WorkflowData.Workflow.Integrations[i]),
						})
					}
					integrations[wref.run.WorkflowData.Workflow.Integrations[i]] = availableIntegrations[j]
				}
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

	for jobID, job := range jobs {
		for _, integ := range job.Integrations {
			if !strings.Contains(integ, "${{") {
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
						integrations[integ] = availableIntegrations[j]
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
	}

	return integrations, infos, nil
}

func (wref *WorkflowRunEntityFinder) checkWorkflowTemplate(ctx context.Context, db *gorp.DbMap, store cache.Store, templateName string) (sdk.EntityWithObject, *sdk.V2WorkflowRunInfo, error) {
	ctx, next := telemetry.Span(ctx, "wref.checkWorkflowTemplate", trace.StringAttribute(telemetry.TagWorkflowTemplate, templateName))
	defer next()

	e, _, msg, err := wref.ef.searchWorkflowTemplate(ctx, db, store, templateName)
	if err != nil {
		return sdk.EntityWithObject{}, nil, err
	}
	if msg != "" {
		runMsg := sdk.V2WorkflowRunInfo{
			WorkflowRunID: wref.run.ID,
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       msg,
		}
		return sdk.EntityWithObject{}, &runMsg, nil
	}
	return *e, nil, nil
}
