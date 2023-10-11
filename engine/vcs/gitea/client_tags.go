package gitea

import (
	"context"
	"github.com/ovh/cds/sdk"
)

// Tags retrieve tags
func (g *giteaClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (c *giteaClient) Tag(ctx context.Context, fullname string, tagName string) (sdk.VCSTag, error) {
	return sdk.VCSTag{}, sdk.WithStack(sdk.ErrNotImplemented)
}
