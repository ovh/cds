package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectV2List(ctx context.Context) ([]sdk.Project, error) {
	p := []sdk.Project{}
	path := "/v2/project"

	if _, err := c.GetJSON(context.Background(), path, &p); err != nil {
		return nil, err
	}
	return p, nil
}
