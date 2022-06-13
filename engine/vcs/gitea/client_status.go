package gitea

import (
	"context"
	"github.com/ovh/cds/sdk"
)

// DEPRECATED VCS
func (g *giteaClient) IsDisableStatusDetails(ctx context.Context) bool {
	return false
}

func (g *giteaClient) SetStatus(ctx context.Context, event sdk.Event, disableStatusDetails bool) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
