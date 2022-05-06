package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

// VCSConfiguration get the vcs servers configuration
func (c *client) VCSConfiguration() (map[string]sdk.VCSConfiguration, error) {
	var vcsServers map[string]sdk.VCSConfiguration
	if _, err := c.GetJSON(context.Background(), "/config/vcs", &vcsServers); err != nil {
		return nil, err
	}
	return vcsServers, nil
}

func (c *client) VCSGerritConfiguration() (map[string]sdk.VCSGerritConfiguration, error) {
	var gerritConfiguration map[string]sdk.VCSGerritConfiguration
	if _, err := c.GetJSON(context.Background(), "/config/vcsgerrit", &gerritConfiguration); err != nil {
		return nil, err
	}
	return gerritConfiguration, nil
}
