package hooks

import (
	"context"
	"net/http"
	"strings"

	"github.com/ovh/cds/sdk"
)

const (
	NoCommit = "0000000000000000000000000000000000000000"
)

func (s *Service) extractDataFromPayload(ctx context.Context, _ http.Header, vcsServerType string, body []byte, eventName, eventType string) (string, sdk.HookRepositoryEventExtractData, error) {
	switch vcsServerType {
	case sdk.VCSTypeBitbucketServer:
		return s.extractDataFromBitbucketRequest(body)
	case sdk.VCSTypeGithub:
		return s.extractDataFromGithubRequest(body, eventName)
	case sdk.VCSTypeGitlab:
		return s.extractDataFromGitlabRequest(body, eventName)
	case sdk.VCSTypeGitea:
		return s.extractDataFromGiteaRequest(body, eventName)
	case sdk.VCSTypeForgejo:
		return s.extractDataFromForgejoRequest(ctx, body, eventName, eventType)
	default:
		return "", sdk.HookRepositoryEventExtractData{}, sdk.WithStack(sdk.ErrNotImplemented)
	}
}

func (s *Service) extractDataFromForgejoRequest(ctx context.Context, body []byte, eventName string, eventType string) (string, sdk.HookRepositoryEventExtractData, error) {
	switch eventName {
	case string(ForgejoEventPush):
		return s.extractDataFromForgejoPushEvent(ctx, body)
	case string(ForgejoEventPullRequest):
		return s.extractDataFromForgejoPullRequestEvent(body, eventType)
	case string(ForgejoEventPullRequestComment): // Signle comment during a review
		return s.extractDataFromForgejoPullRequestCommentEvent(body)
	case string(ForgejoEventIssueComment): // Comment on a pull request or issue
		switch eventType {
		case string(ForgejoEventTypePullRequestComment):
			return s.extractDataFromForgejoIssueCommentPREvent(body)
		default:
			return "", sdk.HookRepositoryEventExtractData{}, sdk.WrapError(sdk.ErrNotImplemented, "event %q type %q not implemented", eventName, eventType)
		}
	default:
		return "", sdk.HookRepositoryEventExtractData{}, sdk.WrapError(sdk.ErrNotImplemented, "event %q type %q not implemented", eventName, eventType)
	}
}

// Update file paths are not is gitea payload
func (s *Service) extractDataFromGiteaRequest(body []byte, eventName string) (string, sdk.HookRepositoryEventExtractData, error) {
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
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePush
		extractedData.CDSEventType = "" // nothing here
		extractedData.Ref = request.Ref
		extractedData.Commit = request.After
		if request.Before != NoCommit {
			extractedData.CommitFrom = request.Before
		}
	case "pull_request":
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequest
		switch request.Action {
		case "synchronized":
			extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestEdited
		default:
			extractedData.CDSEventType = sdk.WorkflowHookEventType(request.Action)
		}
		extractedData.Ref = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
		extractedData.Commit = request.PullRequest.Head.Sha
		extractedData.CommitFrom = request.PullRequest.Base.Sha
		extractedData.PullRequestRefTo = sdk.GitRefBranchPrefix + request.PullRequest.Base.Ref
		extractedData.PullRequestRefFrom = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
		extractedData.PullRequestID = int64(request.PullRequest.Number)
		// On merge, run against the merge result on the target branch. base.sha is refreshed to
		// the post-merge tip (== merge commit), so use merge_base as the changeset origin.
		if request.PullRequest.Merged {
			extractedData.Ref = sdk.GitRefBranchPrefix + request.PullRequest.Base.Ref
			extractedData.Commit = request.PullRequest.MergeCommitSha
			extractedData.CommitFrom = request.PullRequest.MergeBase
		}
	default:
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown event %q", eventName)
	}

	for _, c := range request.Commits {
		extractedData.Paths = append(extractedData.Paths, c.Added...)
		extractedData.Paths = append(extractedData.Paths, c.Modified...)
		extractedData.Paths = append(extractedData.Paths, c.Removed...)
	}

	if !extractedData.CDSEventType.IsValidForEventName(extractedData.CDSEventName) {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown action %q for event %q", extractedData.CDSEventType, extractedData.CDSEventName)
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
	if request.Before != NoCommit {
		extractedData.CommitFrom = request.Before
	}

	for _, c := range request.Commits {
		extractedData.Paths = append(extractedData.Paths, c.Added...)
		extractedData.Paths = append(extractedData.Paths, c.Modified...)
		extractedData.Paths = append(extractedData.Paths, c.Removed...)
	}

	switch eventName {
	case "Push Hook":
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePush
		extractedData.CDSEventType = "" // nothing here
	case "Merge Request Hook":
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequest
		extractedData.CDSEventType = "" // nothing here
	case "Note Hook":
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequestComment
		extractedData.CDSEventType = "" // nothing here
	default:
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown event %q", eventName)
	}

	if !extractedData.CDSEventType.IsValidForEventName(extractedData.CDSEventName) {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown action %q for event %q", extractedData.CDSEventType, extractedData.CDSEventName)
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
	if request.Before != NoCommit {
		extractedData.CommitFrom = request.Before
	}

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
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePush
		extractedData.CDSEventType = "" // nothing here
	case "pull_request":
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventType(request.Action)
		if request.PullRequest != nil {
			extractedData.PullRequestID = request.PullRequest.ID
			extractedData.Commit = request.PullRequest.Head.Sha
			extractedData.Ref = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
			extractedData.CommitFrom = request.PullRequest.Base.Sha
			extractedData.PullRequestRefTo = sdk.GitRefBranchPrefix + request.PullRequest.Base.Ref
			extractedData.PullRequestRefFrom = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
			// On merge, run against the merge result on the target branch
			if request.PullRequest.Merged {
				extractedData.Ref = sdk.GitRefBranchPrefix + request.PullRequest.Base.Ref
				extractedData.Commit = request.PullRequest.MergeCommitSha
			}
		}
	case "pull_request_review_comment":
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequestComment
		extractedData.CDSEventType = sdk.WorkflowHookEventType(request.Action)
		if request.PullRequest != nil {
			extractedData.PullRequestID = request.PullRequest.ID
			extractedData.Commit = request.PullRequest.Head.Sha
			extractedData.Ref = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
			extractedData.CommitFrom = request.PullRequest.Base.Sha
			extractedData.PullRequestRefTo = sdk.GitRefBranchPrefix + request.PullRequest.Base.Ref
			extractedData.PullRequestRefFrom = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
		}
		if request.Comment != nil {
			extractedData.Comment = request.Comment.Body
		}
	case "issue_comment":
		// TODO: comment on a PR conversation; the issue_comment payload carries no
		// head/base refs, so the PR refs must be resolved through the VCS API before
		// the event can be triggered. Not supported yet.
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "event %q not supported yet", eventName)
	default:
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown event %q", eventName)
	}

	if !extractedData.CDSEventType.IsValidForEventName(extractedData.CDSEventName) {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown action %q for event %q", extractedData.CDSEventType, extractedData.CDSEventName)
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
		if request.Changes[0].FromHash != NoCommit {
			extractedData.CommitFrom = request.Changes[0].FromHash
		}
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePush
		extractedData.CDSEventType = "" // no type here
	case "pr:opened":
		extractedData.PullRequestID = int64(request.PullRequest.ID)
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestOpened
		extractedData.CommitFrom = request.PullRequest.ToRef.LatestCommit
		extractedData.PullRequestRefTo = request.PullRequest.ToRef.ID
	case "pr:reopened":
		extractedData.PullRequestID = int64(request.PullRequest.ID)
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestReopened
		extractedData.CommitFrom = request.PullRequest.ToRef.LatestCommit
		extractedData.PullRequestRefTo = request.PullRequest.ToRef.ID
	case "pr:declined":
		extractedData.PullRequestID = int64(request.PullRequest.ID)
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestClosed
		extractedData.CommitFrom = request.PullRequest.ToRef.LatestCommit
		extractedData.PullRequestRefTo = request.PullRequest.ToRef.ID
	case "pr:merged":
		// Merge is reported as a closed pull request (same as GitHub/Forgejo); the run can
		// tell a merge from a decline through the event payload. Run against the merge result
		// on the target branch (merge commit), with the pre-merge target tip as changeset origin.
		extractedData.PullRequestID = int64(request.PullRequest.ID)
		extractedData.Ref = request.PullRequest.ToRef.ID
		extractedData.Commit = request.PullRequest.Properties.MergeCommit.ID
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestClosed
		extractedData.CommitFrom = request.PullRequest.ToRef.LatestCommit
		extractedData.PullRequestRefTo = request.PullRequest.ToRef.ID
	case "pr:from_ref_updated":
		extractedData.PullRequestID = int64(request.PullRequest.ID)
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequest
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestEdited
		extractedData.CommitFrom = request.PullRequest.ToRef.LatestCommit
		extractedData.PullRequestRefTo = request.PullRequest.ToRef.ID
	case "pr:comment:added":
		extractedData.PullRequestID = int64(request.PullRequest.ID)
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequestComment
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestCommentCreated
		extractedData.CommitFrom = request.PullRequest.ToRef.LatestCommit
		extractedData.PullRequestRefTo = request.PullRequest.ToRef.ID
	case "pr:comment:edited":
		extractedData.PullRequestID = int64(request.PullRequest.ID)
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequestComment
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestCommentEdited
		extractedData.CommitFrom = request.PullRequest.ToRef.LatestCommit
		extractedData.PullRequestRefTo = request.PullRequest.ToRef.ID
	case "pr:comment:deleted":
		extractedData.PullRequestID = int64(request.PullRequest.ID)
		extractedData.Ref = request.PullRequest.FromRef.ID
		extractedData.Commit = request.PullRequest.FromRef.LatestCommit
		extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequestComment
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestCommentDeleted
		extractedData.CommitFrom = request.PullRequest.ToRef.LatestCommit
		extractedData.PullRequestRefTo = request.PullRequest.ToRef.ID
	default:
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown event %q", request.EventKey)
	}

	if request.PullRequest != nil {
		extractedData.PullRequestRefFrom = request.PullRequest.FromRef.ID
	}

	if request.Comment != nil {
		extractedData.Comment = request.Comment.Text
	}

	if extractedData.Ref == "" {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrInvalidData, "repoName: %v unable to extract data %s", repoName, string(body))
	}

	if !extractedData.CDSEventType.IsValidForEventName(extractedData.CDSEventName) {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown action %q for event %q", extractedData.CDSEventType, extractedData.CDSEventName)
	}

	return repoName, extractedData, nil
}

func (s *Service) pushInsightReport(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	var projKey string
	if len(hre.Analyses) > 0 {
		projKey = hre.Analyses[0].ProjectKey
	} else if len(hre.WorkflowHooks) > 0 {
		projKey = hre.WorkflowHooks[0].ProjectKey
	} else {
		return nil
	}
	report := hre.ToInsightReport(s.UIURL)
	return s.Client.CreateInsightReport(ctx, projKey, hre.VCSServerName, hre.RepositoryName, hre.ExtractData.Commit, "cds-event", report)
}
