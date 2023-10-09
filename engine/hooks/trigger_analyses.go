package hooks

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (s *Service) triggerAnalyses(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerAnalyses")
	defer next()

	// If first time
	if len(hre.Analyses) == 0 {
		log.Info(ctx, "triggering analysis for event [%s] %s", hre.EventName, hre.GetFullName())
		repos, err := s.Client.HookRepositoriesList(ctx, hre.VCSServerName, hre.RepositoryName)
		if err != nil {
			return err
		}
		log.Info(ctx, "found %d repositories to analyze", len(repos))
		hre.Analyses = make([]sdk.HookRepositoryEventAnalysis, 0, len(repos))
		for _, r := range repos {
			hre.Analyses = append(hre.Analyses, sdk.HookRepositoryEventAnalysis{
				ProjectKey: r.ProjectKey,
			})
		}
		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}
	}

	// Check analysis status and/or run it
	allEnded := true
	for i := range hre.Analyses {
		a := &hre.Analyses[i]
		if a.Status == "" {
			allEnded = false
			// Call cds api to trigger an analyze
			log.Info(ctx, "run analysis on %s %s/%s", a.ProjectKey, hre.VCSServerName, hre.RepositoryName)
			if err := s.runAnalysis(ctx, hre, a); err != nil {
				return err
			}
			if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
				return err
			}
		} else if a.Status == sdk.RepositoryAnalysisStatusInProgress {
			if time.Now().UnixMilli()-hre.LastUpdate > RetryDelayMilli {

				// If we have to message from API, try to get the analysis result
				apiAnalysis, err := s.Client.ProjectRepositoryAnalysisGet(ctx, a.ProjectKey, hre.VCSServerName, hre.RepositoryName, a.AnalyzeID)
				if err != nil {
					return err
				}
				if apiAnalysis.Status != sdk.RepositoryAnalysisStatusInProgress {
					a.Status = apiAnalysis.Status
					if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
						return err
					}
				}
			}
		}
		if a.Status == sdk.RepositoryAnalysisStatusInProgress {
			allEnded = false
		}
	}
	if !allEnded {
		return nil
	}

	hre.Status = sdk.HookEventStatusWorkflowHooks
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	return s.triggerWorkflowHooks(ctx, hre)
}

func (s *Service) runAnalysis(ctx context.Context, hre *sdk.HookRepositoryEvent, analysis *sdk.HookRepositoryEventAnalysis) error {
	ctx, next := telemetry.Span(ctx, "s.runAnalysis")
	defer next()

	if hre.ExtractData.Branch == "" || hre.ExtractData.Commit == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to find branch and commit from payload: %s", string(hre.Body))
	}

	analyze := sdk.AnalysisRequest{
		RepoName:      hre.RepositoryName,
		VcsName:       hre.VCSServerName,
		ProjectKey:    analysis.ProjectKey,
		Branch:        hre.ExtractData.Branch,
		Commit:        hre.ExtractData.Commit,
		HookEventUUID: hre.UUID,
	}
	resp, err := s.Client.ProjectRepositoryAnalysis(ctx, analyze)
	if err != nil {
		return err
	}
	analysis.Status = resp.Status
	analysis.AnalyzeID = resp.AnalysisID
	return nil
}
