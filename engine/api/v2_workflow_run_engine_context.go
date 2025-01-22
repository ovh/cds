package api

import (
	"context"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func computeExistingRunJobContexts(ctx context.Context, runJobs []sdk.V2WorkflowRunJob, runResults []sdk.V2WorkflowRunResult) (sdk.JobsResultContext, sdk.JobsGateContext) {
	runResultMap := make(map[string][]sdk.V2WorkflowRunResult)
	for _, rr := range runResults {
		jobResults, has := runResultMap[rr.WorkflowRunJobID]
		if !has {
			jobResults = make([]sdk.V2WorkflowRunResult, 0)
		}
		jobResults = append(jobResults, rr)
		runResultMap[rr.WorkflowRunJobID] = jobResults
	}

	// Compute jobs context
	jobsContext := sdk.JobsResultContext{}
	gatesContext := sdk.JobsGateContext{}
	matrixJobs := make(map[string][]sdk.JobResultContext)

	for _, rj := range runJobs {
		if rj.Status.IsTerminated() && len(rj.Matrix) == 0 {
			result := sdk.JobResultContext{
				Result:  rj.Status,
				Outputs: sdk.JobResultOutput{},
			}
			if rr, has := runResultMap[rj.ID]; has {
				for _, r := range rr {
					switch r.Type {
					case sdk.V2WorkflowRunResultTypeVariable, sdk.V2WorkflowRunResultVariableDetailType:
						x, err := sdk.GetConcreteDetail[*sdk.V2WorkflowRunResultVariableDetail](&r)
						if err != nil {
							log.ErrorWithStackTrace(ctx, err)
							continue
						}
						result.Outputs[x.Name] = x.Value
					default:
						if result.JobRunResults == nil {
							result.JobRunResults = sdk.JobRunResults{}
						}
						result.JobRunResults[r.Name()], _ = r.GetDetail()
					}
				}
			}
			jobsContext[rj.JobID] = result
			if len(rj.GateInputs) > 0 {
				gatesContext[rj.JobID] = rj.GateInputs
			}
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
					switch r.Type {
					case sdk.V2WorkflowRunResultTypeVariable, sdk.V2WorkflowRunResultVariableDetailType:
						x, err := sdk.GetConcreteDetail[*sdk.V2WorkflowRunResultVariableDetail](&r)
						if err != nil {
							log.ErrorWithStackTrace(ctx, err)
							continue
						}
						jobResultContext.Outputs[x.Name] = x.Value
					default:
						if jobResultContext.JobRunResults == nil {
							jobResultContext.JobRunResults = sdk.JobRunResults{}
						}
						jobResultContext.JobRunResults[r.Name()], _ = r.GetDetail()
					}
				}
			}
			jobs = append(jobs, jobResultContext)
			matrixJobs[rj.JobID] = jobs
			if len(rj.GateInputs) > 0 {
				gatesContext[rj.JobID] = rj.GateInputs
			}
		}
	}

	// Manage matrix jobs
nextjob:
	for k := range matrixJobs {
		// Check if all permutations have run
		var jobDef sdk.V2Job
		for _, rj := range runJobs {
			if rj.JobID == k {
				jobDef = rj.Job
				break
			}
		}
		var nbPermutations = 1
		for _, v := range jobDef.Strategy.Matrix {
			if vString, ok := v.([]string); ok {
				nbPermutations *= len(vString)
			} else if vInterface, ok := v.([]interface{}); ok {
				nbPermutations *= len(vInterface)
			}
		}
		// if there is still permutation to run, ignore this job context
		if nbPermutations > len(matrixJobs[k]) {
			continue
		}

		// Compute job status
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

	return jobsContext, gatesContext
}

func buildContextForJob(ctx context.Context, workflow sdk.V2Workflow, runJobsContexts sdk.JobsResultContext, runContext sdk.WorkflowRunContext, stages sdk.WorkflowRunStages, jobID string) sdk.WorkflowRunJobsContext {
	jobsContext := sdk.JobsResultContext{}
	buildAncestorJobContext(ctx, jobID, workflow, runJobsContexts, stages, jobsContext)

	jobDef := workflow.Jobs[jobID]
	needsContext := sdk.NeedsContext{}

	var jobNeeds []string
	if len(jobDef.Needs) > 0 {
		jobNeeds = jobDef.Needs

	} else if jobDef.Stage != "" {
		jobNeeds = make([]string, 0)
		// add all final jobs from parent stages
		neededStages := workflow.Stages[jobDef.Stage].Needs
		for _, n := range neededStages {
			for jobID, jobInStage := range stages[n].Jobs {
				if jobInStage.IsFinal {
					jobNeeds = append(jobNeeds, jobID)
				}
			}
		}
	}

	for _, n := range jobNeeds {
		if j, has := jobsContext[n]; has {
			needContext := sdk.NeedContext{
				Result:  j.Result,
				Outputs: j.Outputs,
			}
			// override result if job has continue-on-error
			if workflow.Jobs[n].ContinueOnError && j.Result == sdk.V2WorkflowRunJobStatusFail {
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

func buildAncestorJobContext(ctx context.Context, jobID string, workflow sdk.V2Workflow, runJobsContext sdk.JobsResultContext, stages sdk.WorkflowRunStages, currentJobContext sdk.JobsResultContext) {
	jobDef := workflow.Jobs[jobID]
	if len(jobDef.Needs) == 0 && jobDef.Stage != "" {
		// add all final jobs from parent stages
		neededStages := workflow.Stages[jobDef.Stage].Needs
		for _, n := range neededStages {
			for jobID, jobInStage := range stages[n].Jobs {
				if jobInStage.IsFinal {
					jobCtx := runJobsContext[jobID]
					currentJobContext[jobID] = jobCtx
					buildAncestorJobContext(ctx, jobID, workflow, runJobsContext, stages, currentJobContext)
				}
			}
		}
	}

	for _, n := range jobDef.Needs {
		jobCtx := runJobsContext[n]
		currentJobContext[n] = jobCtx
		buildAncestorJobContext(ctx, n, workflow, runJobsContext, stages, currentJobContext)
	}
}
