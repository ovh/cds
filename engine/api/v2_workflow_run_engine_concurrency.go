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

		rj, err := retrieveRunJobToUnLocked(ctx, api.mustDB(), jobRun)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}

		// Enqueue workflow
		if rj != nil {
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
func retrieveRunJobToUnLocked(ctx context.Context, db *gorp.DbMap, jobRun sdk.V2WorkflowRunJob) (*sdk.V2WorkflowRunJob, error) {
	var ruleToApply *sdk.WorkflowConcurrency
	var nbBuilding int64
	var err error
	switch jobRun.Concurrency.Scope {
	case sdk.V2RunJobConcurrencyScopeWorkflow:
		ruleToApply, nbBuilding, _, err = checkJobWorkflowConcurrency(ctx, db, jobRun.ProjectKey, jobRun.VCSServer, jobRun.Repository, jobRun.WorkflowName, jobRun.Concurrency.WorkflowConcurrency)
		if err != nil {
			return nil, err
		}
	default:
		ruleToApply, nbBuilding, _, err = checkJobProjectConcurrency(ctx, db, jobRun.ProjectKey, jobRun.Concurrency.WorkflowConcurrency)
		if err != nil {
			return nil, err
		}
	}

	if ruleToApply == nil {
		log.Error(ctx, "unable to retrieve concurreny rule for workflow % on job %v", jobRun.WorkflowName, jobRun.JobID)
		return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to retrieve concurreny rule for workflow %s on job %s", jobRun.WorkflowName, jobRun.JobID)
	}
	if ruleToApply.CancelInProgress {
		// Nothing to do, there is no queue
		return nil, nil
	}

	// Test if pool is full
	if ruleToApply.Pool <= nbBuilding {
		return nil, nil
	}

	var rj *sdk.V2WorkflowRunJob
	switch jobRun.Concurrency.Scope {
	case sdk.V2RunJobConcurrencyScopeWorkflow:
		if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst {
			// Load oldest
			var err error
			rj, err = workflow_v2.LoadOldestRunJobWithSameConcurrencyOnSameWorkflow(ctx, db, jobRun.ProjectKey, jobRun.VCSServer, jobRun.Repository, jobRun.WorkflowName, jobRun.Concurrency.Name)
			if err != nil {
				return nil, err
			}
		} else {
			// Load newest
			var err error
			rj, err = workflow_v2.LoadNewestRunJobWithSameConcurrencyOnSameWorkflow(ctx, db, jobRun.ProjectKey, jobRun.VCSServer, jobRun.Repository, jobRun.WorkflowName, jobRun.Concurrency.Name)
			if err != nil {
				return nil, err
			}
		}
	default:
		if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst {
			// Load oldest
			var err error
			rj, err = workflow_v2.LoadOldestRunJobWithSameConcurrencyOnSameProject(ctx, db, jobRun.ProjectKey, jobRun.Concurrency.Name)
			if err != nil {
				return nil, err
			}
		} else {
			// Load newest
			var err error
			rj, err = workflow_v2.LoadNewestRunJobWithSameConcurrencyOnSameProject(ctx, db, jobRun.ProjectKey, jobRun.Concurrency.Name)
			if err != nil {
				return nil, err
			}
		}
	}
	return rj, nil
}

// Update new run job with concurrency data, check if we have to lock it
func manageJobConcurrency(ctx context.Context, db *gorp.DbMap, run sdk.V2WorkflowRun, jobID string, runJob *sdk.V2WorkflowRunJob, concurrencyUnlockedCount map[string]int64) (*sdk.V2WorkflowRunJobInfo, error) {
	if runJob.Job.Concurrency != "" {
		scope := sdk.V2RunJobConcurrencyScopeWorkflow

		// Search concurrency rule on workflow
		var jobConcurrencyDef *sdk.WorkflowConcurrency
		for i := range run.WorkflowData.Workflow.Concurrencies {
			if run.WorkflowData.Workflow.Concurrencies[i].Name == runJob.Job.Concurrency {
				jobConcurrencyDef = &run.WorkflowData.Workflow.Concurrencies[i]
				break
			}
		}
		// Search concurrency on project
		if jobConcurrencyDef == nil {
			projectConcurrency, err := project.LoadConcurrencyByNameAndProjectKey(ctx, db, run.ProjectKey, runJob.Job.Concurrency)
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
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    run.ID,
				WorkflowRunJobID: runJob.ID,
				IssuedAt:         time.Now(),
				Level:            sdk.WorkflowRunInfoLevelError,
				Message:          fmt.Sprintf("job %s: concurrency %s not found on workflow or on project", jobID, runJob.Job.Concurrency),
			}, nil
		}

		// Set concurrency on runjob
		runJob.Concurrency = &sdk.V2RunJobConcurrency{
			WorkflowConcurrency: *jobConcurrencyDef,
			Scope:               scope,
		}

		// Retrieve current building and blocked job to check if we can enqueue this one
		var ruleToApply *sdk.WorkflowConcurrency
		var nbRunJobBuilding, nbRunJobBlocked int64
		var err error
		if scope == sdk.V2RunJobConcurrencyScopeProject {
			ruleToApply, nbRunJobBuilding, nbRunJobBlocked, err = checkJobProjectConcurrency(ctx, db, run.ProjectKey, *jobConcurrencyDef)
		} else {
			ruleToApply, nbRunJobBuilding, nbRunJobBlocked, err = checkJobWorkflowConcurrency(ctx, db, run.ProjectKey, run.VCSServer, run.Repository, run.WorkflowName, *jobConcurrencyDef)
		}
		if err != nil {
			return nil, err
		}

		if !ruleToApply.CancelInProgress {
			// Pool is full -> lock the job
			if nbRunJobBlocked+nbRunJobBuilding+concurrencyUnlockedCount[runJob.Job.Concurrency] >= ruleToApply.Pool {
				return &sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    run.ID,
					WorkflowRunJobID: runJob.ID,
					IssuedAt:         time.Now(),
					Level:            sdk.WorkflowRunInfoLevelError,
					Message:          fmt.Sprintf("Locked by concurrency '%s'", runJob.Job.Concurrency),
				}, nil
			}
			// Pool not full, but there are older runjobs
			if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst && nbRunJobBlocked > 0 {
				return &sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    run.ID,
					WorkflowRunJobID: runJob.ID,
					IssuedAt:         time.Now(),
					Level:            sdk.WorkflowRunInfoLevelError,
					Message:          fmt.Sprintf("Locked by concurrency '%s'", runJob.Job.Concurrency),
				}, nil
			}
		} else {
			// /////////////////
			// TODO cancel in progress
			// /////////////////
		}
		concurrencyUnlockedCount[runJob.Job.Concurrency]++
	}
	return nil, nil
}

// Retrieve rule for a project scoped concurrency
func checkJobProjectConcurrency(ctx context.Context, db gorp.SqlExecutor, projKey string, currentConcurrencyDef sdk.WorkflowConcurrency) (*sdk.WorkflowConcurrency, int64, int64, error) {
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
	return &ruleToApply, nbRunJobBuilding, nbBlockedRunJobs, nil
}

// Retrieve rule for a workflow scoped concurrency
func checkJobWorkflowConcurrency(ctx context.Context, db gorp.SqlExecutor, projKey, vcsName, repo, workflowName string, currentConcurrencyDef sdk.WorkflowConcurrency) (*sdk.WorkflowConcurrency, int64, int64, error) {
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

	return &ruleToApply, nbRunJobBuilding, nbBlockedRunJobs, nil
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
