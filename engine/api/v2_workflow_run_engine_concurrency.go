package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

// Call when a workflow run or run job ends
func (api *API) manageEndConcurrency(projectKey, vcsServer, repository, workflowName, parentRunID, currentObjectID string, concurrencyDef *sdk.V2RunConcurrency) {
	if concurrencyDef == nil {
		return
	}

	api.GoRoutines.Exec(context.Background(), "manageEndJobConcurrency."+currentObjectID, func(ctx context.Context) {
		ctx = context.WithValue(ctx, cdslog.WorkflowRunID, parentRunID)
		ctx = context.WithValue(ctx, cdslog.ConcurrencyName, concurrencyDef.Name)
		ctx = context.WithValue(ctx, cdslog.ConcurrencyScope, concurrencyDef.Scope)

		log.Info(ctx, "checking concurrency unlock for %s %s", concurrencyDef.Name, concurrencyDef.Scope)

		runConcurrentObjs, _, err := retrieveRunObjectsToUnLocked(ctx, api.mustDB(), projectKey, vcsServer, repository, workflowName, *concurrencyDef)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return
		}

		// Enqueue workflow
		for _, rco := range runConcurrentObjs {
			var runID, workflowName string
			var runNumber int64
			var initiator sdk.V2Initiator
			var rj *sdk.V2WorkflowRunJob
			var run *sdk.V2WorkflowRun
			switch rco.Type {
			case workflow_v2.ConcurrencyObjectTypeWorkflow:
				run, err = workflow_v2.LoadRunByID(ctx, api.mustDBWithCtx(ctx), rco.ID)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					continue
				}
				runID = run.ID
				workflowName = run.WorkflowName
				runNumber = run.RunNumber
				initiator = *run.Initiator
			default:
				rj, err = workflow_v2.LoadRunJobByID(ctx, api.mustDBWithCtx(ctx), rco.ID)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					continue
				}
				runID = rj.WorkflowRunID
				workflowName = rj.WorkflowName
				runNumber = rj.RunNumber
				initiator = rj.Initiator
			}

			api.EnqueueWorkflowRun(ctx, runID, initiator, workflowName, runNumber)

			tx, err := api.mustDB().Begin()
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				return
			}
			defer tx.Rollback()

			var msg string
			switch rco.Type {
			case workflow_v2.ConcurrencyObjectTypeWorkflow:
				msg = fmt.Sprintf("Unlocking workflow %s/%s/%s on run %d for concurrency '%s'", run.VCSServer, run.Repository, run.WorkflowName, run.RunNumber, run.Concurrency.Name)
			default:
				msg = fmt.Sprintf("Unlocking job %s on workflow %s/%s/%s on run %d for concurrency '%s'", rj.JobID, rj.VCSServer, rj.Repository, rj.WorkflowName, rj.RunNumber, rj.Concurrency.Name)
			}

			if parentRunID == currentObjectID {
				runInfo := sdk.V2WorkflowRunInfo{
					WorkflowRunID: parentRunID,
					IssuedAt:      time.Now(),
					Level:         sdk.WorkflowRunInfoLevelInfo,
					Message:       msg,
				}
				if err := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return
				}
			} else {
				runJobInfo := sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    parentRunID,
					WorkflowRunJobID: currentObjectID,
					IssuedAt:         time.Now(),
					Level:            sdk.WorkflowRunInfoLevelInfo,
					Message:          msg,
				}
				if err := workflow_v2.InsertRunJobInfo(ctx, tx, &runJobInfo); err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return
				}
			}

			if err := tx.Commit(); err != nil {
				log.ErrorWithStackTrace(ctx, err)
				return
			}
		}
	})
}

func (api *API) enqueueCancelledRunObjects(ctx context.Context, runs []sdk.V2WorkflowRun, runJobs []sdk.V2WorkflowRunJob) {
	for _, run := range runs {
		api.EnqueueWorkflowRunWithStatus(ctx, run.ID, *run.Initiator, run.WorkflowName, run.RunNumber, sdk.V2WorkflowRunStatusCancelled)
	}
	for _, rj := range runJobs {
		api.EnqueueWorkflowRun(ctx, rj.WorkflowRunID, rj.Initiator, rj.WorkflowName, rj.RunNumber)
		runToTrigger, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), rj.WorkflowRunID)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobCancelled, *runToTrigger, rj)
	}
}

// Update new run job with concurrency data, check if we have to lock it
func manageJobConcurrency(ctx context.Context, db *gorp.DbMap, run sdk.V2WorkflowRun, jobID string, runJob *sdk.V2WorkflowRunJob, concurrenciesDef map[string]sdk.V2RunConcurrency, concurrencyUnlockedCount map[string]int64, toCancelled map[string]workflow_v2.ConcurrencyObject) (*sdk.V2WorkflowRunJobInfo, error) {
	if runJob.Job.Concurrency != "" {
		concurrencyDef, has := concurrenciesDef[jobID]
		if !has {
			runJob.Status = sdk.V2WorkflowRunJobStatusFail
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				WorkflowRunJobID: runJob.ID,
				IssuedAt:         time.Now(),
				Level:            sdk.WorkflowRunInfoLevelError,
				Message:          fmt.Sprintf("job %s: concurrency %s not found on workflow or on project", jobID, runJob.Job.Concurrency),
			}, nil
		}

		// Set concurrency on runjob
		runJob.Concurrency = &concurrencyDef

		// Retrieve current building and blocked job to check if we can enqueue this one
		canRun, err := canRunWithConcurrency(ctx, concurrencyDef, db, run, workflow_v2.ConcurrencyObject{ID: runJob.ID, Type: workflow_v2.ConcurrencyObjectTypeJob}, concurrencyUnlockedCount, toCancelled)
		if err != nil {
			return nil, err
		}

		// If job has to be cancelled
		if _, has := toCancelled[runJob.ID]; has {
			runJob.Status = sdk.V2WorkflowRunJobStatusCancelled
			delete(toCancelled, runJob.ID)
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    runJob.WorkflowRunID,
				WorkflowRunJobID: runJob.ID,
				Level:            sdk.WorkflowRunInfoLevelInfo,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job cancelled due to concurrency %q", runJob.Concurrency.Name),
			}, nil
		}

		if !canRun {
			runJob.Status = sdk.V2WorkflowRunJobStatusBlocked
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    runJob.WorkflowRunID,
				WorkflowRunJobID: runJob.ID,
				Level:            sdk.WorkflowRunInfoLevelInfo,
				IssuedAt:         time.Now(),
				Message:          fmt.Sprintf("Job locked due to concurrency %q", runJob.Concurrency.Name),
			}, nil
		}

		for _, runObj := range toCancelled {
			if runObj.Type == workflow_v2.ConcurrencyObjectTypeWorkflow {
				runJob.Status = sdk.V2WorkflowRunJobStatusBlocked
				return &sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    runJob.WorkflowRunID,
					WorkflowRunJobID: runJob.ID,
					Level:            sdk.WorkflowRunInfoLevelInfo,
					IssuedAt:         time.Now(),
					Message:          "Job blocked, waiting for workfow cancellation before starting",
				}, nil
			}
		}
	}
	return nil, nil
}

// Update new workflow run check if we have to lock it
func manageWorkflowConcurrency(ctx context.Context, db *gorp.DbMap, run *sdk.V2WorkflowRun, concurrencyUnlockedCount map[string]int64, toCancel map[string]workflow_v2.ConcurrencyObject) (*sdk.V2WorkflowRunInfo, error) {
	if run.WorkflowData.Workflow.Concurrency != "" {
		concurrencyDef, err := retrieveConcurrencyDefinition(ctx, db, *run, run.WorkflowData.Workflow.Concurrency)
		if err != nil {
			return nil, err
		}
		if concurrencyDef == nil {
			runInfo := sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("concurrency %q not found on workflow nor on project", run.WorkflowData.Workflow.Concurrency),
			}
			return &runInfo, nil
		}
		run.Concurrency = concurrencyDef

		// Retrieve current building and blocked job to check if we can enqueue this one
		canRun, err := canRunWithConcurrency(ctx, *concurrencyDef, db, *run, workflow_v2.ConcurrencyObject{ID: run.ID, Type: workflow_v2.ConcurrencyObjectTypeWorkflow}, concurrencyUnlockedCount, toCancel)
		if err != nil {
			return nil, err
		}

		if _, has := toCancel[run.ID]; has {
			run.Status = sdk.V2WorkflowRunStatusCancelled
			delete(toCancel, run.ID)
			return &sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelInfo,
				IssuedAt:      time.Now(),
				Message:       fmt.Sprintf("Workflow cancelled due to concurrency %q", run.Concurrency.Name),
			}, nil
		}

		if !canRun {
			run.Status = sdk.V2WorkflowRunStatusBlocked
			return &sdk.V2WorkflowRunInfo{
				WorkflowRunID: run.ID,
				Level:         sdk.WorkflowRunInfoLevelInfo,
				IssuedAt:      time.Now(),
				Message:       fmt.Sprintf("Workflow locked due to concurrency %q", run.Concurrency.Name),
			}, nil
		}

		for _, runObj := range toCancel {
			if runObj.Type == workflow_v2.ConcurrencyObjectTypeWorkflow {
				run.Status = sdk.V2WorkflowRunStatusBlocked
				return &sdk.V2WorkflowRunInfo{
					WorkflowRunID: run.ID,
					Level:         sdk.WorkflowRunInfoLevelInfo,
					IssuedAt:      time.Now(),
					Message:       "Job blocked, waiting for workfow cancellation before starting",
				}, nil
			}
		}
	}
	return nil, nil
}

func canRunWithConcurrency(ctx context.Context, concurrencyDef sdk.V2RunConcurrency, db *gorp.DbMap, run sdk.V2WorkflowRun, currentConcurrencyObject workflow_v2.ConcurrencyObject, concurrencyUnlockedCount map[string]int64, toCancelled map[string]workflow_v2.ConcurrencyObject) (bool, error) {
	var ruleToApply *sdk.V2RunConcurrency
	var nbRunJobBuilding, nbRunJobBlocked int64
	var err error
	if concurrencyDef.Scope == sdk.V2RunConcurrencyScopeProject {
		ruleToApply, nbRunJobBuilding, nbRunJobBlocked, err = checkProjectScopedConcurrency(ctx, db, run.ProjectKey, concurrencyDef.WorkflowConcurrency)
	} else {
		ruleToApply, nbRunJobBuilding, nbRunJobBlocked, err = checkWorkflowScopedConcurrency(ctx, db, run.ProjectKey, run.VCSServer, run.Repository, run.WorkflowName, concurrencyDef.WorkflowConcurrency)
	}
	if err != nil {
		return false, err
	}

	poolIsFull := nbRunJobBlocked+nbRunJobBuilding+concurrencyUnlockedCount[concurrencyDef.Name] >= ruleToApply.Pool

	if !ruleToApply.CancelInProgress {
		if poolIsFull {
			return false, nil
		} else {
			if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst && nbRunJobBlocked > 0 {
				return false, nil
			}
		}
	} else {
		// Manage cancel-in-progress
		objectsToCancelled, err := retrieveConcurrencyObjectToCancelled(ctx, db, run.ProjectKey, run.VCSServer, run.Repository, run.WorkflowName, currentConcurrencyObject, ruleToApply, nbRunJobBuilding, concurrencyUnlockedCount[concurrencyDef.Name])
		if err != nil {
			return false, err
		}
		for _, obj := range objectsToCancelled {
			toCancelled[obj.ID] = obj
		}
	}

	_, has := toCancelled[currentConcurrencyObject.ID]
	if has {
		return false, nil
	}
	concurrencyUnlockedCount[concurrencyDef.Name]++
	return true, nil
}

// Retrieve the next rj to unblocked.
func retrieveRunObjectsToUnLocked(ctx context.Context, db *gorp.DbMap, projKey, vcsServer, repository, workflowName string, concurrencyDef sdk.V2RunConcurrency) ([]workflow_v2.ConcurrencyObject, []workflow_v2.ConcurrencyObject, error) {
	var ruleToApply *sdk.V2RunConcurrency
	var nbBuilding int64
	var err error
	switch concurrencyDef.Scope {
	case sdk.V2RunConcurrencyScopeProject:
		ruleToApply, nbBuilding, _, err = checkProjectScopedConcurrency(ctx, db, projKey, concurrencyDef.WorkflowConcurrency)
		if err != nil {
			return nil, nil, err
		}
	default:
		ruleToApply, nbBuilding, _, err = checkWorkflowScopedConcurrency(ctx, db, projKey, vcsServer, repository, workflowName, concurrencyDef.WorkflowConcurrency)
		if err != nil {
			return nil, nil, err
		}
	}

	if ruleToApply == nil {
		msg := fmt.Sprintf("unable to retrieve concurreny rule %s scope %s for workflow %s", concurrencyDef.Name, concurrencyDef.Scope, workflowName)
		log.Error(ctx, msg)
		return nil, nil, sdk.NewErrorFrom(sdk.ErrInvalidData, msg)
	}

	// If cancel in progress, just retrieve the newest rjs to start
	if ruleToApply.CancelInProgress {
		var concurrencyRunObjects []workflow_v2.ConcurrencyObject
		switch concurrencyDef.Scope {
		case sdk.V2RunConcurrencyScopeProject:
			concurrencyRunObjects, err = workflow_v2.LoadNewestRunJobWithProjectScopedConcurrency(ctx, db, projKey, concurrencyDef.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, sdk.V2WorkflowRunStatusBlocked, nil)
		default:
			concurrencyRunObjects, err = workflow_v2.LoadNewestRunJobWithWorkflowScopedConcurrency(ctx, db, projKey, vcsServer, repository, workflowName, concurrencyDef.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, sdk.V2WorkflowRunStatusBlocked, nil)
		}
		if err != nil {
			return nil, nil, err
		}
		runObjectsToUnlock := make([]workflow_v2.ConcurrencyObject, 0)
		runObjectsToCancel := make([]workflow_v2.ConcurrencyObject, 0)
		nbToUnlocked := ruleToApply.Pool - nbBuilding

		// All can be unlocked
		if nbToUnlocked >= int64(len(concurrencyRunObjects)) {
			runObjectsToUnlock = append(runObjectsToUnlock, concurrencyRunObjects...)
		} else {
			runObjectsToUnlock = append(runObjectsToUnlock, concurrencyRunObjects[:nbToUnlocked]...)
			for i := nbToUnlocked; i < int64(len(concurrencyRunObjects)); i++ {
				runObjectsToCancel = append(runObjectsToCancel, concurrencyRunObjects[i])
			}
		}
		return runObjectsToUnlock, runObjectsToCancel, nil
	}

	// No cancel in progress

	// Test if pool is full
	if ruleToApply.Pool <= nbBuilding {
		return nil, nil, nil
	}

	nbToUnlocked := ruleToApply.Pool - nbBuilding
	toUnlocked := make([]workflow_v2.ConcurrencyObject, 0)
	switch concurrencyDef.Scope {
	case sdk.V2RunConcurrencyScopeProject:
		if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst {
			// Load oldest
			runObjToUnlock, err := workflow_v2.LoadOldestRunJobWithProjectScopedConcurrency(ctx, db, projKey, concurrencyDef.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, sdk.V2WorkflowRunStatusBlocked, nbToUnlocked)
			if err != nil {
				return nil, nil, err
			}
			toUnlocked = append(toUnlocked, runObjToUnlock...)
		} else {
			// Load newest
			rjs, err := workflow_v2.LoadNewestRunJobWithProjectScopedConcurrency(ctx, db, projKey, concurrencyDef.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, sdk.V2WorkflowRunStatusBlocked, nbToUnlocked)
			if err != nil {
				return nil, nil, err
			}
			toUnlocked = append(toUnlocked, rjs...)
		}
	default:
		if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst {
			// Load oldest
			runObjToUnlock, err := workflow_v2.LoadOldestRunJobWithWorkflowScopedConcurrency(ctx, db, projKey, vcsServer, repository, workflowName, concurrencyDef.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, sdk.V2WorkflowRunStatusBlocked, nbToUnlocked)
			if err != nil {
				return nil, nil, err
			}
			toUnlocked = append(toUnlocked, runObjToUnlock...)
		} else {
			// Load newest
			var err error
			runObjToUnlock, err := workflow_v2.LoadNewestRunJobWithWorkflowScopedConcurrency(ctx, db, projKey, vcsServer, repository, workflowName, concurrencyDef.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, sdk.V2WorkflowRunStatusBlocked, nbToUnlocked)
			if err != nil {
				return nil, nil, err
			}
			toUnlocked = append(toUnlocked, runObjToUnlock...)
		}
	}
	return toUnlocked, nil, nil
}

func retrieveConcurrencyObjectToCancelled(ctx context.Context, db gorp.SqlExecutor, projKey, vcs, repo, workflow string, currentRunObject workflow_v2.ConcurrencyObject, ruleToApply *sdk.V2RunConcurrency, nbBuilding, currentOnSameRule int64) ([]workflow_v2.ConcurrencyObject, error) {
	nbToCancelled := nbBuilding + currentOnSameRule - ruleToApply.Pool + 1
	toCancel := make([]workflow_v2.ConcurrencyObject, 0)

	if nbToCancelled <= 0 {
		return nil, nil
	}

	// First cancel building jobs
	var err error
	var buildingObjects []workflow_v2.ConcurrencyObject
	switch ruleToApply.Scope {
	case sdk.V2RunConcurrencyScopeProject:
		buildingObjects, err = workflow_v2.LoadOldestRunJobWithProjectScopedConcurrency(ctx, db, projKey, ruleToApply.Name, []string{string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}, sdk.V2WorkflowRunStatusBuilding, nbToCancelled)
	default:
		buildingObjects, err = workflow_v2.LoadOldestRunJobWithWorkflowScopedConcurrency(ctx, db, projKey, vcs, repo, workflow, ruleToApply.Name, []string{string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}, sdk.V2WorkflowRunStatusBuilding, nbToCancelled)
	}
	if err != nil {
		return nil, err
	}
	toCancel = append(toCancel, buildingObjects...)

	nbToCancelled -= int64(len(toCancel))

	if nbToCancelled > 0 {
		// Then cancel blocked jobs
		var blockedObjects []workflow_v2.ConcurrencyObject
		switch ruleToApply.Scope {
		case sdk.V2RunConcurrencyScopeProject:
			blockedObjects, err = workflow_v2.LoadOldestRunJobWithProjectScopedConcurrency(ctx, db, projKey, ruleToApply.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, sdk.V2WorkflowRunStatusBlocked, nbToCancelled)
		default:
			blockedObjects, err = workflow_v2.LoadOldestRunJobWithWorkflowScopedConcurrency(ctx, db, projKey, vcs, repo, workflow, ruleToApply.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, sdk.V2WorkflowRunStatusBlocked, nbToCancelled)
		}
		if err != nil {
			return nil, err
		}
		toCancel = append(toCancel, blockedObjects...)
		nbToCancelled -= int64(len(blockedObjects))

		// Then cancel the current job
		if nbToCancelled > 0 {
			toCancel = append(toCancel, currentRunObject)
		}
	}

	return toCancel, nil
}

// Retrieve rule for a project scoped concurrency
func checkProjectScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, projKey string, currentConcurrencyDef sdk.WorkflowConcurrency) (*sdk.V2RunConcurrency, int64, int64, error) {
	nbRunJobBuilding, err := workflow_v2.CountRunningWithProjectConcurrency(ctx, db, projKey, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}
	nbBlockedRunJobs, err := workflow_v2.CountBlockedRunWithProjectConcurrency(ctx, db, projKey, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}

	// Check if rules are differents between ongoing job run
	ongoingRules, err := workflow_v2.LoadProjectConcurrencyRules(ctx, db, projKey, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}
	ruleToApply := mergeConcurrencyRules(ongoingRules, currentConcurrencyDef)
	return &sdk.V2RunConcurrency{WorkflowConcurrency: ruleToApply, Scope: sdk.V2RunConcurrencyScopeProject}, nbRunJobBuilding, nbBlockedRunJobs, nil
}

// Retrieve rule for a workflow scoped concurrency
func checkWorkflowScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, projKey, vcsName, repo, workflowName string, currentConcurrencyDef sdk.WorkflowConcurrency) (*sdk.V2RunConcurrency, int64, int64, error) {
	nbRunJobBuilding, err := workflow_v2.CountRunningWithWorkflowConcurrency(ctx, db, projKey, vcsName, repo, workflowName, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}
	nbBlockedRunJobs, err := workflow_v2.CountBlockedWithWorkflowConcurrency(ctx, db, projKey, vcsName, repo, workflowName, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}

	// Check if rules are differents between ongoing job run
	ongoingRules, err := workflow_v2.LoadWorkflowConcurrencyRules(ctx, db, projKey, vcsName, repo, workflowName, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}
	ruleToApply := mergeConcurrencyRules(ongoingRules, currentConcurrencyDef)

	return &sdk.V2RunConcurrency{WorkflowConcurrency: ruleToApply, Scope: sdk.V2RunConcurrencyScopeWorkflow}, nbRunJobBuilding, nbBlockedRunJobs, nil
}

// Merge concurrency rule between all running jobs
// Default is ConcurrencyOrderOldestFirst + min(pool) + CancelInProgress = false
func mergeConcurrencyRules(rulesInDB []workflow_v2.ConcurrencyRule, refRule sdk.WorkflowConcurrency) sdk.WorkflowConcurrency {
	mergedRule := refRule
	if len(rulesInDB) > 0 {
		for _, r := range rulesInDB {
			// Default behaviours if there is multiple configuration
			if r.Order != string(mergedRule.Order) {
				mergedRule.Order = sdk.ConcurrencyOrderOldestFirst
			}
			if r.MinPool < mergedRule.Pool {
				mergedRule.Pool = r.MinPool
			}
			if r.Cancel != mergedRule.CancelInProgress {
				mergedRule.CancelInProgress = false
			}
		}
	}
	return mergedRule
}

func retrieveConcurrencyDefinition(ctx context.Context, db gorp.SqlExecutor, run sdk.V2WorkflowRun, concurrencyName string) (*sdk.V2RunConcurrency, error) {
	// Search concurrency rule on workflow
	scope := sdk.V2RunConcurrencyScopeWorkflow
	var jobConcurrencyDef *sdk.WorkflowConcurrency
	for i := range run.WorkflowData.Workflow.Concurrencies {
		if run.WorkflowData.Workflow.Concurrencies[i].Name == concurrencyName {
			jobConcurrencyDef = &run.WorkflowData.Workflow.Concurrencies[i]
			break
		}
	}
	// Search concurrency on project
	if jobConcurrencyDef == nil {
		projectConcurrency, err := project.LoadConcurrencyByNameAndProjectKey(ctx, db, run.ProjectKey, concurrencyName)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, err
			}
		}
		if projectConcurrency != nil {
			scope = sdk.V2RunConcurrencyScopeProject
			pwc := projectConcurrency.ToWorkflowConcurrency()
			jobConcurrencyDef = &pwc
		}
	}

	// No concurrency found
	if jobConcurrencyDef == nil {
		return nil, nil
	}

	return &sdk.V2RunConcurrency{
		Scope:               scope,
		WorkflowConcurrency: *jobConcurrencyDef,
	}, nil
}

func (api *API) cancelRunObjects(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, runObjectsToCancel map[string]workflow_v2.ConcurrencyObject) error {
	runCancelled := make([]sdk.V2WorkflowRun, 0)
	runJobCancelled := make([]sdk.V2WorkflowRunJob, 0)
	for _, runObject := range runObjectsToCancel {
		switch runObject.Type {
		case workflow_v2.ConcurrencyObjectTypeWorkflow:
			run, err := workflow_v2.LoadRunByID(ctx, tx, runObject.ID)
			if err != nil {
				return err
			}
			// Do nothing, we will enqueue the workflow with a dedicated status
			runCancelled = append(runCancelled, *run)
		default:
			rj, err := workflow_v2.LoadRunJobByID(ctx, tx, runObject.ID)
			if err != nil {
				return err
			}
			rj.Status = sdk.V2WorkflowRunJobStatusCancelled
			now := time.Now()
			rj.Ended = &now
			if err := workflow_v2.UpdateJobRun(ctx, tx, rj); err != nil {
				return err
			}
			jobInfo := sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    rj.WorkflowRunID,
				WorkflowRunJobID: rj.ID,
				IssuedAt:         time.Now(),
				Level:            sdk.WorkflowRunInfoLevelWarning,
				Message:          fmt.Sprintf("Job cancelled due to concurrency %q", rj.Concurrency.Name),
			}
			if err := workflow_v2.InsertRunJobInfo(ctx, tx, &jobInfo); err != nil {
				return err
			}
			runJobCancelled = append(runJobCancelled, *rj)
		}
	}
	api.enqueueCancelledRunObjects(ctx, runCancelled, runJobCancelled)
	return nil
}
