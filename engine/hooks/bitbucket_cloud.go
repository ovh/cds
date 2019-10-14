package hooks

import (
	"encoding/json"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"strings"
)

func (s *Service) generatePayloadFromBitbucketCloudRequest(t *sdk.TaskExecution, event string) ([]map[string]interface{}, error) {
	switch event {
	case "repo:push":
		return s.generatePayloadFromBitbucketCloudPushEvent(t)
	default:
		payloads := []map[string]interface{}{}
		payload, err := getAllPayloadMap(t.WebHook.RequestBody)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
		return payloads, nil
	}
}

func (s *Service) generatePayloadFromBitbucketCloudPushEvent(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	projectKey := t.Config["project"].Value
	workflowName := t.Config["workflow"].Value
	var event BitbucketCloudPushEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &event); err != nil {
		return nil, sdk.WrapError(err, "unable ro read bitbucket request: %s", string(t.WebHook.RequestBody))
	}
	pushEvent := event.Push
	if len(pushEvent.Changes) == 0 {
		return nil, nil
	}

	for _, pushChange := range pushEvent.Changes {
		if pushChange.Closed {
			if pushChange.Old.Type == "branch" {
				err := s.enqueueBranchDeletion(projectKey, workflowName, strings.TrimPrefix(pushChange.Old.Name, "refs/heads/"))
				if err != nil {
					log.Error("cannot enqueue branch deletion: %v", err)
				}
			}
			continue
		}
		payload := make(map[string]interface{})
		payload["git.author"] = event.Actor.DisplayName
		if len(pushChange.New.Target.Message) > 0 {
			payload["git.message"] = pushChange.New.Target.Message
		}

		if pushChange.New.Type == "branch" {
			branch := strings.TrimPrefix(pushChange.New.Name, "refs/heads/")
			payload["git.branch"] = branch
			if err := s.stopBranchDeletionTask(branch); err != nil {
				log.Error("cannot stop branch deletion task for branch %s : %v", branch, err)
			}

		} else if pushChange.New.Type == "tag" {
			payload["git.tag"] = strings.TrimPrefix(pushChange.New.Name, "refs/tags/")
		} else {
			log.Warning("Uknown push type: %s", pushChange.New.Type)
			continue
		}
		payload["git.hash.before"] = pushChange.Old.Target.Hash
		payload["git.hash"] = pushChange.New.Target.Hash
		hashShort := pushChange.New.Target.Hash
		if len(hashShort) >= 7 {
			hashShort = hashShort[:7]
		}
		payload["git.hash.short"] = hashShort
		payload["git.repository"] = event.Repository.FullName

		payload["cds.triggered_by.username"] = event.Actor.Username
		payload["cds.triggered_by.fullname"] = event.Actor.DisplayName
		payloadStr, err := json.Marshal(pushEvent)
		if err != nil {
			log.Error("Unable to marshal payload: %v", err)
		}
		payload["payload"] = string(payloadStr)
		payloads = append(payloads, payload)

	}
	return payloads, nil
}
