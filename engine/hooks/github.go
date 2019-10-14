package hooks

import (
	"encoding/json"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) generatePayloadFromGithubRequest(t *sdk.TaskExecution, event string) (map[string]interface{}, error) {
	switch event {
	case "push":
		return s.generatePayloadFromGithubPushEvent(t)
	default:
		return getAllPayloadMap(t.WebHook.RequestBody)
	}
}

func (s *Service) generatePayloadFromGithubPushEvent(t *sdk.TaskExecution) (map[string]interface{}, error) {
	payload := make(map[string]interface{})
	projectKey := t.Config["project"].Value
	workflowName := t.Config["workflow"].Value

	var pushEvent GithubPushEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &pushEvent); err != nil {
		return nil, sdk.WrapError(err, "unable ro read github request: %s", string(t.WebHook.RequestBody))
	}
	branch := strings.TrimPrefix(pushEvent.Ref, "refs/heads/")
	if pushEvent.Deleted {
		err := s.enqueueBranchDeletion(projectKey, workflowName, branch)

		return nil, sdk.WrapError(err, "cannot enqueue branch deletion")
	}
	if err := s.stopBranchDeletionTask(branch); err != nil {
		log.Error("cannot stop branch deletion task for branch %s : %v", branch, err)
	}
	payload["git.author"] = pushEvent.HeadCommit.Author.Username
	payload["git.author.email"] = pushEvent.HeadCommit.Author.Email

	if !strings.HasPrefix(pushEvent.Ref, "refs/tags/") {
		payload["git.branch"] = branch
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
	payload["git.repository"] = pushEvent.Repository.FullName
	payload["cds.triggered_by.username"] = pushEvent.HeadCommit.Author.Username
	payload["cds.triggered_by.fullname"] = pushEvent.HeadCommit.Author.Name
	payload["cds.triggered_by.email"] = pushEvent.HeadCommit.Author.Email

	if len(pushEvent.Commits) > 0 {
		payload["git.message"] = pushEvent.Commits[0].Message
	}
	for i := range pushEvent.Commits {
		pushEvent.Commits[i].Added = nil
		pushEvent.Commits[i].Removed = nil
		pushEvent.Commits[i].Modified = nil
	}
	payloadStr, err := json.Marshal(pushEvent)
	if err != nil {
		log.Error("Unable to marshal payload: %v", err)
	}
	payload["payload"] = string(payloadStr)
	return payload, nil
}
