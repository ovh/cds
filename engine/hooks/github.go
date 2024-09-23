package hooks

import (
	"context"
	"strings"

	"github.com/ovh/cds/sdk"
)

func (s *Service) generatePayloadFromGithubRequest(ctx context.Context, t *sdk.TaskExecution, event string) (map[string]interface{}, error) {
	var request GithubWebHookEvent
	if err := sdk.JSONUnmarshal(t.WebHook.RequestBody, &request); err != nil {
		return nil, sdk.WrapError(err, "unable ro read github request: %s", string(t.WebHook.RequestBody))
	}

	payload := make(map[string]interface{})
	payload[GIT_EVENT] = event

	if request.Ref != "" {
		if request.Deleted {
			return nil, nil
		}

		branch := strings.TrimPrefix(request.Ref, sdk.GitRefBranchPrefix)
		if !strings.HasPrefix(request.Ref, sdk.GitRefTagPrefix) {
			payload[GIT_BRANCH] = branch
		} else {
			payload[GIT_TAG] = strings.TrimPrefix(request.Ref, sdk.GitRefTagPrefix)
		}
	}
	if request.Before != "" {
		payload[GIT_HASH_BEFORE] = request.Before
	}
	if request.After != "" {
		payload[GIT_HASH] = request.After
		payload[GIT_HASH_SHORT] = sdk.StringFirstN(request.After, 7)
	}

	getPayloadFromRepository(payload, request.Repository)
	getPayloadFromCommit(payload, request.HeadCommit)

	if len(request.Commits) > 0 {
		payload[GIT_MESSAGE] = request.Commits[0].Message
	}

	for i := range request.Commits {
		request.Commits[i].Added = nil
		request.Commits[i].Removed = nil
		request.Commits[i].Modified = nil
	}
	getPayloadStringVariable(ctx, payload, request)

	return payload, nil
}

func getPayloadFromRepository(payload map[string]interface{}, repo *GithubRepository) {
	if repo == nil {
		return
	}
	payload[GIT_REPOSITORY] = repo.FullName
}

func getPayloadFromCommit(payload map[string]interface{}, commit *GithubCommit) {
	if commit == nil {
		return
	}
	payload[GIT_AUTHOR] = commit.Author.Username
	payload[GIT_AUTHOR_EMAIL] = commit.Author.Email
	payload[CDS_TRIGGERED_BY_USERNAME] = commit.Author.Username
	payload[CDS_TRIGGERED_BY_FULLNAME] = commit.Author.Name
	payload[CDS_TRIGGERED_BY_EMAIL] = commit.Author.Email
}
