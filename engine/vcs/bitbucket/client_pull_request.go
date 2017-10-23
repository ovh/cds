package bitbucket

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) PullRequests(repo string) ([]sdk.VCSPullRequest, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
