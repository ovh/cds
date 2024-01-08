package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishRunJobRunResult(ctx context.Context, store cache.Store, eventType, vcsName, repoName string, rj sdk.V2WorkflowRunJob, rr sdk.V2WorkflowRunResult) {
	bts, _ := json.Marshal(rr)
	e := sdk.WorkflowRunJobRunResultEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: rj.ProjectKey,
		},
		VCSName:       vcsName,
		Repository:    repoName,
		Workflow:      rj.WorkflowName,
		WorkflowRunID: rj.WorkflowRunID,
		RunJobID:      rj.ID,
		RunNumber:     rj.RunNumber,
		RunAttempt:    rj.RunAttempt,
		Region:        rj.Region,
		Hatchery:      rj.HatcheryName,
		ModelType:     rj.ModelType,
		JobID:         rj.JobID,
		RunResult:     rr.Name(),
		Status:        rr.Status,
		UserID:        rj.UserID,
		Username:      rj.Username,
	}
	publish(ctx, store, e)
}

func PublishRunJobManualEvent(ctx context.Context, store cache.Store, eventType string, wr sdk.V2WorkflowRun, jobID string, gateInputs map[string]interface{}, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(gateInputs)
	e := sdk.WorkflowRunJobManualEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: wr.ProjectKey,
		},
		VCSName:       wr.Contexts.Git.Server,
		Repository:    wr.Contexts.Git.Repository,
		Workflow:      wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        wr.Status,
		WorkflowRunID: wr.ID,
		UserID:        u.ID,
		Username:      u.Username,
		JobID:         jobID,
	}
	publish(ctx, store, e)
}

func PublishRunJobEvent(ctx context.Context, store cache.Store, eventType, vcsName, repoName string, rj sdk.V2WorkflowRunJob) {
	bts, _ := json.Marshal(rj)
	e := sdk.WorkflowRunJobEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: rj.ProjectKey,
		},
		VCSName:       vcsName,
		Repository:    repoName,
		Workflow:      rj.WorkflowName,
		WorkflowRunID: rj.WorkflowRunID,
		RunJobID:      rj.ID,
		RunNumber:     rj.RunNumber,
		RunAttempt:    rj.RunAttempt,
		Region:        rj.Region,
		Hatchery:      rj.HatcheryName,
		ModelType:     rj.ModelType,
		JobID:         rj.JobID,
		Status:        rj.Status,
		UserID:        rj.UserID,
		Username:      rj.Username,
	}
	publish(ctx, store, e)
}

func PublishRunEvent(ctx context.Context, store cache.Store, eventType string, wr sdk.V2WorkflowRun, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(wr)
	e := sdk.WorkflowRunEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: wr.ProjectKey,
		},
		VCSName:       wr.Contexts.Git.Server,
		Repository:    wr.Contexts.Git.Repository,
		Workflow:      wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        wr.Status,
		WorkflowRunID: wr.ID,
		UserID:        u.ID,
		Username:      u.Username,
	}
	publish(ctx, store, e)
}
