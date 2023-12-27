package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishAnalysisStart(ctx context.Context, store cache.Store, vcsName, repoName string, a *sdk.ProjectRepositoryAnalysis, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(a)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: a.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Status:     a.Status,
		Type:       sdk.EventAnalysisStart,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishAnalysisDone(ctx context.Context, store cache.Store, vcsName, repoName string, a *sdk.ProjectRepositoryAnalysis, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(a)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: a.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Status:     a.Status,
		Type:       sdk.EventAnalysisDone,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
