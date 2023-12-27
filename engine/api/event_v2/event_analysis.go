package event_v2

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishAnalysisStart(ctx context.Context, store cache.Store, vcsName, repoName string, a *sdk.ProjectRepositoryAnalysis) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: a.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Status:     a.Status,
		Type:       sdk.EventAnalysisStart,
		Payload:    *a,
	}
	publish(ctx, store, e)
}

func PublishAnalysisDone(ctx context.Context, store cache.Store, vcsName, repoName string, a *sdk.ProjectRepositoryAnalysis) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: a.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Status:     a.Status,
		Type:       sdk.EventAnalysisDone,
		Payload:    *a,
	}
	publish(ctx, store, e)
}
