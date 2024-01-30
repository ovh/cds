package gitea

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (client *giteaClient) SetDisableStatusDetails(disableStatusDetails bool) {
	// no implementation for gitea
}

func (client *giteaClient) SetStatus(ctx context.Context, event sdk.Event) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}

func (client *giteaClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
