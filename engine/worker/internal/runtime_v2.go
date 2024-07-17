package internal

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/glob"
	"github.com/ovh/cds/sdk/jws"
	"github.com/rockbears/log"
)

func (wk *CurrentWorker) V2AddRunResult(ctx context.Context, req workerruntime.V2RunResultRequest) (*workerruntime.V2AddResultResponse, error) {
	ctx = workerruntime.SetRunJobID(ctx, wk.currentJobV2.runJob.ID)
	var (
		runResult = req.RunResult
		integ     *sdk.JobIntegrationsContext
	)

	if wk.currentJobV2.runJobContext.Integrations.ArtifactManager.Name != "" {
		integ = &wk.currentJobV2.runJobContext.Integrations.ArtifactManager
	}

	// Create the run result on API side
	log.Info(ctx, "creating run result %s", runResult.Name())
	if err := wk.clientV2.V2QueueJobRunResultCreate(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, runResult); err != nil {
		return nil, sdk.NewError(sdk.ErrUnknownError, err)
	}

	response := workerruntime.V2AddResultResponse{
		RunResult: runResult,
	}

	if integ == nil {
		// Generate a worker signature
		sig := cdn.Signature{
			JobName:       wk.currentJobV2.runJob.Job.Name,
			RunJobID:      wk.currentJobV2.runJob.ID,
			ProjectKey:    wk.currentJobV2.runJob.ProjectKey,
			WorkflowName:  wk.currentJobV2.runJob.WorkflowName,
			WorkflowRunID: wk.currentJobV2.runJob.WorkflowRunID,
			RunNumber:     wk.currentJobV2.runJob.RunNumber,
			RunAttempt:    wk.currentJobV2.runJob.RunAttempt,
			Region:        wk.currentJobV2.runJob.Region,

			Timestamp: time.Now().UnixNano(),

			Worker: &cdn.SignatureWorker{
				WorkerID:      wk.id,
				WorkerName:    wk.Name(),
				RunResultName: runResult.Name(),
				RunResultType: runResult.Typ(),
				RunResultID:   runResult.ID,
			},
		}

		signature, err := jws.Sign(wk.signer, sig)
		if err != nil {
			return nil, sdk.NewError(sdk.ErrUnknownError, err)
		}
		// Returns the signature and CDN info
		response.CDNSignature = signature
		response.CDNAddress = wk.CDNHttpURL()
	} else {
		if response.RunResult.Type != sdk.V2WorkflowRunResultTypeArsenalDeployment &&
			response.RunResult.Type != sdk.V2WorkflowRunResultTypeRelease && response.RunResult.Type != sdk.V2WorkflowRunResultTypeVariable {
			log.Info(ctx, "enabling integration %q for run result %s", integ.Name, response.RunResult.ID)
			response.RunResult.ArtifactManagerIntegrationName = &integ.Name
		}
	}

	if err := wk.addRunResultToCurrentJobContext(ctx, response.RunResult); err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return nil, err
	}

	return &response, nil
}

func (wk *CurrentWorker) addRunResultToCurrentJobContext(_ context.Context, newRunResult *sdk.V2WorkflowRunResult) error {
	jobContext, has := wk.currentJobV2.runJobContext.Jobs[wk.currentJobV2.runJob.JobID]
	if !has {
		jobContext = sdk.JobResultContext{}
	}
	switch newRunResult.Type {
	case sdk.V2WorkflowRunResultTypeVariable, "V2WorkflowRunResultVariableDetail":
		x, err := newRunResult.GetDetailAsV2WorkflowRunResultVariableDetail()
		if err != nil {
			return err
		} else {
			jobContext.Outputs[x.Name] = x.Value
		}
	default:
		if jobContext.JobRunResults == nil {
			jobContext.JobRunResults = sdk.JobRunResults{}
		}
		jobContext.JobRunResults[newRunResult.Name()], _ = newRunResult.GetDetail()
	}
	wk.currentJobV2.runJobContext.Jobs[wk.currentJobV2.runJob.JobID] = jobContext
	return nil
}

var _ workerruntime.Runtime = new(CurrentWorker)

func (wk *CurrentWorker) V2GetProjectKey(ctx context.Context, keyName string, clear bool) (*sdk.ProjectKey, error) {
	k, err := wk.clientV2.V2WorkerProjectGetKey(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, keyName, clear)
	if err != nil {
		return nil, err
	}
	if clear {
		wk.currentJobV2.sensitiveDatas = append(wk.currentJobV2.sensitiveDatas, k.Private)
		wk.blur, err = sdk.NewBlur(wk.currentJobV2.sensitiveDatas)
		if err != nil {
			return nil, err
		}
	}
	return k, err
}

func (wk *CurrentWorker) V2GetJobRun(ctx context.Context) *sdk.V2WorkflowRunJob {
	return wk.currentJobV2.runJob
}

func (wk *CurrentWorker) V2GetJobContext(ctx context.Context) *sdk.WorkflowRunJobsContext {
	return &wk.currentJobV2.runJobContext
}

func (wk *CurrentWorker) V2UpdateRunResult(ctx context.Context, req workerruntime.V2RunResultRequest) (*workerruntime.V2UpdateResultResponse, error) {
	ctx = workerruntime.SetRunJobID(ctx, wk.currentJobV2.runJob.ID)
	var runResult = req.RunResult

	log.Info(ctx, "updating run result %s (%s)", runResult.Name(), runResult.ID)

	// Update the run result on API side
	if err := wk.clientV2.V2QueueJobRunResultUpdate(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, runResult); err != nil {
		ctx := log.ContextWithStackTrace(ctx, err)
		log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to update run result %s", runResult.ID))
		return nil, sdk.NewError(sdk.ErrUnknownError, err)
	}

	duration := time.Since(runResult.IssuedAt)

	if runResult.DataSync == nil || runResult.DataSync.LatestPromotionOrRelease() == nil {
		wk.clientV2.V2QueuePushJobInfo(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, sdk.V2SendJobRunInfo{
			Level:   sdk.WorkflowRunInfoLevelInfo,
			Message: fmt.Sprintf("Job %q issued a new result %q in %.3f seconds", wk.currentJobV2.runJob.JobID, runResult.Name(), duration.Seconds()),
			Time:    time.Now(),
		})

		wk.clientV2.V2QueuePushRunInfo(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, sdk.V2WorkflowRunInfo{
			Level:   sdk.WorkflowRunInfoLevelInfo,
			Message: fmt.Sprintf("Job %q issued a new result %q in %.3f seconds", wk.currentJobV2.runJob.JobID, runResult.Name(), duration.Seconds()),
		})
	} else {
		var isRelease = len(runResult.DataSync.Releases) > 0
		var message = "has promoted"
		if isRelease {
			message = "has released"
		}

		latestRelease := runResult.DataSync.LatestPromotionOrRelease()

		wk.clientV2.V2QueuePushJobInfo(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, sdk.V2SendJobRunInfo{
			Level:   sdk.WorkflowRunInfoLevelInfo,
			Message: fmt.Sprintf("Job %q %s artifact %q to %s", wk.currentJobV2.runJob.JobID, message, runResult.Name(), latestRelease.ToMaturity),
			Time:    time.Now(),
		})

		wk.clientV2.V2QueuePushRunInfo(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, sdk.V2WorkflowRunInfo{
			Level:   sdk.WorkflowRunInfoLevelInfo,
			Message: fmt.Sprintf("Job %q %s artifact %q to %s", wk.currentJobV2.runJob.JobID, message, runResult.Name(), latestRelease.ToMaturity),
		})
	}

	// TODO: save run-result in the current job to populate context

	return &workerruntime.V2UpdateResultResponse{RunResult: runResult}, nil
}

func (wk *CurrentWorker) V2GetRunResult(ctx context.Context, filter workerruntime.V2FilterRunResult) (*workerruntime.V2GetResultResponse, error) {
	ctx = workerruntime.SetRunJobID(ctx, wk.currentJobV2.runJob.ID)

	resp, err := wk.clientV2.V2QueueJobRunResultsGet(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID)
	if err != nil {
		return nil, err
	}

	var result workerruntime.V2GetResultResponse
	if strings.TrimSpace(filter.Pattern) == "" {
		filter.Pattern = "**/*"
	}
	pattern := glob.New(filter.Pattern)
	for _, r := range resp {
		if len(filter.Type) > 0 && !slices.Contains(filter.Type, r.Type) {
			continue
		}
		switch r.Detail.Type {
		case "V2WorkflowRunResultGenericDetail":
			var res *glob.Result
			if r.Type == sdk.V2WorkflowRunResultTypeCoverage || r.Type == sdk.V2WorkflowRunResultTypeGeneric { // If the filter is set to "V2WorkflowRunResultGenericDetail" we can directly check the artifact name. This is the usecase of plugin "downloadArtifact"
				x, _ := r.GetDetailAsV2WorkflowRunResultGenericDetail()
				res, err = pattern.MatchString(x.Name)
			} else {
				res, err = pattern.MatchString(r.Name())
			}
			if err != nil {
				log.Error(ctx, "unable to perform glob expression on %s (%s): %v", r.Name(), r.ID, err)
				continue
			}
			if res != nil {
				result.RunResults = append(result.RunResults, r)
			}
		default:
			res, err := pattern.MatchString(r.Name()) // We match with the implementation of the Name function that depends on V2WorkflowRunResult.Type (docker:image/latest, generic:foo.txt, etc...)
			if err != nil {
				log.Error(ctx, "unable to perform glob expression on %s (%s): %v", r.Name(), r.ID, err)
				continue
			}
			if res != nil {
				result.RunResults = append(result.RunResults, r)
			}
		}
	}

	sig := cdn.Signature{
		JobName:       wk.currentJobV2.runJob.Job.Name,
		RunJobID:      wk.currentJobV2.runJob.ID,
		ProjectKey:    wk.currentJobV2.runJob.ProjectKey,
		WorkflowName:  wk.currentJobV2.runJob.WorkflowName,
		WorkflowRunID: wk.currentJobV2.runJob.WorkflowRunID,
		RunNumber:     wk.currentJobV2.runJob.RunNumber,
		RunAttempt:    wk.currentJobV2.runJob.RunAttempt,
		Region:        wk.currentJobV2.runJob.Region,

		Timestamp: time.Now().UnixNano(),

		Worker: &cdn.SignatureWorker{
			WorkerID:   wk.id,
			WorkerName: wk.Name(),
		},
	}

	signature, err := jws.Sign(wk.signer, sig)
	if err != nil {
		return nil, sdk.NewError(sdk.ErrUnknownError, err)
	}

	result.CDNSignature = signature

	return &result, nil
}

func (wk *CurrentWorker) AddStepOutput(_ context.Context, outputName string, outputValue string) {
	currentStepsStatus := wk.GetCurrentStepsStatus()
	stepName := wk.GetSubStepName()
	stepStatus := currentStepsStatus[stepName]
	if stepStatus.Outputs == nil {
		stepStatus.Outputs = sdk.JobResultOutput{}
	}
	stepStatus.Outputs[outputName] = outputValue
	currentStepsStatus[stepName] = stepStatus
}
