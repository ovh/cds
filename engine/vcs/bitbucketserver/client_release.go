package bitbucketserver

import (
	"context"
	"io"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) Release(ctx context.Context, repo, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
func (b *bitbucketClient) UploadReleaseFile(ctx context.Context, repo string, releaseName string, uploadURL string, artifactName string, r io.Reader, fileLength int) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}
