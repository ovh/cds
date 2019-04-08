package gerrit

import (
	"context"

	"github.com/ovh/cds/sdk"
)

//Tags retrieves the tags
func (c *gerritClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	return nil, nil
}
