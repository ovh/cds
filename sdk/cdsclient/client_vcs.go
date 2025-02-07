package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) VCSGerritConfiguration() (map[string]sdk.VCSGerritConfiguration, error) {
	var gerritConfiguration map[string]sdk.VCSGerritConfiguration
	if _, err := c.GetJSON(context.Background(), "/config/vcsgerrit", &gerritConfiguration); err != nil {
		return nil, err
	}
	return gerritConfiguration, nil
}

func (c *client) VCSGPGKey(ctx context.Context, gpgKeyID string) ([]sdk.VCSUserGPGKey, error) {
	var results []sdk.VCSUserGPGKey
	if _, err := c.GetJSON(context.Background(), "/v2/vcs/gpgkeys/"+url.QueryEscape(gpgKeyID), &results); err != nil {
		return nil, err
	}
	return results, nil
}
