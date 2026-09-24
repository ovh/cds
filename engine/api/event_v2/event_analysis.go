package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishAnalysisStart(ctx context.Context, store cache.Store, vcsName, repoName string, a *sdk.ProjectRepositoryAnalysis) {
	bts, _ := json.Marshal(a)
	e := sdk.AnalysisEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      sdk.EventAnalysisStart,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		ProjectEventV2: sdk.ProjectEventV2{
			ProjectKey: a.ProjectKey,
		},
		VCSName:    vcsName,
		Repository: repoName,
		Status:     a.Status,
	}
	publish(ctx, store, e)
}

func PublishAnalysisDone(ctx context.Context, store cache.Store, vcsName, repoName string, a *sdk.ProjectRepositoryAnalysis) {
	bts, _ := json.Marshal(a)
	e := sdk.AnalysisEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      sdk.EventAnalysisDone,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		ProjectEventV2: sdk.ProjectEventV2{
			ProjectKey: a.ProjectKey,
		},
		VCSName:    vcsName,
		Repository: repoName,
		Status:     a.Status,
		Username:   a.Data.InitiatorUsername(),
	}
	// Initiator is nil when the analysis stopped before the committer was resolved
	if a.Data.Initiator != nil {
		e.UserID = a.Data.Initiator.UserID
	}
	publish(ctx, store, e)
}
