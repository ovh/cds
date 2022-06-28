package gitea

import (
	"context"
	"io"

	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) Release(ctx context.Context, repo, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
func (g *giteaClient) UploadReleaseFile(ctx context.Context, repo string, releaseName string, uploadURL string, artifactName string, r io.Reader, fileLength int) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}
