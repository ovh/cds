package bitbucketcloud

import (
	"context"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (g *bitbucketcloudClient) GetArchive(ctx context.Context, repo string, dir string, format string, commit string) (io.Reader, http.Header, error) {
	return nil, nil, sdk.WithStack(sdk.ErrNotImplemented)
}
