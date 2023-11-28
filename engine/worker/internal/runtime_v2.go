package internal

import (
	"context"
	"fmt"
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
	var runResult = req.RunResult

	// Create the run result on API side
	if err := wk.clientV2.V2QueueJobRunResultCreate(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, runResult); err != nil {
		return nil, sdk.NewError(sdk.ErrUnknownError, err)
	}

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

	response := workerruntime.V2AddResultResponse{
		RunResult:    runResult,
		CDNSignature: signature,
		CDNAddress:   wk.CDNHttpURL(),
	}

	return &response, nil
}

var _ workerruntime.Runtime = new(CurrentWorker)

func (wk *CurrentWorker) V2UpdateRunResult(ctx context.Context, req workerruntime.V2RunResultRequest) (*workerruntime.V2UpdateResultResponse, error) {
	ctx = workerruntime.SetRunJobID(ctx, wk.currentJobV2.runJob.ID)
	var runResult = req.RunResult

	runResult.Status = sdk.V2WorkflowRunResultStatusCompleted

	log.Info(ctx, "updating run result %s to status completed", runResult.ID)

	// TODO compute CDN Item links and push it into RunResult object

	// Update the run result on API side
	if err := wk.clientV2.V2QueueJobRunResultUpdate(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, runResult); err != nil {
		return nil, sdk.NewError(sdk.ErrUnknownError, err)
	}

	duration := time.Since(runResult.IssuedAt)
	wk.clientV2.V2QueuePushJobInfo(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, sdk.V2SendJobRunInfo{
		Level:   sdk.WorkflowRunInfoLevelInfo,
		Message: fmt.Sprintf("Job %q issued a new result %q in %.3f seconds", wk.currentJobV2.runJob.JobID, runResult.Name(), duration.Seconds()),
		Time:    time.Now(),
	})

	wk.clientV2.V2QueuePushRunInfo(ctx, wk.currentJobV2.runJob.Region, wk.currentJobV2.runJob.ID, sdk.V2WorkflowRunInfo{
		Level:   sdk.WorkflowRunInfoLevelInfo,
		Message: fmt.Sprintf("Job %q issued a new result %q in %.3f seconds", wk.currentJobV2.runJob.JobID, runResult.Name(), duration.Seconds()),
	})

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
		if r.Type != filter.Type {
			continue
		}
		switch r.Detail.Type {
		case "V2WorkflowRunResultGenericDetail":
			x, _ := r.GetDetailAsV2WorkflowRunResultGenericDetail()
			res, err := pattern.MatchString(x.Name)
			if err != nil {
				log.Error(ctx, "unable to perform glob expression on %s (%s): %v", r.Name(), r.ID, err)
				continue
			}
			if res != nil {
				result.RunResults = append(result.RunResults, r)
			}
		default:
			log.Error(ctx, "unsupported run result detail %q type", r.Detail.Type)
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

func (wk *CurrentWorker) AddStepOutput(ctx context.Context, outputName string, outputValue string) {
	ctx = workerruntime.SetRunJobID(ctx, wk.currentJobV2.runJob.ID)
	stepStatus := wk.currentJobV2.runJob.StepsStatus[wk.currentJobV2.currentStepName]
	if stepStatus.Outputs == nil {
		stepStatus.Outputs = sdk.JobResultOutput{}
	}
	stepStatus.Outputs[outputName] = outputValue
	wk.currentJobV2.runJob.StepsStatus[wk.currentJobV2.currentStepName] = stepStatus
}
