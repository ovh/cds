package cdsclient

import (
	"context"
	"fmt"
	"github.com/ovh/cds/sdk"
)

// EntityGet retrieve an entity
func (c *client) EntityGet(ctx context.Context, projKey string, vcsIdentifier string, repoIdentifier string, entityType string, entityName string) (*sdk.Entity, error) {
	var e sdk.Entity
	if _, err := c.GetJSON(ctx, fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/entities/%s/%s", projKey, vcsIdentifier, repoIdentifier, entityType, entityName), &e, nil); err != nil {
		return nil, err
	}
	return &e, nil
}
