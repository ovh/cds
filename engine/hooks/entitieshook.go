package hooks

import (
	"context"
	"strings"

	"github.com/ovh/cds/sdk"
)

func (s *Service) doAnalyzeExecution(ctx context.Context, t *sdk.TaskExecution) error {
	var branch, commit string
	var err error
	switch t.Configuration[sdk.HookConfigVCSType].Value {
	case sdk.VCSTypeGithub:
		return sdk.WithStack(sdk.ErrNotImplemented)
	case sdk.VCSTypeGerrit:
		return sdk.WithStack(sdk.ErrNotImplemented)
	case sdk.VCSTypeGitlab:
		return sdk.WithStack(sdk.ErrNotImplemented)
	case sdk.VCSTypeBitbucketCloud:
		return sdk.WithStack(sdk.ErrNotImplemented)
	case sdk.VCSTypeGitea:
		branch, commit, err = s.extractAnalyzeDataFromGiteaRequest(t.EntitiesHook.RequestBody)
	case sdk.VCSTypeBitbucketServer:
		branch, commit, err = s.extractAnalyzeDataFromBitbucketRequest(t.EntitiesHook.RequestBody)
	default:
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unknown vcs of type: %s", t.Configuration[sdk.HookConfigVCSType].Value)
	}
	if err != nil {
		return err
	}
	if branch == "" || commit == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to find branch and commit from payload: %s", string(t.EntitiesHook.RequestBody))
	}

	analyze := sdk.AnalysisRequest{
		RepoName:   t.Configuration[sdk.HookConfigRepoFullName].Value,
		VcsName:    t.Configuration[sdk.HookConfigVCSServer].Value,
		ProjectKey: t.Configuration[sdk.HookConfigProject].Value,
		Branch:     strings.TrimPrefix(branch, "refs/heads/"),
		Commit:     commit,
	}
	resp, err := s.Client.ProjectRepositoryAnalysis(ctx, analyze)
	if err != nil {
		return err
	}
	t.EntitiesHook.AnalysisID = resp.AnalysisID
	t.EntitiesHook.OperationID = resp.OperationID
	return nil
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
