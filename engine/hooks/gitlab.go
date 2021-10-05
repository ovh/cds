package hooks

import (
	"context"
	"strings"

	"github.com/ovh/cds/sdk"
)

func (s *Service) generatePayloadFromGitlabRequest(ctx context.Context, t *sdk.TaskExecution, event string) (map[string]interface{}, error) {
	var request GitlabEvent
	if err := sdk.JSONUnmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable ro read gitlab request: %s", string(t.WebHook.RequestBody))
	}

	// Branch deletion ( gitlab return 0000000000000000000000000000000000000000 as git hash)
	if request.After == "0000000000000000000000000000000000000000" {
		return nil, nil
	}

	payload := make(map[string]interface{})

	payload[GIT_EVENT] = event

	payload[GIT_AUTHOR] = request.UserUsername
	payload[GIT_AUTHOR_EMAIL] = request.UserEmail
	payload[CDS_TRIGGERED_BY_USERNAME] = request.UserUsername
	payload[CDS_TRIGGERED_BY_FULLNAME] = request.UserName
	payload[CDS_TRIGGERED_BY_EMAIL] = request.UserEmail

	if request.Ref != "" {
		if !strings.HasPrefix(request.Ref, "refs/tags/") {
			branch := strings.TrimPrefix(request.Ref, "refs/heads/")
			payload[GIT_BRANCH] = branch
		} else {
			payload[GIT_TAG] = strings.TrimPrefix(request.Ref, "refs/tags/")
		}
	}
	if request.Before != "" {
		payload[GIT_HASH_BEFORE] = request.Before
	}
	if request.After != "" {
		payload[GIT_HASH] = request.After
		payload[GIT_HASH_SHORT] = sdk.StringFirstN(request.After, 7)
	}

	getPayloadFromGitlabProject(payload, request.Project)
	getPayloadFromGitlabCommit(payload, request.Commits)
	getPayloadStringVariable(ctx, payload, request)

	return payload, nil
}

func getPayloadFromGitlabCommit(payload map[string]interface{}, commits []GitlabCommit) {
	if len(commits) == 0 {
		return
	}
	payload[GIT_MESSAGE] = commits[0].Message
}

func getPayloadFromGitlabProject(payload map[string]interface{}, project *GitlabProject) {
	if project == nil {
		return
	}
	payload[GIT_REPOSITORY] = project.PathWithNamespace
}
