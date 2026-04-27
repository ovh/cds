package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) VCSGPGKey(ctx context.Context, gpgKeyID string) ([]sdk.VCSUserGPGKey, error) {
	var results []sdk.VCSUserGPGKey
	if _, err := c.GetJSON(context.Background(), "/v2/vcs/gpgkeys/"+url.QueryEscape(gpgKeyID), &results); err != nil {
		return nil, err
	}
	return results, nil
}
