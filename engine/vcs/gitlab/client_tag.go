package gitlab

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

//Tags retrieves the tags
func (c *gitlabClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	return nil, fmt.Errorf("not implemented")
}
