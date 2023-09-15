package hooks

import (
	"context"
	"github.com/rockbears/log"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
)

func (s *Service) triggerAnalyses(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	log.Info(ctx, "triggering analysis for event [%s] %s", hre.EventName, hre.GetFullName())

	// If first time
	if len(hre.Analyses) == 0 {
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
			// Call cds api to trigger an analyze
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
	var branch, commit string
	var err error
	switch hre.VCSServerType {
	case sdk.VCSTypeGithub:
		branch, commit, err = s.extractAnalyzeDataFromGithubRequest(hre.Body)
	case sdk.VCSTypeGitlab:
		branch, commit, err = s.extractAnalyzeDataFromGitlabRequest(hre.Body)
	case sdk.VCSTypeGitea:
		branch, commit, err = s.extractAnalyzeDataFromGiteaRequest(hre.Body)
	case sdk.VCSTypeBitbucketServer:
		branch, commit, err = s.extractAnalyzeDataFromBitbucketRequest(hre.Body)
	case sdk.VCSTypeGerrit:
		return sdk.WithStack(sdk.ErrNotImplemented)
	case sdk.VCSTypeBitbucketCloud:
		return sdk.WithStack(sdk.ErrNotImplemented)
	default:
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unknown vcs of type: %s", hre.VCSServerType)
	}
	if err != nil {
		return err
	}
	if branch == "" || commit == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to find branch and commit from payload: %s", string(hre.Body))
	}

	analyze := sdk.AnalysisRequest{
		RepoName:      hre.RepositoryName,
		VcsName:       hre.VCSServerName,
		ProjectKey:    analysis.ProjectKey,
		Branch:        strings.TrimPrefix(branch, "refs/heads/"),
		Commit:        commit,
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

func (s *Service) extractAnalyzeDataFromGitlabRequest(body []byte) (string, string, error) {
	var request GitlabEvent
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", "", sdk.WrapError(err, "unable ro read gitlab request: %s", string(body))
	}
	return request.Ref, request.After, nil
}

func (s *Service) extractAnalyzeDataFromGithubRequest(body []byte) (string, string, error) {
	var request GithubWebHookEvent
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", "", sdk.WrapError(err, "unable ro read github request: %s", string(body))
	}
	return request.Ref, request.After, nil
}

func (s *Service) extractAnalyzeDataFromBitbucketRequest(body []byte) (string, string, error) {
	var request sdk.BitbucketServerWebhookEvent
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", "", sdk.WrapError(err, "unable ro read bitbucket request: %s", string(body))
	}
	if len(request.Changes) == 0 {
		return "", "", sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to know branch and commit: %s", string(body))
	}
	return request.Changes[0].RefID, request.Changes[0].ToHash, nil
}

func (s *Service) extractAnalyzeDataFromGiteaRequest(body []byte) (string, string, error) {
	var request GiteaEventPayload
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", "", sdk.WrapError(err, "unable ro read gitea request: %s", string(body))
	}
	return request.Ref, request.After, nil
}
