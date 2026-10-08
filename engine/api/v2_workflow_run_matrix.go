package api

import (
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

const (
	defaultMatrixParallelism    int64 = 20
	defaultMatrixMaxParallelism int64 = 40
)

// matrixParallelism is how many permutations of a matrix job may run in parallel: Default applies when the job has no
// concurrency, Max applies whatever its concurrency.
type matrixParallelism struct {
	Default int64
	Max     int64
}

func (api *API) matrixParallelism() matrixParallelism {
	p := matrixParallelism{
		Default: api.Config.WorkflowV2.MatrixDefaultParallelism,
		Max:     api.Config.WorkflowV2.MatrixMaxParallelism,
	}
	if p.Default <= 0 {
		p.Default = defaultMatrixParallelism
	}
	if p.Max <= 0 {
		p.Max = defaultMatrixMaxParallelism
	}
	if p.Max < p.Default {
		p.Max = p.Default
	}
	return p
}

// limitMatrixParallelism gives a matrix job without concurrency an implicit rule, scoped to the workflow and the git
// ref name, when it has more permutations than the default parallelism. The rule is written in the run definition so
// that the next engine pass handles the job like any job with a concurrency. It returns the run info to record, or nil.
func limitMatrixParallelism(run *sdk.V2WorkflowRun, jobID string, jobDef sdk.V2Job, nbPermutations int, defaultParallelism int64) *sdk.V2WorkflowRunInfo {
	if jobDef.NeedsTemplateResolution() || jobDef.Concurrency != "" || int64(nbPermutations) <= defaultParallelism {
		return nil
	}
	ref := run.Contexts.Git.RefName
	if ref == "" {
		ref = run.Contexts.Git.Ref
	}
	name := fmt.Sprintf("matrix/%s/%s", jobID, ref)

	found := false
	for _, c := range run.WorkflowData.Workflow.Concurrencies {
		if c.Name == name {
			found = true
			break
		}
	}
	if !found {
		run.WorkflowData.Workflow.Concurrencies = append(run.WorkflowData.Workflow.Concurrencies, sdk.WorkflowConcurrency{
			Name:  name,
			Pool:  defaultParallelism,
			Order: sdk.ConcurrencyOrderOldestFirst,
		})
	}
	job := run.WorkflowData.Workflow.Jobs[jobID]
	job.Concurrency = name
	run.WorkflowData.Workflow.Jobs[jobID] = job

	return &sdk.V2WorkflowRunInfo{
		WorkflowRunID: run.ID,
		IssuedAt:      time.Now(),
		Level:         sdk.WorkflowRunInfoLevelWarning,
		Message:       fmt.Sprintf("Job %s: %d permutations, CDS limits it to %d jobs in parallel", jobID, nbPermutations, defaultParallelism),
	}
}

// capMatrixParallelism bounds the pool of the concurrency of a matrix job to the maximum parallelism, on the definition
// used to create its permutations; the rule itself is left untouched. It returns the run info to record when the
// permutations are first created, or nil.
func capMatrixParallelism(run *sdk.V2WorkflowRun, jobID string, concurrenciesDef map[string]sdk.V2RunConcurrency, nbPermutations int, existingRunJobs []sdk.V2WorkflowRunJob, maxParallelism int64) *sdk.V2WorkflowRunInfo {
	def, has := concurrenciesDef[jobID]
	if !has || int64(nbPermutations) <= maxParallelism || def.Pool <= maxParallelism {
		return nil
	}
	def.Pool = maxParallelism
	concurrenciesDef[jobID] = def

	// The cap applies to every pass creating permutations; the limit is only reported on the first one
	for _, rj := range existingRunJobs {
		if rj.JobID == jobID {
			return nil
		}
	}
	return &sdk.V2WorkflowRunInfo{
		WorkflowRunID: run.ID,
		IssuedAt:      time.Now(),
		Level:         sdk.WorkflowRunInfoLevelWarning,
		Message:       fmt.Sprintf("Job %s: %d permutations, concurrency %q pool limited to %d jobs in parallel", jobID, nbPermutations, def.Name, maxParallelism),
	}
}
