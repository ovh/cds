package hooks

import (
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"strings"
)

func (s *Service) generatePayloadFromBitbucketServerRequest(t *sdk.TaskExecution, event string) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	switch event {
	case "repo:refs_changed":
		return s.generatePayloadFromBitbucketServerPushEvent(t)
	case "pr:opened":
		return s.generatePayloadFromBitbucketServerPROpened(t)
	case "pr:modified":
		return s.generatePayloadFromBitbucketServerPRModified(t)
	case "pr:declined":
		return s.generatePayloadFromBitbucketServerPRDeclined(t)
	case "pr:deleted":
		return s.generatePayloadFromBitbucketServerPRDeleted(t)
	case "pr:merged":
		return s.generatePayloadFromBitbucketServerPRMerged(t)
	case "pr:comment:added":
		return s.generatePayloadFromBitbucketServerPRCommentAdded(t)
	case "pr:comment:edited":
		return s.generatePayloadFromBitbucketServerPRCommentEdited(t)
	case "pr:comment:deleted":
		return s.generatePayloadFromBitbucketServerPRCommentDeleted(t)
	case "pr:reviewer:approved":
		return s.generatePayloadFromBitbucketServerPRReviewerApproved(t)
	case "pr:reviewer:updated":
		return s.generatePayloadFromBitbucketServerPRReviewerUpdated(t)
	case "pr:reviewer:unapproved":
		return s.generatePayloadFromBitbucketServerPRReviewerUnapproved(t)
	case "pr:reviewer:needs_work":
		return s.generatePayloadFromBitbucketServerPRReviewerNeedsWork(t)
	default:
		payload, err := getAllPayloadMap(t.WebHook.RequestBody)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPushEvent(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	projectKey := t.Config["project"].Value
	workflowName := t.Config["workflow"].Value
	var pushEvent sdk.BitbucketServerPushEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &pushEvent); err != nil {
		return nil, sdk.WrapError(err, "unable ro read bitbucket request: %s", string(t.WebHook.RequestBody))
	}

	for _, pushChange := range pushEvent.Changes {
		if pushChange.Type == "DELETE" {
			err := s.enqueueBranchDeletion(projectKey, workflowName, strings.TrimPrefix(pushChange.RefID, "refs/heads/"))
			if err != nil {
				log.Error("cannot enqueue branch deletion: %v", err)
			}
			continue
		}

		if !strings.HasPrefix(pushChange.RefID, "refs/tags/") {
			branch := strings.TrimPrefix(pushChange.RefID, "refs/heads/")
			if err := s.stopBranchDeletionTask(branch); err != nil {
				log.Error("cannot stop branch deletion task for branch %s : %v", branch, err)
			}
		}

		payload := make(map[string]interface{})
		payload[GIT_EVENT] = pushEvent.EventKey
		getVariableFromAuthor(payload, pushEvent.Actor)
		getVariableFromChange(payload, pushChange)
		getVariableFromRepository(payload, pushEvent.Repository)
		getPayloadStringVariable(payload, pushEvent)

		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRReviewerUpdated(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRReviewerUpdated
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRReviewerUpdated: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getPayloadStringVariable(payload, request)
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRReviewerApproved(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRReviewerApproved
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRReviewerApproved: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getVariableFromParticipant(payload, request.Participant)
	getPayloadStringVariable(payload, request)
	payload[PR_PREVIOUS_STATE] = request.PreviousStatus
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRReviewerUnapproved(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRReviewerApproved
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRReviewerApproved: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getVariableFromParticipant(payload, request.Participant)
	getPayloadStringVariable(payload, request)
	payload[PR_PREVIOUS_STATE] = request.PreviousStatus
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRReviewerNeedsWork(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRReviewerApproved
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRReviewerApproved: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getVariableFromParticipant(payload, request.Participant)
	getPayloadStringVariable(payload, request)
	payload[PR_PREVIOUS_STATE] = request.PreviousStatus
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPROpened(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPROpenedEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPROpenedEvent: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getPayloadStringVariable(payload, request)
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRModified(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRModifiedEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRModifiedEvent: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getPayloadStringVariable(payload, request)
	payload[PR_PREVIOUS_TITLE] = request.PreviousTitle
	payload[PR_PREVIOUS_BRANCH] = request.PreviousTarget.DisplayID
	payload[PR_PREVIOUS_HASH] = request.PreviousTarget.LatestCommit
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRDeclined(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRDeclinedEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRDeclinedEvent: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getPayloadStringVariable(payload, request)
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRDeleted(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRDeletedEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRDeletedEvent: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getPayloadStringVariable(payload, request)
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRMerged(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRMergedEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRMergedEvent: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getPayloadStringVariable(payload, request)
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRCommentAdded(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRCommentAddedEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRCommentAddedEvent: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getPayloadFromPRComment(payload, request.Comment)
	getPayloadStringVariable(payload, request)
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRCommentEdited(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRCommentEditedEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRCommentEditedEvent: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getPayloadFromPRComment(payload, request.Comment)
	getPayloadStringVariable(payload, request)
	payload[PR_COMMENT_TEXT_PREVIOUS] = request.PreviousComment
	payloads = append(payloads, payload)
	return payloads, nil
}

func (s *Service) generatePayloadFromBitbucketServerPRCommentDeleted(t *sdk.TaskExecution) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}
	var request sdk.BitbucketServerPRCommentDeletedEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal into BitbucketServerPRCommentDeletedEvent: %s", string(t.WebHook.RequestBody))
	}
	payload := make(map[string]interface{})
	payload[GIT_EVENT] = request.EventKey
	getVariableFromAuthor(payload, request.Actor)
	getVariableFromPullRequest(payload, request.PullRequest)
	getPayloadFromPRComment(payload, request.Comment)
	getPayloadStringVariable(payload, request)
	payloads = append(payloads, payload)
	return payloads, nil
}

func getVariableFromChange(payload map[string]interface{}, change sdk.BitbucketServerChange) {
	if !strings.HasPrefix(change.RefID, "refs/tags/") {
		branch := strings.TrimPrefix(change.RefID, "refs/heads/")
		payload[GIT_BRANCH] = branch
	} else {
		payload[GIT_TAG] = strings.TrimPrefix(change.RefID, "refs/tags/")
	}
	payload[GIT_HASH_BEFORE] = change.FromHash
	payload[GIT_HASH] = change.ToHash
	hashShort := change.ToHash
	if len(hashShort) >= 7 {
		hashShort = hashShort[:7]
	}
	payload[GIT_HASH_SHORT] = hashShort
}

func getPayloadStringVariable(payload map[string]interface{}, msg interface{}) {
	payloadStr, err := json.Marshal(msg)
	if err != nil {
		log.Error("Unable to marshal payload: %v", err)
	}
	payload[PAYLOAD] = string(payloadStr)
}

func getVariableFromRepository(payload map[string]interface{}, repo sdk.BitbucketServerRepository) {
	payload[GIT_REPOSITORY] = fmt.Sprintf("%s/%s", repo.Project.Key, repo.Slug)
}

func getVariableFromSrcRepository(payload map[string]interface{}, repo sdk.BitbucketServerRepository) {
	payload[GIT_REPOSITORY_BEFORE] = fmt.Sprintf("%s/%s", repo.Project.Key, repo.Slug)
}

func getVariableFromAuthor(payload map[string]interface{}, actor sdk.BitbucketServerActor) {
	payload[GIT_AUTHOR] = actor.Name
	payload[GIT_AUTHOR_EMAIL] = actor.EmailAddress
	payload[CDS_TRIGGERED_BY_USERNAME] = actor.Name
	payload[CDS_TRIGGERED_BY_FULLNAME] = actor.DisplayName
	payload[CDS_TRIGGERED_BY_EMAIL] = actor.EmailAddress
}

func getVariableFromPullRequest(payload map[string]interface{}, pr sdk.BitbucketServerPullRequest) {
	payload[PR_ID] = pr.ID
	payload[PR_STATE] = pr.State
	payload[PR_TITLE] = pr.Title
	payload[GIT_BRANCH_BEFORE] = pr.FromRef.DisplayID
	payload[GIT_HASH_BEFORE] = pr.FromRef.LatestCommit
	payload[GIT_BRANCH] = pr.ToRef.DisplayID
	payload[GIT_HASH] = pr.ToRef.LatestCommit
	hashShort := pr.ToRef.LatestCommit
	if len(hashShort) >= 7 {
		hashShort = hashShort[:7]
	}
	payload[GIT_HASH_SHORT] = hashShort

	getVariableFromRepository(payload, pr.ToRef.Repository)
	getVariableFromSrcRepository(payload, pr.FromRef.Repository)
}

func getPayloadFromPRComment(payload map[string]interface{}, comment sdk.BitbucketServerComment) {
	payload[PR_COMMENT_AUTHOR] = comment.Author.Name
	payload[PR_COMMENT_AUTHOR_EMAIL] = comment.Author.EmailAddress
	payload[PR_COMMENT_TEXT] = comment.Text
}

func getVariableFromParticipant(payload map[string]interface{}, participant sdk.BitbucketServerParticipant) {
	payload[PR_REVIEWER] = participant.User.Name
	payload[PR_REVIEWER_EMAIL] = participant.User.EmailAddress
	payload[PR_REVIEWER_STATUS] = participant.Status
	payload[PR_REVIEWER_ROLE] = participant.Role
}
