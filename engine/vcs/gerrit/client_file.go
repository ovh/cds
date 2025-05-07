package gerrit

import (
	"context"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *gerritClient) ListContent(_ context.Context, repo string, commit, dir string, offset, limit string) ([]sdk.VCSContent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *gerritClient) GetContent(ctx context.Context, repo string, commit, file string) (sdk.VCSContent, error) {
	return sdk.VCSContent{}, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *gerritClient) GetArchive(ctx context.Context, repo string, dir string, format string, commit string) (io.Reader, http.Header, error) {
	return nil, nil, sdk.WithStack(sdk.ErrNotImplemented)
}
