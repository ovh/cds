package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

// Call when a run job ends
func (api *API) manageEndJobConcurrency(jobRun sdk.V2WorkflowRunJob) {
	if jobRun.Concurrency == nil {
		return
	}

	api.GoRoutines.Exec(context.Background(), "manageEndJobConcurrency."+jobRun.ID, func(ctx context.Context) {
		ctx = context.WithValue(ctx, cdslog.WorkflowRunID, jobRun.WorkflowRunID)
		log.Info(ctx, "job %s: unblock job for concurrency %s scope %s", jobRun.Concurrency.Name, jobRun.Concurrency.Scope)

		rjs, _, err := retrieveRunJobToUnLocked(ctx, api.mustDB(), jobRun)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}

		// Enqueue workflow
		for _, rj := range rjs {
			api.EnqueueWorkflowRun(ctx, rj.WorkflowRunID, rj.Initiator, rj.WorkflowName, rj.RunNumber)

			tx, err := api.mustDB().Begin()
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				return
			}
			defer tx.Rollback()

			msg := sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    jobRun.WorkflowRunID,
				WorkflowRunJobID: jobRun.ID,
				IssuedAt:         time.Now(),
				Level:            sdk.WorkflowRunInfoLevelInfo,
				Message:          fmt.Sprintf("Unlocking job %s on workflow %s/%s/%s on run %d for concurrency '%s'", rj.JobID, rj.VCSServer, rj.Repository, rj.WorkflowName, rj.RunNumber, rj.Concurrency.Name),
			}
			if err := workflow_v2.InsertRunJobInfo(ctx, tx, &msg); err != nil {
				log.ErrorWithStackTrace(ctx, err)
				return
			}

			if err := tx.Commit(); err != nil {
				log.ErrorWithStackTrace(ctx, err)
				return
			}
		}
	})
}

// Retrieve the next rj to unblocked.
func retrieveRunJobToUnLocked(ctx context.Context, db *gorp.DbMap, jobRun sdk.V2WorkflowRunJob) ([]sdk.V2WorkflowRunJob, []string, error) {
	var ruleToApply *sdk.V2RunJobConcurrency
	var nbBuilding int64
	var err error
	switch jobRun.Concurrency.Scope {
	case sdk.V2RunJobConcurrencyScopeProject:
		ruleToApply, nbBuilding, _, err = checkJobProjectConcurrency(ctx, db, jobRun.ProjectKey, jobRun.Concurrency.WorkflowConcurrency)
		if err != nil {
			return nil, nil, err
		}
	default:
		ruleToApply, nbBuilding, _, err = checkJobWorkflowConcurrency(ctx, db, jobRun.ProjectKey, jobRun.VCSServer, jobRun.Repository, jobRun.WorkflowName, jobRun.Concurrency.WorkflowConcurrency)
		if err != nil {
			return nil, nil, err
		}
	}

	if ruleToApply == nil {
		log.Error(ctx, "unable to retrieve concurreny rule for workflow % on job %v", jobRun.WorkflowName, jobRun.JobID)
		return nil, nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to retrieve concurreny rule for workflow %s on job %s", jobRun.WorkflowName, jobRun.JobID)
	}

	// If cancel in progress, just retrieve the newest rjs to start
	if ruleToApply.CancelInProgress {
		var rjs []sdk.V2WorkflowRunJob
		switch jobRun.Concurrency.Scope {
		case sdk.V2RunJobConcurrencyScopeProject:
			rjs, err = workflow_v2.LoadNewestRunJobWithProjectScopedConcurrency(ctx, db, jobRun.ProjectKey, jobRun.Concurrency.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, -1)
		default:
			rjs, err = workflow_v2.LoadNewestRunJobWithWorkflowScopedConcurrency(ctx, db, jobRun.ProjectKey, jobRun.VCSServer, jobRun.Repository, jobRun.WorkflowName, jobRun.Concurrency.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, -1)
		}
		if err != nil {
			return nil, nil, err
		}
		rjsUnlocked := make([]sdk.V2WorkflowRunJob, 0)
		rjsCancelled := make([]string, 0)
		nbToUnlocked := ruleToApply.Pool - nbBuilding

		// All can be unlocked
		if nbToUnlocked >= int64(len(rjs)) {
			rjsUnlocked = append(rjsUnlocked, rjs...)
		} else {
			rjsUnlocked = append(rjsUnlocked, rjs[:nbToUnlocked]...)
			for i := nbToUnlocked; i < int64(len(rjs)); i++ {
				rjsCancelled = append(rjsCancelled, rjs[i].ID)
			}
		}
		return rjsUnlocked, rjsCancelled, nil
	}

	// No cancel in progress

	// Test if pool is full
	if ruleToApply.Pool <= nbBuilding {
		return nil, nil, nil
	}

	nbToUnlocked := ruleToApply.Pool - nbBuilding
	rjToUnlocked := make([]sdk.V2WorkflowRunJob, 0)
	switch jobRun.Concurrency.Scope {
	case sdk.V2RunJobConcurrencyScopeProject:
		if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst {
			// Load oldest
			rjs, err := workflow_v2.LoadOldestRunJobWithProjectScopedConcurrency(ctx, db, jobRun.ProjectKey, jobRun.Concurrency.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, nbToUnlocked)
			if err != nil {
				return nil, nil, err
			}
			rjToUnlocked = append(rjToUnlocked, rjs...)
		} else {
			// Load newest
			rjs, err := workflow_v2.LoadNewestRunJobWithProjectScopedConcurrency(ctx, db, jobRun.ProjectKey, jobRun.Concurrency.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, nbToUnlocked)
			if err != nil {
				return nil, nil, err
			}
			rjToUnlocked = append(rjToUnlocked, rjs...)
		}
	default:
		if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst {
			// Load oldest
			rjs, err := workflow_v2.LoadOldestRunJobWithWorkflowScopedConcurrency(ctx, db, jobRun.ProjectKey, jobRun.VCSServer, jobRun.Repository, jobRun.WorkflowName, jobRun.Concurrency.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, nbToUnlocked)
			if err != nil {
				return nil, nil, err
			}
			rjToUnlocked = append(rjToUnlocked, rjs...)
		} else {
			// Load newest
			var err error
			rjs, err := workflow_v2.LoadNewestRunJobWithWorkflowScopedConcurrency(ctx, db, jobRun.ProjectKey, jobRun.VCSServer, jobRun.Repository, jobRun.WorkflowName, jobRun.Concurrency.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, nbToUnlocked)
			if err != nil {
				return nil, nil, err
			}
			rjToUnlocked = append(rjToUnlocked, rjs...)
		}
	}
	return rjToUnlocked, nil, nil
}

// Update new run job with concurrency data, check if we have to lock it
func manageJobConcurrency(ctx context.Context, db *gorp.DbMap, run sdk.V2WorkflowRun, jobID string, runJob *sdk.V2WorkflowRunJob, concurrenciesDef map[string]sdk.V2RunJobConcurrency, concurrencyUnlockedCount map[string]int64, rjToCancelled map[string]struct{}) (*sdk.V2WorkflowRunJobInfo, error) {
	if runJob.Job.Concurrency != "" {
		concurrencyDef, has := concurrenciesDef[jobID]
		if !has {
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
		var ruleToApply *sdk.V2RunJobConcurrency
		var nbRunJobBuilding, nbRunJobBlocked int64
		var err error
		if concurrencyDef.Scope == sdk.V2RunJobConcurrencyScopeProject {
			ruleToApply, nbRunJobBuilding, nbRunJobBlocked, err = checkJobProjectConcurrency(ctx, db, run.ProjectKey, concurrencyDef.WorkflowConcurrency)
		} else {
			ruleToApply, nbRunJobBuilding, nbRunJobBlocked, err = checkJobWorkflowConcurrency(ctx, db, run.ProjectKey, run.VCSServer, run.Repository, run.WorkflowName, concurrencyDef.WorkflowConcurrency)
		}
		if err != nil {
			return nil, err
		}

		poolIsFull := nbRunJobBlocked+nbRunJobBuilding+concurrencyUnlockedCount[runJob.Job.Concurrency] >= ruleToApply.Pool

		if !ruleToApply.CancelInProgress {
			if poolIsFull {
				return &sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    run.ID,
					WorkflowRunJobID: runJob.ID,
					IssuedAt:         time.Now(),
					Level:            sdk.WorkflowRunInfoLevelError,
					Message:          fmt.Sprintf("Locked by concurrency '%s'", runJob.Job.Concurrency),
				}, nil
			} else {
				if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst && nbRunJobBlocked > 0 {
					return &sdk.V2WorkflowRunJobInfo{
						WorkflowRunID:    run.ID,
						WorkflowRunJobID: runJob.ID,
						IssuedAt:         time.Now(),
						Level:            sdk.WorkflowRunInfoLevelError,
						Message:          fmt.Sprintf("Locked by concurrency '%s'", runJob.Job.Concurrency),
					}, nil
				}
			}
		} else {
			// Manage cancel-in-progress
			idsToCancelled, err := retrieveRunJobToCancelled(ctx, db, runJob, ruleToApply, nbRunJobBuilding, concurrencyUnlockedCount[runJob.Job.Concurrency])
			if err != nil {
				return nil, err
			}
			for _, id := range idsToCancelled {
				rjToCancelled[id] = struct{}{}
			}
		}

		if _, has := rjToCancelled[runJob.ID]; !has {
			concurrencyUnlockedCount[runJob.Job.Concurrency]++
		}
	}
	return nil, nil
}

func retrieveRunJobToCancelled(ctx context.Context, db gorp.SqlExecutor, runJob *sdk.V2WorkflowRunJob, ruleToApply *sdk.V2RunJobConcurrency, nbBuilding, currentOnSameRule int64) ([]string, error) {
	nbToCancelled := nbBuilding + currentOnSameRule - ruleToApply.Pool + 1
	rjsToCancelled := make([]string, 0)

	// First cancel building jobs
	var err error
	var buildingJobs []sdk.V2WorkflowRunJob
	switch ruleToApply.Scope {
	case sdk.V2RunJobConcurrencyScopeProject:
		buildingJobs, err = workflow_v2.LoadOldestRunJobWithProjectScopedConcurrency(ctx, db, runJob.ProjectKey, ruleToApply.Name, []string{string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}, nbToCancelled)
	default:
		buildingJobs, err = workflow_v2.LoadOldestRunJobWithWorkflowScopedConcurrency(ctx, db, runJob.ProjectKey, runJob.VCSServer, runJob.Repository, runJob.WorkflowName, runJob.Concurrency.Name, []string{string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}, nbToCancelled)
	}
	if err != nil {
		return nil, err
	}
	for _, brj := range buildingJobs {
		rjsToCancelled = append(rjsToCancelled, brj.ID)
	}
	nbToCancelled -= int64(len(buildingJobs))

	// Then cancel blocked jobs
	var blockedJobs []sdk.V2WorkflowRunJob
	switch ruleToApply.Scope {
	case sdk.V2RunJobConcurrencyScopeProject:
		blockedJobs, err = workflow_v2.LoadOldestRunJobWithProjectScopedConcurrency(ctx, db, runJob.ProjectKey, ruleToApply.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, nbToCancelled)
	default:
		blockedJobs, err = workflow_v2.LoadOldestRunJobWithWorkflowScopedConcurrency(ctx, db, runJob.ProjectKey, runJob.VCSServer, runJob.Repository, runJob.WorkflowName, runJob.Concurrency.Name, []string{string(sdk.V2WorkflowRunJobStatusBlocked)}, nbToCancelled)
	}
	if err != nil {
		return nil, err
	}
	for _, brj := range blockedJobs {
		rjsToCancelled = append(rjsToCancelled, brj.ID)
	}
	nbToCancelled -= int64(len(blockedJobs))

	// Then cancel the current job
	if nbToCancelled > 0 {
		rjsToCancelled = append(rjsToCancelled, runJob.ID)
	}
	return rjsToCancelled, nil
}

// Retrieve rule for a project scoped concurrency
func checkJobProjectConcurrency(ctx context.Context, db gorp.SqlExecutor, projKey string, currentConcurrencyDef sdk.WorkflowConcurrency) (*sdk.V2RunJobConcurrency, int64, int64, error) {
	nbRunJobBuilding, err := workflow_v2.CountRunningRunJobWithProjectConcurrency(ctx, db, projKey, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}
	nbBlockedRunJobs, err := workflow_v2.CountBlockedRunJobWithProjectConcurrency(ctx, db, projKey, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}
	// Check if rules are differents between ongoing job run
	ongoingRules, err := workflow_v2.LoadProjectConcurrencyRules(ctx, db, projKey, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}
	ruleToApply := mergeConcurrencyRules(ongoingRules, currentConcurrencyDef)
	return &sdk.V2RunJobConcurrency{WorkflowConcurrency: ruleToApply, Scope: sdk.V2RunJobConcurrencyScopeProject}, nbRunJobBuilding, nbBlockedRunJobs, nil
}

// Retrieve rule for a workflow scoped concurrency
func checkJobWorkflowConcurrency(ctx context.Context, db gorp.SqlExecutor, projKey, vcsName, repo, workflowName string, currentConcurrencyDef sdk.WorkflowConcurrency) (*sdk.V2RunJobConcurrency, int64, int64, error) {
	nbRunJobBuilding, err := workflow_v2.CountRunningRunJobWithWorkflowConcurrency(ctx, db, projKey, vcsName, repo, workflowName, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}
	nbBlockedRunJobs, err := workflow_v2.CountBlockedRunJobWithWorkflowConcurrency(ctx, db, projKey, vcsName, repo, workflowName, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}

	// Check if rules are differents between ongoing job run
	ongoingRules, err := workflow_v2.LoadWorkflowConcurrencyRules(ctx, db, projKey, vcsName, repo, workflowName, currentConcurrencyDef.Name)
	if err != nil {
		return nil, 0, 0, err
	}
	ruleToApply := mergeConcurrencyRules(ongoingRules, currentConcurrencyDef)

	return &sdk.V2RunJobConcurrency{WorkflowConcurrency: ruleToApply, Scope: sdk.V2RunJobConcurrencyScopeWorkflow}, nbRunJobBuilding, nbBlockedRunJobs, nil
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

func retrieveConcurrencyDefinition(ctx context.Context, db gorp.SqlExecutor, run sdk.V2WorkflowRun, concurrencyName string) (*sdk.V2RunJobConcurrency, error) {
	// Search concurrency rule on workflow
	scope := sdk.V2RunJobConcurrencyScopeWorkflow
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
			scope = sdk.V2RunJobConcurrencyScopeProject
			pwc := projectConcurrency.ToWorkflowConcurrency()
			jobConcurrencyDef = &pwc
		}
	}

	// No concurrency found
	if jobConcurrencyDef == nil {
		return nil, nil
	}

	return &sdk.V2RunJobConcurrency{
		Scope:               scope,
		WorkflowConcurrency: *jobConcurrencyDef,
	}, nil
}
