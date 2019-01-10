package gerrit

import (
	"context"

	"github.com/ovh/cds/sdk"
)

//SetStatus set build status on Gitlab
func (c *gerritClient) SetStatus(ctx context.Context, event sdk.Event) error {
	return nil
}

func (c *gerritClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	return nil, nil
}
