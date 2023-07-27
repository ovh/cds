package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) HasProjectRole(ctx context.Context, projectKey, sessionID string, role string) error {
	req := sdk.CheckProjectAccess{
		ProjectKey: projectKey,
		Role:       role,
		SessionID:  sessionID,
	}
	if _, err := c.PostJSON(ctx, "/v2/rbac/access/project/session/check", req, nil); err != nil {
		return err
	}
	return nil
}
