package github

import (
	"context"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (g *githubClient) ListContent(_ context.Context, repo string, commit, dir string) ([]sdk.VCSContent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *githubClient) GetContent(ctx context.Context, repo string, commit, filePath string) (sdk.VCSContent, error) {
	return sdk.VCSContent{}, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *githubClient) GetArchive(ctx context.Context, repo, dir, format, commit string) (io.Reader, http.Header, error) {
	return nil, nil, sdk.WithStack(sdk.ErrNotImplemented)
}
