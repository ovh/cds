package bitbucket

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

// Tags retrieve tags
func (b *bitbucketClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	return nil, fmt.Errorf("not implemented")
}
