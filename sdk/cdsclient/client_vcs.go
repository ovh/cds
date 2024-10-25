package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) VCSGerritConfiguration() (map[string]sdk.VCSGerritConfiguration, error) {
	var gerritConfiguration map[string]sdk.VCSGerritConfiguration
	if _, err := c.GetJSON(context.Background(), "/config/vcsgerrit", &gerritConfiguration); err != nil {
		return nil, err
	}
	return gerritConfiguration, nil
}
