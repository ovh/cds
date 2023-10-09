package cdsclient

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) PollVCSEvents(uuid string, workflowID int64, vcsServer string, timestamp int64) (events sdk.RepositoryEvents, interval time.Duration, err error) {
	url := fmt.Sprintf("/hook/%s/workflow/%d/vcsevent/%s", uuid, workflowID, vcsServer)
	header, _, errGet := c.GetJSONWithHeaders(url, &events, SetHeader("X-CDS-Last-Execution", fmt.Sprint(timestamp)))
	if errGet != nil {
		return events, interval, errGet
	}

	//Check poll interval
	if header.Get("X-CDS-Poll-Interval") != "" {
		f, errParse := strconv.ParseFloat(header.Get("X-CDS-Poll-Interval"), 64)
		if errParse == nil {
			interval = time.Duration(f) * time.Second
		}
	}

	return events, interval, nil
}

func (c *client) HookRepositoriesList(ctx context.Context, vcsServer, repoName string) ([]sdk.ProjectRepository, error) {
	path := fmt.Sprintf("/v2/hooks/repositories/%s/%s", vcsServer, url.PathEscape(repoName))
	var repos []sdk.ProjectRepository
	_, err := c.GetJSON(ctx, path, &repos)
	return repos, err
}

func (c *client) ListWorkflowToTrigger(ctx context.Context, req sdk.HookListWorkflowRequest) ([]sdk.V2WorkflowHook, error) {
	var workflowHooks []sdk.V2WorkflowHook
	_, err := c.PostJSON(ctx, "/v2/hooks/workflows", &req, &workflowHooks)
	return workflowHooks, err
}

func (c *client) RetrieveHookEventSigningKey(ctx context.Context, req sdk.HookRetrieveSignKeyRequest) (sdk.Operation, error) {
	var ope sdk.Operation
	_, err := c.PostJSON(ctx, "/v2/hooks/event/signKey", &req, &ope)
	return ope, err
}
func (c *client) RetrieveHookEventUser(ctx context.Context, req sdk.HookRetrieveUserRequest) (sdk.HookRetrieveUserResponse, error) {
	var resp sdk.HookRetrieveUserResponse
	_, err := c.PostJSON(ctx, "/v2/hooks/event/user", &req, &resp)
	return resp, err
}

func (c *client) RetrieveHookEventSigningKeyOperation(ctx context.Context, operationUUID string) (sdk.Operation, error) {
	var ope sdk.Operation
	_, err := c.GetJSON(ctx, "/v2/hooks/event/signKey/"+operationUUID, &ope)
	return ope, err
}
