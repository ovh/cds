package gerrit

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *gerritClient) ListForks(ctx context.Context, repo string) ([]sdk.VCSRepo, error) {
	return nil, nil
}
