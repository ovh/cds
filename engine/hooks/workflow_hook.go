package hooks

import "github.com/ovh/cds/sdk"

func GetWorkflowHookEventFromBitbucketEvent(event string) string {
	switch event {
	case "repo:refs_changed":
		return sdk.WorkflowHookEventPush
	case "pr:opened", "pr:from_ref_updated":
		return sdk.WorkflowHookEventPullRequest
	default:
		return ""
	}
}

func GetWorkflowHookEventFromGithubEvent(event string) string {
	switch event {
	case "push":
		return sdk.WorkflowHookEventPush
	case "pull_request":
		return sdk.WorkflowHookEventPullRequest
	default:
		return ""
	}
}

func GetWorkflowHookEventFromGiteaEvent(event string) string {
	switch event {
	case "push":
		return sdk.WorkflowHookEventPush
	case "pull_request":
		return sdk.WorkflowHookEventPullRequest
	default:
		return ""
	}
}

func GetWorkflowHookEventFromGitlabEvent(event string) string {
	switch event {
	case "Push Hook":
		return sdk.WorkflowHookEventPush
	case "Merge Request Hook":
		return sdk.WorkflowHookEventPullRequest
	default:
		return ""
	}
}
