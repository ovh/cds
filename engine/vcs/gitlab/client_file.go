package gitlab

import (
	"context"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *gitlabClient) ListContent(_ context.Context, repo string, commit, dir string) ([]sdk.VCSContent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (c *gitlabClient) GetContent(ctx context.Context, repo string, commit, filePath string) (sdk.VCSContent, error) {
	return sdk.VCSContent{}, sdk.WithStack(sdk.ErrNotImplemented)
}
func (c *gitlabClient) GetArchive(ctx context.Context, repo string, dir string, format string, commit string) (io.Reader, http.Header, error) {
	return nil, nil, sdk.WithStack(sdk.ErrNotImplemented)
}
