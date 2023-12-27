package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishRunJobRunResult(ctx context.Context, store cache.Store, eventType, vcsName, repoName string, rj sdk.V2WorkflowRunJob, rr sdk.V2WorkflowRunResult) {
	bts, _ := json.Marshal(rr)
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
		Payload:       bts,
	}
	publish(ctx, store, e)
}

func PublishRunJobEvent(ctx context.Context, store cache.Store, eventType, vcsName, repoName string, rj sdk.V2WorkflowRunJob) {
	bts, _ := json.Marshal(rj)
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
		Payload:       bts,
	}
	publish(ctx, store, e)
}

func PublishRunEvent(ctx context.Context, store cache.Store, eventType string, wr sdk.V2WorkflowRun, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(wr)
	e := sdk.EventV2{
		ID:            sdk.UUID(),
		ProjectKey:    wr.ProjectKey,
		VCSName:       wr.Contexts.Git.Server,
		Repository:    wr.Contexts.Git.Repository,
		Workflow:      wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Type:          eventType,
		Status:        wr.Status,
		Payload:       bts,
		WorkflowRunID: wr.ID,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
