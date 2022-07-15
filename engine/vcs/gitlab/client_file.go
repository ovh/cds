package gitlab

import (
	"context"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *gitlabClient) GetArchive(ctx context.Context, repo string, dir string, format string, commit string) (io.Reader, http.Header, error) {
	return nil, nil, sdk.WithStack(sdk.ErrNotImplemented)
}
