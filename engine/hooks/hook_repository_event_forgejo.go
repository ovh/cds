package hooks

import (
	"context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"
	"github.com/rockbears/log"
)

func (s *Service) extractDataFromForgejoPushEvent(ctx context.Context, body []byte) (string, sdk.HookRepositoryEventExtractData, error) {
	extractedData := sdk.HookRepositoryEventExtractData{}
	var request ForgejoPushPayload
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", extractedData, sdk.WrapError(err, "unable ro read forgejo push event: %s", string(body))
	}
	repoName := request.Repository.FullName

	extractedData.CDSEventName = sdk.WorkflowHookEventNamePush
	extractedData.CDSEventType = "" // nothing here
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

	if !extractedData.CDSEventType.IsValidForEventName(extractedData.CDSEventName) {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown action %q for event %q", extractedData.CDSEventType, extractedData.CDSEventName)
	}
	if request.HeadCommit.Verification != nil {
		extractedData.CommitVerified = request.HeadCommit.Verification.Verified
		keyID, err := gpg.GetKeyIdFromSignature(request.HeadCommit.Verification.Signature)
		if err != nil {
			log.Warn(ctx, "unable to get gpg key id from signature: %v", err)
		}
		extractedData.CommitGpgKeyID = keyID
	}

	return repoName, extractedData, nil
}

func (s *Service) extractDataFromForgejoPullRequestCommentEvent(body []byte) (string, sdk.HookRepositoryEventExtractData, error) {
	extractedData := sdk.HookRepositoryEventExtractData{}
	var request ForgejoPullRequestPayload
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", extractedData, sdk.WrapError(err, "unable ro read gitea request: %s", string(body))
	}

	// X-Forgejo-Event: pull_request_comment
	// X-Forgejo-Event-Type: pull_request_review_comment
	// action: reviewed

	repoName := request.Repository.FullName
	extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequestComment
	extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestCommentCreated
	extractedData.Ref = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
	extractedData.Commit = request.PullRequest.Head.Sha
	extractedData.CommitFrom = request.PullRequest.Base.Sha
	extractedData.PullRequestRefTo = sdk.GitRefBranchPrefix + request.PullRequest.Base.Ref
	extractedData.PullRequestID = int64(request.PullRequest.Number)

	return repoName, extractedData, nil
}

func (s *Service) extractDataFromForgejoPullRequestEvent(body []byte, eventType string) (string, sdk.HookRepositoryEventExtractData, error) {
	extractedData := sdk.HookRepositoryEventExtractData{}
	var request ForgejoPullRequestPayload
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", extractedData, sdk.WrapError(err, "unable ro read gitea request: %s", string(body))
	}

	// X-Forgejo-Event: pull_request
	// X-Forgejo-Event-Type: pull_request

	repoName := request.Repository.FullName
	// opened reopened closed
	extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequest

	if eventType == string(ForgejoEventTypePullRequest) {
		switch request.Action {
		case HookIssueOpened:
			extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestOpened
		case HookIssueClosed:
			extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestClosed
		case HookIssueReOpened:
			extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestReopened
		default:
			// HookIssueEdited not managed as it's for update title/description of the PR
			return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown action %q for event %q and type %q", extractedData.CDSEventType, extractedData.CDSEventName, eventType)
		}
	} else if eventType == string(ForgejoEventTypePullRequestSync) && request.Action == HookIssueSynchronized {
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestEdited
	} else {
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown action %q for event %q and type %q", extractedData.CDSEventType, extractedData.CDSEventName, eventType)
	}

	extractedData.Ref = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
	extractedData.Commit = request.PullRequest.Head.Sha
	extractedData.CommitFrom = request.PullRequest.Base.Sha
	extractedData.PullRequestRefTo = sdk.GitRefBranchPrefix + request.PullRequest.Base.Ref
	extractedData.PullRequestID = int64(request.PullRequest.Number)

	return repoName, extractedData, nil
}

func (s *Service) extractDataFromForgejoIssueCommentPREvent(body []byte) (string, sdk.HookRepositoryEventExtractData, error) {
	extractedData := sdk.HookRepositoryEventExtractData{}
	var request ForgejoIssueCommentPayload
	if err := sdk.JSONUnmarshal(body, &request); err != nil {
		return "", extractedData, sdk.WrapError(err, "unable ro read gitea request: %s", string(body))
	}

	// X-Forgejo-Event: issue_comment
	// X-Forgejo-Event-Type: pull_request_comment

	repoName := request.Repository.FullName
	extractedData.CDSEventName = sdk.WorkflowHookEventNamePullRequestComment

	switch request.Action {
	case HookIssueCommentCreated:
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestCommentCreated
	case HookIssueCommentEdited:
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestCommentEdited
	case HookIssueCommentDeleted:
		extractedData.CDSEventType = sdk.WorkflowHookEventTypePullRequestCommentDeleted
	default:
		return "", extractedData, sdk.NewErrorFrom(sdk.ErrNotImplemented, "unknown action %q for event %q", extractedData.CDSEventType, extractedData.CDSEventName)
	}
	extractedData.PullRequestID = int64(request.PullRequest.ID)
	extractedData.Ref = sdk.GitRefBranchPrefix + request.PullRequest.Head.Ref
	extractedData.Commit = request.PullRequest.Head.Sha
	extractedData.CommitFrom = request.PullRequest.Base.Sha
	extractedData.PullRequestRefTo = sdk.GitRefBranchPrefix + request.PullRequest.Base.Ref

	return repoName, extractedData, nil
}
