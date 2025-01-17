package hooks

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (s *Service) triggerCheckAnalyses(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	allEnded := true

	repos, err := s.Client.HookRepositoriesList(ctx, hre.VCSServerName, hre.RepositoryName)
	if err != nil {
		return err
	}
	// If there are known repositories, and no project checked, init analyses map
	if len(hre.Analyses) == 0 && len(repos) > 0 {
		log.Info(ctx, "found %d repositories to check analyze", len(repos))
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

	// For each project, check analysis
	for i := range hre.Analyses {
		hreAnl := &hre.Analyses[i]
		// If first time, retrieve analysis by commit
		if hreAnl.AnalyzeID == "" {
			foundAnl := false
			as, err := s.Client.ProjectRepositoryAnalysisList(ctx, hreAnl.ProjectKey, hre.VCSServerName, hre.RepositoryName)
			if err != nil {
				return err
			}
			for _, a := range as {
				if a.Commit == hre.ExtractData.Commit {
					hreAnl.AnalyzeID = a.ID
					hreAnl.Status = a.Status
					foundAnl = true
					break
				}
			}
			if !foundAnl {
				hreAnl.FindRetryCount++
				if hreAnl.FindRetryCount > s.Cfg.OldRepositoryEventRetry {
					hreAnl.Status = sdk.RepositoryAnalysisStatusError
					hreAnl.AnalyzeID = ""
					hreAnl.Error = "unable to find analysis"
				} else {
					allEnded = false
				}

			} else {
				// Reset error for next part
				hreAnl.FindRetryCount = 0
			}
			if hreAnl.Status == sdk.RepositoryAnalysisStatusInProgress {
				allEnded = false
			}
		} else {
			// Check status
			apiAnalysis, err := s.Client.ProjectRepositoryAnalysisGet(ctx, hreAnl.ProjectKey, hre.VCSServerName, hre.RepositoryName, hreAnl.AnalyzeID)
			if err != nil {
				return err
			}
			hreAnl.Status = apiAnalysis.Status
			if hreAnl.Status == sdk.RepositoryAnalysisStatusInProgress {
				hreAnl.FindRetryCount++
				if hreAnl.FindRetryCount > s.Cfg.OldRepositoryEventRetry {
					hreAnl.Status = sdk.RepositoryAnalysisStatusError
					hreAnl.Error = "analysis is too long"
				} else {
					allEnded = false
				}
			}
		}
		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}
	}

	if !allEnded {
		// If there is still analysis to check, wait and renqueue
		s.GoRoutines.Exec(ctx, "hook-repository-event-"+hre.UUID, func(ctx context.Context) {
			time.Sleep(10 * time.Second)
			if err := s.Dao.EnqueueRepositoryEvent(ctx, hre); err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
		})
		return nil
	}

	hre.Status = sdk.HookEventStatusWorkflowHooks
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	return s.triggerGetWorkflowHooks(ctx, hre)

}

func (s *Service) triggerAnalyses(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerAnalyses")
	defer next()

	// If first time
	if len(hre.Analyses) == 0 {
		log.Info(ctx, "triggering analysis for event [%s] %s", hre.EventName, hre.GetFullName())
		if hre.EventName == sdk.WorkflowHookEventNameManual {
			hre.Analyses = []sdk.HookRepositoryEventAnalysis{{
				ProjectKey: hre.ExtractData.Manual.Project,
			}}
		} else {
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
		}
		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}
	}

	// Check analysis status and/or run it
	allEnded := true
	allInError := len(hre.Analyses) > 0
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
					a.Error = apiAnalysis.Data.Error
					if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
						return err
					}
					hre.SignKey = apiAnalysis.Data.SignKeyID
					hre.DeprecatedUsername = apiAnalysis.Data.Initiator.Username()
					hre.DeprecatedUserID = apiAnalysis.Data.Initiator.UserID
					hre.Initiator = apiAnalysis.Data.Initiator
				}
			}
		}
		if a.Status == sdk.RepositoryAnalysisStatusInProgress {
			allEnded = false
		}
		if a.Status != sdk.RepositoryAnalysisStatusError {
			allInError = false
		}
	}

	// If all analysis are in errors
	if allInError {
		if len(hre.Analyses) == 1 {
			hre.LastError = hre.Analyses[0].Error
		} else {
			hre.LastError = "All Repository analyses failed: " + hre.Analyses[0].Error
		}
		hre.Status = sdk.HookEventStatusError
		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			return err
		}
		if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, hre.UUID); err != nil {
			return err
		}
		return nil
	}

	if !allEnded {
		return nil
	}

	hre.Status = sdk.HookEventStatusWorkflowHooks
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}

	return s.triggerGetWorkflowHooks(ctx, hre)
}

func (s *Service) runAnalysis(ctx context.Context, hre *sdk.HookRepositoryEvent, analysis *sdk.HookRepositoryEventAnalysis) error {
	ctx, next := telemetry.Span(ctx, "s.runAnalysis")
	defer next()

	if hre.ExtractData.Ref == "" || hre.ExtractData.Commit == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to find branch and commit from payload: %s", string(hre.Body))
	}

	analyze := sdk.AnalysisRequest{
		RepoName:           hre.RepositoryName,
		VcsName:            hre.VCSServerName,
		ProjectKey:         analysis.ProjectKey,
		Ref:                hre.ExtractData.Ref,
		Commit:             hre.ExtractData.Commit,
		HookEventUUID:      hre.UUID,
		HookEventKey:       cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hre.VCSServerName, hre.RepositoryName), hre.UUID),
		DeprecatedUserID:   hre.DeprecatedUserID,
		DeprecatedAdminMFA: hre.ExtractData.DeprecatedAdminMFA,
		Initiator:          hre.Initiator,
	}
	resp, err := s.Client.ProjectRepositoryAnalysis(ctx, analyze)
	if err != nil {
		return err
	}
	analysis.Status = resp.Status
	analysis.AnalyzeID = resp.AnalysisID
	return nil
}
