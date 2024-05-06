package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectGetKey(ctx context.Context, projectKey, keyName string, clear bool) (*sdk.ProjectKey, error) {
	path := fmt.Sprintf("/v2/project/%s/key/%s?clearKey=%v", projectKey, keyName, clear)
	var pk sdk.ProjectKey
	if _, err := c.GetJSON(context.Background(), path, &pk); err != nil {
		return nil, err
	}
	return &pk, nil
}
