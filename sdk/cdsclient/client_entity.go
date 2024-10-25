package cdsclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

// EntityGet retrieve an entity
func (c *client) EntityGet(ctx context.Context, projKey string, vcsIdentifier string, repoIdentifier string, entityType string, entityName string, mods ...RequestModifier) (*sdk.Entity, error) {
	var e sdk.Entity
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/entities/%s/%s", projKey, vcsIdentifier, url.PathEscape(repoIdentifier), entityType, entityName)
	if _, err := c.GetJSON(ctx, path, &e, mods...); err != nil {
		return nil, err
	}
	return &e, nil
}

func (c *client) EntityLint(ctx context.Context, entityType string, data interface{}) (*sdk.EntityCheckResponse, error) {
	path := fmt.Sprintf("/v2/entity/%s/check", entityType)
	var resp sdk.EntityCheckResponse
	if _, err := c.PostJSON(ctx, path, data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
