package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

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

		rj, err := retrieveRunJobToUnblocked(ctx, api.mustDB(), jobRun)
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
func retrieveRunJobToUnblocked(ctx context.Context, db *gorp.DbMap, jobRun sdk.V2WorkflowRunJob) (*sdk.V2WorkflowRunJob, error) {
	var ruleToApply *sdk.Concurrency
	var nbBuilding int64
	var err error
	switch jobRun.Concurrency.Scope {
	case sdk.V2RunJobConcurrencyScopeWorkflow:

		ruleToApply, nbBuilding, _, err = checkJobWorkflowConcurrency(ctx, db, jobRun.ProjectKey, jobRun.VCSServer, jobRun.Repository, jobRun.WorkflowName, jobRun.Job, jobRun.Concurrency.Concurrency)
		if err != nil {
			return nil, err
		}

	default:
		// //////
		// TODO Project scoped
	}

	if ruleToApply == nil {
		log.Error(ctx, "unable to retrieve concurreny rule for workflow % on job %", jobRun.WorkflowName, jobRun.JobID)
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
		// //////
		// TODO Project scoped
	}
	return rj, nil
}

func manageJobConcurrency(ctx context.Context, db *gorp.DbMap, run sdk.V2WorkflowRun, jobID string, runJob *sdk.V2WorkflowRunJob) (*sdk.V2WorkflowRunJobInfo, error) {
	if runJob.Job.Concurrency != "" {
		scope := sdk.V2RunJobConcurrencyScopeWorkflow

		// Search concurrency rule on workflow or project
		var jobConcurrencyDef *sdk.Concurrency
		for i := range run.WorkflowData.Workflow.Concurrencies {
			if run.WorkflowData.Workflow.Concurrencies[i].Name == runJob.Job.Concurrency {
				jobConcurrencyDef = &run.WorkflowData.Workflow.Concurrencies[i]
			}
		}
		// If not found on workflow, check on project
		if jobConcurrencyDef == nil {
			// //////////////
			// TODO load from project
			// //////////////
			scope = sdk.V2RunJobConcurrencyScopeProject
		}

		// If nothing found, fail the job
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
			Concurrency: *jobConcurrencyDef,
			Scope:       scope,
		}

		// Check order and pool
		if scope == sdk.V2RunJobConcurrencyScopeProject {
			// TODO Manage Concurrency at project level
		} else {
			ruleToApply, nbRunJobBuilding, nbRunJobBlocked, err := checkJobWorkflowConcurrency(ctx, db, run.ProjectKey, run.VCSServer, run.Repository, run.WorkflowName, runJob.Job, *jobConcurrencyDef)
			if err != nil {
				return nil, err
			}
			if !ruleToApply.CancelInProgress {
				// Pool is full
				if nbRunJobBlocked+nbRunJobBuilding >= ruleToApply.Pool {
					return &sdk.V2WorkflowRunJobInfo{
						WorkflowRunID:    run.ID,
						WorkflowRunJobID: runJob.ID,
						IssuedAt:         time.Now(),
						Level:            sdk.WorkflowRunInfoLevelError,
						Message:          fmt.Sprintf("Locked by concurrency '%s'", runJob.Job.Concurrency),
					}, nil
				}
				// Pool not found, but there are older runjobs
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
				// TODO cancel in progress
			}
		}

	}
	return nil, nil
}

func checkJobWorkflowConcurrency(ctx context.Context, db *gorp.DbMap, projKey, vcsName, repo, workflowName string, jobDef sdk.V2Job, currentConcurrencyDef sdk.Concurrency) (*sdk.Concurrency, int64, int64, error) {
	nbRunJobBuilding, err := workflow_v2.CountRunningRunJobWithWorkflowConcurrency(ctx, db, projKey, vcsName, repo, workflowName, jobDef.Concurrency)
	if err != nil {
		return nil, 0, 0, err
	}
	ruleToApply := currentConcurrencyDef

	// Check if rules are differents between ongoing job run
	ongoingRules, err := workflow_v2.LoadConcurrencyRules(ctx, db, projKey, vcsName, repo, workflowName, jobDef.Concurrency)
	if err != nil {
		return nil, 0, 0, err
	}
	if len(ongoingRules) > 0 {
		for _, r := range ongoingRules {
			// Default behaviours if there is multiple configuration
			if r.Order != string(ruleToApply.Order) {
				ruleToApply.Order = sdk.ConcurrencyOrderOldestFirst
			}
			if r.MinPool < currentConcurrencyDef.Pool {
				ruleToApply.Pool = r.MinPool
			}
			if r.Cancel != currentConcurrencyDef.CancelInProgress {
				ruleToApply.CancelInProgress = false
			}
		}
	}

	nbBlockedRunJobs, err := workflow_v2.CountBlockedRunJobWithWorkflowConcurrency(ctx, db, projKey, vcsName, repo, workflowName, jobDef.Concurrency)
	if err != nil {
		return nil, 0, 0, err
	}
	return &ruleToApply, nbRunJobBuilding, nbBlockedRunJobs, nil
}
