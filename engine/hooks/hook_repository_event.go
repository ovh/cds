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
	extractedData.Branch = strings.TrimPrefix(request.Ref, "refs/heads/")
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
	extractedData.Branch = strings.TrimPrefix(request.Ref, "refs/heads/")
	extractedData.Commit = request.After

	for _, c := range request.Commits {
		for _, p := range c.Added {
			extractedData.Paths = append(extractedData.Paths, p)
		}
		for _, p := range c.Modified {
			extractedData.Paths = append(extractedData.Paths, p)
		}
		for _, p := range c.Removed {
			extractedData.Paths = append(extractedData.Paths, p)
		}
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
	extractedData.Branch = strings.TrimPrefix(request.Ref, "refs/heads/")
	extractedData.Commit = request.After

	var repoName string
	if request.Repository != nil {
		repoName = request.Repository.FullName
	}

	for _, c := range request.Commits {
		for _, p := range c.Added {
			extractedData.Paths = append(extractedData.Paths, p)
		}
		for _, p := range c.Modified {
			extractedData.Paths = append(extractedData.Paths, p)
		}
		for _, p := range c.Removed {
			extractedData.Paths = append(extractedData.Paths, p)
		}
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
	if len(request.Changes) == 0 {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to know branch and commit: %s", string(body))
	}
	extractedData.Branch = strings.TrimPrefix(request.Changes[0].RefID, "refs/heads/")
	extractedData.Commit = request.Changes[0].ToHash
	return repoName, extractedData, nil
}
