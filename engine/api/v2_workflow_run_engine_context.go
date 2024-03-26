package api

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func computeExistingRunJobContexts(runJobs []sdk.V2WorkflowRunJob, runResults []sdk.V2WorkflowRunResult) sdk.JobsResultContext {
	runResultMap := make(map[string][]sdk.V2WorkflowRunResultVariableDetail)
	for _, rr := range runResults {
		if rr.Type != sdk.V2WorkflowRunResultTypeVariable {
			continue
		}
		detail := rr.Detail.Data.(*sdk.V2WorkflowRunResultVariableDetail)
		jobResults, has := runResultMap[rr.WorkflowRunJobID]
		if !has {
			jobResults = make([]sdk.V2WorkflowRunResultVariableDetail, 0)
		}
		jobResults = append(jobResults, *detail)
		runResultMap[rr.WorkflowRunJobID] = jobResults
	}

	// Compute jobs context
	jobsContext := sdk.JobsResultContext{}
	matrixJobs := make(map[string][]sdk.JobResultContext)
	for _, rj := range runJobs {
		if rj.Status.IsTerminated() && len(rj.Matrix) == 0 {
			result := sdk.JobResultContext{
				Result:  rj.Status,
				Outputs: sdk.JobResultOutput{},
			}
			if rr, has := runResultMap[rj.ID]; has {
				for _, r := range rr {
					result.Outputs[r.Name] = r.Value
				}
			}
			jobsContext[rj.JobID] = result
		} else if len(rj.Matrix) > 0 {
			jobs, has := matrixJobs[rj.JobID]
			if !has {
				jobs = make([]sdk.JobResultContext, 0)
			}
			jobResultContext := sdk.JobResultContext{
				Result:  rj.Status,
				Outputs: sdk.JobResultOutput{},
			}
			rr, has := runResultMap[rj.ID]
			if has {
				for _, r := range rr {
					jobResultContext.Outputs[r.Name] = r.Value
				}
			}
			jobs = append(jobs, jobResultContext)
			matrixJobs[rj.JobID] = jobs
		}
	}

	// Manage matric jobs
nextjob:
	for k := range matrixJobs {
		outputs := sdk.JobResultOutput{}
		var finalStatus sdk.V2WorkflowRunJobStatus
		for _, rj := range matrixJobs[k] {
			if !rj.Result.IsTerminated() {
				continue nextjob
			}
			for outputK, outputV := range rj.Outputs {
				outputs[outputK] = outputV
			}

			switch finalStatus {
			case sdk.V2WorkflowRunJobStatusUnknown:
				finalStatus = rj.Result
			case sdk.V2WorkflowRunJobStatusSuccess:
				if rj.Result == sdk.V2WorkflowRunJobStatusStopped || rj.Result == sdk.V2WorkflowRunJobStatusFail {
					finalStatus = rj.Result
				}
			case sdk.V2WorkflowRunJobStatusFail:
				if rj.Result == sdk.V2WorkflowRunJobStatusStopped {
					finalStatus = rj.Result
				}
			}
		}
		result := sdk.JobResultContext{
			Result:  finalStatus,
			Outputs: outputs,
		}
		jobsContext[k] = result
	}

	return jobsContext
}

func buildContextForJob(_ context.Context, allJobs map[string]sdk.V2Job, runJobsContexts sdk.JobsResultContext, runContext sdk.WorkflowRunContext, jobID string) sdk.WorkflowRunJobsContext {
	jobsContext := sdk.JobsResultContext{}
	buildAncestorJobContext(jobID, allJobs, runJobsContexts, jobsContext)

	jobDef := allJobs[jobID]
	needsContext := sdk.NeedsContext{}
	for _, n := range jobDef.Needs {
		if j, has := jobsContext[n]; has {
			needContext := sdk.NeedContext{
				Result:  j.Result,
				Outputs: j.Outputs,
			}
			// override result if job has continue-on-error
			if allJobs[n].ContinueOnError && j.Result == sdk.V2WorkflowRunJobStatusFail {
				needContext.Result = sdk.V2WorkflowRunJobStatusSuccess
			}

			needsContext[n] = needContext
		}
	}

	currentJobContext := sdk.WorkflowRunJobsContext{
		WorkflowRunContext: runContext,
		Jobs:               jobsContext,
		Needs:              needsContext,
	}
	return currentJobContext
}

func buildAncestorJobContext(jobID string, jobs map[string]sdk.V2Job, runJobsContext sdk.JobsResultContext, currentJobContext sdk.JobsResultContext) {
	jobDef := jobs[jobID]
	if len(jobDef.Needs) == 0 {
		return
	}
	for _, n := range jobDef.Needs {
		jobCtx := runJobsContext[n]
		currentJobContext[n] = jobCtx
		buildAncestorJobContext(n, jobs, runJobsContext, currentJobContext)
	}
}
