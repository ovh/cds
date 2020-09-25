package hooks

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) generatePayloadFromBitbucketCloudRequest(ctx context.Context, t *sdk.TaskExecution, event string) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}

	var request BitbucketCloudWebHookEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable ro read bitbucket request: %s", string(t.WebHook.RequestBody))
	}

	projectKey := t.Config["project"].Value
	workflowName := t.Config["workflow"].Value

	payload := make(map[string]interface{})
	payload[GIT_EVENT] = event
	getVariableFromBitbucketCloudAuthor(payload, request.Actor)
	getVariableFromBitbucketCloudRepository(payload, request.Repository)
	getPayloadStringVariable(ctx, payload, request)

	for _, pushChange := range request.Push.Changes {
		if pushChange.Closed {
			if pushChange.Old.Type == "branch" {
				if err := s.enqueueBranchDeletion(projectKey, workflowName, strings.TrimPrefix(pushChange.Old.Name, "refs/heads/")); err != nil {
					log.Error(ctx, "cannot enqueue branch deletion: %v", err)
				}
			}
			continue
		}

		if pushChange.New.Type == "branch" {
			branch := strings.TrimPrefix(pushChange.New.Name, "refs/heads/")
			if err := s.stopBranchDeletionTask(ctx, branch); err != nil {
				log.Error(ctx, "cannot stop branch deletion task for branch %s : %v", branch, err)
			}

		}

		payloadChange := make(map[string]interface{})
		for k, v := range payload {
			payloadChange[k] = v
		}

		getVariableFromBitbucketCloudChange(ctx, payloadChange, pushChange)
		payloads = append(payloads, payloadChange)
	}

	return payloads, nil
}

func getVariableFromBitbucketCloudChange(ctx context.Context, payload map[string]interface{}, change BitbucketCloudChange) {
	if change.New.Type == "branch" {
		branch := strings.TrimPrefix(change.New.Name, "refs/heads/")
		payload[GIT_BRANCH] = branch
	} else if change.New.Type == "tag" {
		payload[GIT_TAG] = strings.TrimPrefix(change.New.Name, "refs/tags/")
	} else {
		log.Warning(ctx, "unknown push type: %s", change.New.Type)
		return
	}
	payload[GIT_HASH_BEFORE] = change.Old.Target.Hash
	payload[GIT_HASH] = change.New.Target.Hash
	payload[GIT_HASH_SHORT] = sdk.StringFirstN(change.New.Target.Hash, 7)
}

func getVariableFromBitbucketCloudRepository(payload map[string]interface{}, repo *BitbucketCloudRepository) {
	if repo == nil {
		return
	}
	payload[GIT_REPOSITORY] = repo.FullName
}

func getVariableFromBitbucketCloudAuthor(payload map[string]interface{}, actor *BitbucketCloudActor) {
	if actor == nil {
		return
	}
	payload[GIT_AUTHOR] = actor.DisplayName
	payload[CDS_TRIGGERED_BY_USERNAME] = actor.Username
	payload[CDS_TRIGGERED_BY_FULLNAME] = actor.DisplayName
}
