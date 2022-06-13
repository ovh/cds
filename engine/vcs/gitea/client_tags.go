package gitea

import (
	"context"
	"github.com/ovh/cds/sdk"
)

// Tags retrieve tags
func (g *giteaClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
