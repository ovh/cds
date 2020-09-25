package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) generatePayloadFromBitbucketServerRequest(ctx context.Context, t *sdk.TaskExecution, event string) ([]map[string]interface{}, error) {
	payloads := []map[string]interface{}{}

	var request sdk.BitbucketServerWebhookEvent
	if err := json.Unmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable ro read bitbucket request: %s", string(t.WebHook.RequestBody))
	}

	payload := make(map[string]interface{})

	payload[GIT_EVENT] = request.EventKey
	getVariableFromBitbucketServerAuthor(payload, request.Actor)
	getVariableFromBitbucketServerPullRequest(payload, request.PullRequest)
	getVariableFromBitbucketServerParticipant(payload, request.Participant)
	getPayloadStringVariable(ctx, payload, request)
	getPayloadFromBitbucketServerPRComment(payload, request.Comment)
	getPayloadFromBitbucketServerPreviousTarget(payload, request.PreviousTarget)
	getVariableFromBitbucketServerRepository(payload, request.Repository)

	if request.PreviousStatus != "" {
		payload[PR_PREVIOUS_STATE] = request.PreviousStatus
	}
	if request.PreviousComment != "" {
		payload[PR_COMMENT_TEXT_PREVIOUS] = request.PreviousComment
	}
	if request.PreviousTitle != "" {
		payload[PR_PREVIOUS_TITLE] = request.PreviousTitle
	}

	if len(request.Changes) == 0 {
		payloads = append(payloads, payload)
	}

	projectKey := t.Config["project"].Value
	workflowName := t.Config["workflow"].Value
	for _, pushChange := range request.Changes {
		if pushChange.Type == "DELETE" {
			err := s.enqueueBranchDeletion(projectKey, workflowName, strings.TrimPrefix(pushChange.RefID, "refs/heads/"))
			if err != nil {
				log.Error(ctx, "cannot enqueue branch deletion: %v", err)
			}
			continue
		}
		if !strings.HasPrefix(pushChange.RefID, "refs/tags/") {
			branch := strings.TrimPrefix(pushChange.RefID, "refs/heads/")
			if err := s.stopBranchDeletionTask(ctx, branch); err != nil {
				log.Error(ctx, "cannot stop branch deletion task for branch %s : %v", branch, err)
			}
		}

		payloadChanges := make(map[string]interface{})
		for k, v := range payload {
			payloadChanges[k] = v
		}
		getVariableFromBitbucketServerChange(payloadChanges, pushChange)
		payloads = append(payloads, payloadChanges)
	}

	return payloads, nil
}

func getVariableFromBitbucketServerChange(payload map[string]interface{}, change sdk.BitbucketServerChange) {
	if !strings.HasPrefix(change.RefID, "refs/tags/") {
		branch := strings.TrimPrefix(change.RefID, "refs/heads/")
		payload[GIT_BRANCH] = branch
	} else {
		payload[GIT_TAG] = strings.TrimPrefix(change.RefID, "refs/tags/")
	}
	payload[GIT_HASH_BEFORE] = change.FromHash
	payload[GIT_HASH] = change.ToHash
	payload[GIT_HASH_SHORT] = sdk.StringFirstN(change.ToHash, 7)
}

func getVariableFromBitbucketServerRepository(payload map[string]interface{}, repo *sdk.BitbucketServerRepository) {
	if repo == nil {
		return
	}
	payload[GIT_REPOSITORY_DEST] = fmt.Sprintf("%s/%s", repo.Project.Key, repo.Slug)
}

func getVariableFromBitbucketServerSrcRepository(payload map[string]interface{}, repo *sdk.BitbucketServerRepository) {
	if repo == nil {
		return
	}
	payload[GIT_REPOSITORY] = fmt.Sprintf("%s/%s", repo.Project.Key, repo.Slug)
}

func getVariableFromBitbucketServerAuthor(payload map[string]interface{}, actor *sdk.BitbucketServerActor) {
	if actor == nil {
		return
	}
	payload[GIT_AUTHOR] = actor.Name
	payload[GIT_AUTHOR_EMAIL] = actor.EmailAddress
	payload[CDS_TRIGGERED_BY_USERNAME] = actor.Name
	payload[CDS_TRIGGERED_BY_FULLNAME] = actor.DisplayName
	payload[CDS_TRIGGERED_BY_EMAIL] = actor.EmailAddress
}

func getVariableFromBitbucketServerPullRequest(payload map[string]interface{}, pr *sdk.BitbucketServerPullRequest) {
	if pr == nil {
		return
	}
	payload[PR_ID] = pr.ID
	payload[PR_STATE] = pr.State
	payload[PR_TITLE] = pr.Title
	payload[GIT_BRANCH] = pr.FromRef.DisplayID
	payload[GIT_HASH] = pr.FromRef.LatestCommit
	payload[GIT_BRANCH_DEST] = pr.ToRef.DisplayID
	payload[GIT_HASH_DEST] = pr.ToRef.LatestCommit
	payload[GIT_HASH_SHORT] = sdk.StringFirstN(pr.FromRef.LatestCommit, 7)

	getVariableFromBitbucketServerRepository(payload, &pr.ToRef.Repository)
	getVariableFromBitbucketServerSrcRepository(payload, &pr.FromRef.Repository)
}

func getPayloadFromBitbucketServerPRComment(payload map[string]interface{}, comment *sdk.BitbucketServerComment) {
	if comment == nil {
		return
	}
	payload[PR_COMMENT_AUTHOR] = comment.Author.Name
	payload[PR_COMMENT_AUTHOR_EMAIL] = comment.Author.EmailAddress
	payload[PR_COMMENT_TEXT] = comment.Text
}

func getPayloadFromBitbucketServerPreviousTarget(payload map[string]interface{}, target *sdk.BitbucketServerPreviousTarget) {
	if target == nil {
		return
	}
	payload[PR_PREVIOUS_BRANCH] = target.DisplayID
	payload[PR_PREVIOUS_HASH] = target.LatestCommit
}

func getVariableFromBitbucketServerParticipant(payload map[string]interface{}, participant *sdk.BitbucketServerParticipant) {
	if participant == nil {
		return
	}
	payload[PR_REVIEWER] = participant.User.Name
	payload[PR_REVIEWER_EMAIL] = participant.User.EmailAddress
	payload[PR_REVIEWER_STATUS] = participant.Status
	payload[PR_REVIEWER_ROLE] = participant.Role
}
