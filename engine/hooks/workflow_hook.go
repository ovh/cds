package hooks

import "github.com/ovh/cds/sdk"

func GetWorkflowHookEventFromBitbucketEvent(event string) string {
	switch event {
	case "repo:refs_changed":
		return sdk.WorkflowHookEventPush
	default:
		return ""
	}
}

func GetWorkflowHookEventFromGithubEvent(event string) string {
	switch event {
	case "push":
		return sdk.WorkflowHookEventPush
	default:
		return ""
	}
}

func GetWorkflowHookEventFromGiteaEvent(event string) string {
	switch event {
	case "push":
		return sdk.WorkflowHookEventPush
	default:
		return ""
	}
}

func GetWorkflowHookEventFromGitlabEvent(event string) string {
	switch event {
	case "Push Hook":
		return sdk.WorkflowHookEventPush
	default:
		return ""
	}
}
