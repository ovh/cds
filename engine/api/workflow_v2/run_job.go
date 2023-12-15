package workflow_v2

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func RetrieveJobToKeep(_ context.Context, w sdk.V2Workflow, runJobsMap map[string]sdk.V2WorkflowRunJob, runJobToRestart map[string]sdk.V2WorkflowRunJob) map[string]sdk.V2WorkflowRunJob {
	runJobsToKeep := make(map[string]sdk.V2WorkflowRunJob)

	// Browse all run jobs
allJobsLoop:
	for runJobID, runJob := range runJobsMap {
		// exclude job to restart
		for id := range runJobToRestart {
			if runJobID == id {
				continue allJobsLoop
			}
		}

		// Get all job ancestors
		parentJobs := sdk.WorkflowJobParents(w, runJob.JobID)

		// Check if in job ancestors there is a job to restart, if yes do not keep this job
		for _, parentJobID := range parentJobs {
			for _, rj := range runJobToRestart {
				if rj.JobID == parentJobID {
					continue allJobsLoop
				}
			}
		}

		// Keep this job and all ancestors
		runJobsToKeep[runJob.ID] = runJob
		for _, a := range parentJobs {
			for _, rj := range runJobsMap {
				if rj.JobID == a {
					runJobsToKeep[rj.ID] = rj
					if len(rj.Matrix) == 0 {
						break
					}
				}
			}
		}
	}
	return runJobsToKeep
}
