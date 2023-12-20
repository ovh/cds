package event_v2

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishRunJobRunResult(ctx context.Context, store cache.Store, eventType, vcsName, repoName string, rj sdk.V2WorkflowRunJob, rr sdk.V2WorkflowRunResult) {
	e := sdk.EventV2{
		ID:            sdk.UUID(),
		ProjectKey:    rj.ProjectKey,
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
		Type:          eventType,
		RunResultName: rr.Name(),
		Status:        rr.Status,
		Payload:       rr,
	}
	publish(ctx, store, e)
}

func PublishRunJobEvent(ctx context.Context, store cache.Store, eventType, vcsName, repoName string, rj sdk.V2WorkflowRunJob) {
	e := sdk.EventV2{
		ID:            sdk.UUID(),
		ProjectKey:    rj.ProjectKey,
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
		Type:          eventType,
		Status:        rj.Status,
		Payload:       rj,
	}
	publish(ctx, store, e)
}
