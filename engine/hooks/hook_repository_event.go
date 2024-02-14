package hooks

import (
	"net/http"
	"strings"

	"github.com/ovh/cds/sdk"
)

func (s *Service) extractDataFromPayload(headers http.Header, vcsServerType string, body []byte, eventName string) (string, sdk.HookRepositoryEventExtractData, error) {
	switch vcsServerType {
	case sdk.VCSTypeBitbucketServer:
		return s.extractDataFromBitbucketRequest(body)
	case sdk.VCSTypeGithub:
		return s.extractDataFromGithubRequest(body, eventName)
	case sdk.VCSTypeGitlab:
		return s.extractDataFromGitlabRequest(body, eventName)
	case sdk.VCSTypeGitea:
		return s.extractDataFromGiteaRequest(headers, body, eventName)
	default:
		return "", sdk.HookRepositoryEventExtractData{}, sdk.WithStack(sdk.ErrNotImplemented)
	}
}

// Update file paths are not is gitea payload
func (s *Service) extractDataFromGiteaRequest(headers http.Header, body []byte, eventName string) (string, sdk.HookRepositoryEventExtractData, error) {
	extractedData := sdk.HookRepositoryEventExtractData{}
	var request GiteaEventPayload
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", extractedData, sdk.WrapError(err, "unable ro read gitea request: %s", string(body))
	}

	repoName := request.Repository.FullName

	// https://github.com/go-gitea/gitea/blob/main/modules/webhook/type.go
	// https://github.com/go-gitea/gitea/blob/main/services/webhook/deliver.go#L128

	switch eventName {
	case "push":
		extractedData.CDSEventName = sdk.WorkflowHookEventPush
		extractedData.CDSEventType = "" // nothing here
		extractedData.Ref = request.Ref
		extractedData.Commit = request.After
	case "pull_request":
		extractedData.Ref = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
		extractedData.Commit = request.PullRequest.Head.Sha
		switch request.Action {
		case "opened":
			extractedData.CDSEventName = sdk.WorkflowHookEventPullRequest
			extractedData.CDSEventType = sdk.WorkflowHookEventPullRequestTypeOpened
		}
	case "pull_request_comment":
		// Not managed. Should needs to get the pull-request detail to get the ref / sha from the pull-request
		// with a comment event, gitea does not send these details
	}

	return repoName, extractedData, nil
}

func (s *Service) extractDataFromGitlabRequest(body []byte, eventName string) (string, sdk.HookRepositoryEventExtractData, error) {
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

	switch eventName {
	case "Push Hook":
		extractedData.CDSEventName = sdk.WorkflowHookEventPush
		extractedData.CDSEventType = "" // nothing here
	case "Merge Request Hook":
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequest
		extractedData.CDSEventType = "" // nothing here
	case "Note Hook":
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequestComment
		extractedData.CDSEventType = "" // nothing here
	}
	return repoName, extractedData, nil
}

func (s *Service) extractDataFromGithubRequest(body []byte, eventName string) (string, sdk.HookRepositoryEventExtractData, error) {
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

	switch eventName {
	case "push":
		extractedData.CDSEventName = sdk.WorkflowHookEventPush
		extractedData.CDSEventType = "" // nothing here
	case "pull_request":
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequest
		extractedData.CDSEventType = request.Action
	case "pull_request_comment":
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequestComment
		extractedData.CDSEventType = request.Action
	}
	return repoName, extractedData, nil
}

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
		extractedData.CDSEventName = sdk.WorkflowHookEventPush
		extractedData.CDSEventType = "" // no type here
	case "pr:opened":
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventPullRequestTypeOpened
	case "pr:reopened":
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventPullRequestTypeReopened
	case "pr:declined":
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventPullRequestTypeClosed
	case "pr:from_ref_updated":
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequest
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventPullRequestTypeEdited
	case "pr:comment:added":
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequestComment
		extractedData.CDSEventType = sdk.WorkflowHookEventPullRequestCommentTypeCreated
	case "pr:comment:edited":
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequestComment
		extractedData.CDSEventType = sdk.WorkflowHookEventPullRequestCommentTypeEdited
	case "pr:comment:deleted":
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventPullRequestComment
		extractedData.CDSEventType = sdk.WorkflowHookEventPullRequestCommentTypeDeleted
	}

	if extractedData.Ref == "" {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrInvalidData, "repoName: %v unable to extract data %s", repoName, string(body))
	}

	return repoName, extractedData, nil
}
