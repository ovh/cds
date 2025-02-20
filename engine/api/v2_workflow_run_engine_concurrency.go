package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
)

func manageJobConcurrency(ctx context.Context, db *gorp.DbMap, run sdk.V2WorkflowRun, jobID string, jobDef sdk.V2Job) (*sdk.V2WorkflowRunJobInfo, error) {
	if jobDef.Concurrency != "" {
		scope := sdk.V2RunJobConcurrencyScopeWorkflow

		// Search concurrency rule on workflow or project
		var jobConcurrencyDef *sdk.Concurrency
		for i := range run.WorkflowData.Workflow.Concurrencies {
			if run.WorkflowData.Workflow.Concurrencies[i].Name == jobDef.Concurrency {
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
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("job %s: concurrency %s not found on workflow or on project", jobID, jobDef.Concurrency),
			}, nil
		}

		// Check order and pool
		if scope == sdk.V2RunJobConcurrencyScopeProject {
			// TODO Manage Concurrency at project level
		} else {
			return checkJobWorkflowConcurrency(ctx, db, run, jobID, jobDef, *jobConcurrencyDef)
		}

	}
	return nil, nil
}

func checkJobWorkflowConcurrency(ctx context.Context, db *gorp.DbMap, run sdk.V2WorkflowRun, jobID string, jobDef sdk.V2Job, currentConcurrencyDef sdk.Concurrency) (*sdk.V2WorkflowRunJobInfo, error) {
	nbRunJobBuilding, err := workflow_v2.CountRunningRunJobWithWorkflowConcurrency(ctx, db, run.ProjectKey, run.VCSServer, run.Repository, run.WorkflowName, jobDef.Concurrency)
	if err != nil {
		return nil, err
	}
	ruleToApply := currentConcurrencyDef

	// Check if rules are differents between ongoing job run
	ongoingRules, err := workflow_v2.LoadConcurrencyRules(ctx, db, run.ProjectKey, run.VCSServer, run.Repository, run.WorkflowName, jobDef.Concurrency)
	if err != nil {
		return nil, err
	}
	if ongoingRules != nil {
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

	nbBlockedRunJobs, err := workflow_v2.CountBlockedRunJobWithWorkflowConcurrency(ctx, db, run.ProjectKey, run.VCSServer, run.Repository, run.WorkflowName, jobDef.Concurrency)
	if err != nil {
		return nil, err
	}

	if !ruleToApply.CancelInProgress {
		if nbBlockedRunJobs+nbRunJobBuilding >= ruleToApply.Pool {
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("job %s: blocked by concurrency %s...waiting", jobID, jobDef.Concurrency),
			}, nil
		}

		if ruleToApply.Order == sdk.ConcurrencyOrderOldestFirst && nbBlockedRunJobs > 0 {
			return &sdk.V2WorkflowRunJobInfo{
				WorkflowRunID: run.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelError,
				Message:       fmt.Sprintf("job %s: blocked by concurrency %s...waiting", jobID, jobDef.Concurrency),
			}, nil
		}
		// Run the job
	} else {
		// TODO cancel in progress
	}
	return nil, nil
}
