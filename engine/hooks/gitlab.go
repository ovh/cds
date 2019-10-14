package hooks

import (
	"encoding/json"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) generatePayloadFromGitlabRequest(t *sdk.TaskExecution, event string) (map[string]interface{}, error) {
	switch event {
	case "Push Hook":
		return s.generatePayloadFromGitlabPushEvent(t)
	default:
		return getAllPayloadMap(t.WebHook.RequestBody)
	}
}

func (s *Service) generatePayloadFromGitlabPushEvent(t *sdk.TaskExecution) (map[string]interface{}, error) {
	payload := make(map[string]interface{})
	projectKey := t.Config["project"].Value
	workflowName := t.Config["workflow"].Value

	var pushEvent GitlabPushEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &pushEvent); err != nil {
		return nil, sdk.WrapError(err, "unable ro read gitlab request: %s", string(t.WebHook.RequestBody))
	}
	// Branch deletion ( gitlab return 0000000000000000000000000000000000000000 as git hash)
	if pushEvent.After == "0000000000000000000000000000000000000000" {
		err := s.enqueueBranchDeletion(projectKey, workflowName, strings.TrimPrefix(pushEvent.Ref, "refs/heads/"))
		return nil, sdk.WrapError(err, "cannot enqueue branch deletion")
	}
	payload["git.author"] = pushEvent.UserUsername
	payload["git.author.email"] = pushEvent.UserEmail
	if !strings.HasPrefix(pushEvent.Ref, "refs/tags/") {
		branch := strings.TrimPrefix(pushEvent.Ref, "refs/heads/")
		payload["git.branch"] = branch
		if err := s.stopBranchDeletionTask(branch); err != nil {
			log.Error("cannot stop branch deletion task for branch %s : %v", branch, err)
		}
	} else {
		payload["git.tag"] = strings.TrimPrefix(pushEvent.Ref, "refs/tags/")
	}
	payload["git.hash.before"] = pushEvent.Before
	payload["git.hash"] = pushEvent.After
	hashShort := pushEvent.After
	if len(hashShort) >= 7 {
		hashShort = hashShort[:7]
	}
	payload["git.hash.short"] = hashShort
	payload["git.repository"] = pushEvent.Project.PathWithNamespace

	payload["cds.triggered_by.username"] = pushEvent.UserUsername
	payload["cds.triggered_by.fullname"] = pushEvent.UserName
	payload["cds.triggered_by.email"] = pushEvent.UserEmail

	if len(pushEvent.Commits) > 0 {
		payload["git.message"] = pushEvent.Commits[0].Message
	}
	payloadStr, err := json.Marshal(pushEvent)
	if err != nil {
		log.Error("Unable to marshal payload: %v", err)
	}
	payload["payload"] = string(payloadStr)
	return payload, nil
}
