package bitbucketcloud

import (
	"context"

	"github.com/ovh/cds/sdk"
)

// GetAuthorized returns an authorized client
func (consumer *bitbucketcloudConsumer) GetAuthorizedClient(ctx context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	c := &bitbucketcloudClient{
		appPassword: vcsAuth.Token,
		username:    vcsAuth.Username,
		Cache:       consumer.Cache,
		apiURL:      consumer.apiURL,
		uiURL:       consumer.uiURL,
		proxyURL:    consumer.proxyURL,
	}
	return c, nil
}
