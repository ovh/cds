package hooks

import (
	"strings"

	"github.com/ovh/cds/sdk"
)

func (s *Service) extractDataFromPayload(vcsServerType string, body []byte) (string, sdk.HookRepositoryEventExtractData, error) {
	switch vcsServerType {
	case sdk.VCSTypeBitbucketServer:
		return s.extractDataFromBitbucketRequest(body)
	case sdk.VCSTypeGithub:
		return s.extractDataFromGithubRequest(body)
	case sdk.VCSTypeGitlab:
		return s.extractDataFromGitlabRequest(body)
	case sdk.VCSTypeGitea:
		return s.extractDataFromGiteaRequest(body)
	default:
		return "", sdk.HookRepositoryEventExtractData{}, sdk.WithStack(sdk.ErrNotImplemented)
	}
}

// Update file paths are not is gitea payload
func (s *Service) extractDataFromGiteaRequest(body []byte) (string, sdk.HookRepositoryEventExtractData, error) {
	extractedData := sdk.HookRepositoryEventExtractData{}
	var request GiteaEventPayload
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", extractedData, sdk.WrapError(err, "unable ro read gitea request: %s", string(body))
	}

	repoName := request.Repository.FullName
	extractedData.Ref = request.Ref

	extractedData.Commit = request.After

	return repoName, extractedData, nil
}

func (s *Service) extractDataFromGitlabRequest(body []byte) (string, sdk.HookRepositoryEventExtractData, error) {
	extractedData := sdk.HookRepositoryEventExtractData{
		Paths: make([]string, 0),
	}
	var request GitlabEvent
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", extractedData, sdk.WrapError(err, "unable ro read gitlab request: %s", string(body))
	}
	var repoName string
	if request.Project != nil {
		repoName = request.Project.PathWithNamespace
	}
	extractedData.Ref = request.Ref
	extractedData.Commit = request.After

	for _, c := range request.Commits {
		extractedData.Paths = append(extractedData.Paths, c.Added...)
		extractedData.Paths = append(extractedData.Paths, c.Modified...)
		extractedData.Paths = append(extractedData.Paths, c.Removed...)
	}

	return repoName, extractedData, nil
}

func (s *Service) extractDataFromGithubRequest(body []byte) (string, sdk.HookRepositoryEventExtractData, error) {
	extractedData := sdk.HookRepositoryEventExtractData{
		Paths: make([]string, 0),
	}
	var request GithubWebHookEvent
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", extractedData, sdk.WrapError(err, "unable ro read github request: %s", string(body))
	}
	extractedData.Ref = request.Ref
	extractedData.Commit = request.After

	var repoName string
	if request.Repository != nil {
		repoName = request.Repository.FullName
	}

	for _, c := range request.Commits {
		extractedData.Paths = append(extractedData.Paths, c.Added...)
		extractedData.Paths = append(extractedData.Paths, c.Modified...)
		extractedData.Paths = append(extractedData.Paths, c.Removed...)
	}
	return repoName, extractedData, nil
}

// Update file paths are not is gitea payload
func (s *Service) extractDataFromBitbucketRequest(body []byte) (string, sdk.HookRepositoryEventExtractData, error) {
	extractedData := sdk.HookRepositoryEventExtractData{}
	var request sdk.BitbucketServerWebhookEvent
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", extractedData, sdk.WrapError(err, "unable ro read bitbucket request: %s", string(body))
	}
	var repoName string
	if request.Repository != nil {
		repoName = strings.ToLower(request.Repository.Project.Key) + "/" + request.Repository.Slug
	}
	switch request.EventKey {
	case "repo:refs_changed":
		extractedData.Ref = request.Changes[0].RefID
		extractedData.Commit = request.Changes[0].ToHash
	case "pr:opened", "pr:from_ref_updated":
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
	}
	if request.EventKey == "repo:refs_changed" {
		extractedData.Ref = request.Changes[0].RefID
		extractedData.Commit = request.Changes[0].ToHash
	}

	if extractedData.Ref == "" {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrInvalidData, "repoName: %v unable to know branch and commit: %s", repoName, string(body))
	}

	return repoName, extractedData, nil
}
