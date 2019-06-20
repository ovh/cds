package bitbucketcloud

import (
	"context"
	"io"

	"github.com/ovh/cds/sdk"
)

// Release Create a release
func (client *bitbucketcloudClient) Release(ctx context.Context, fullname string, tagName string, title string, releaseNote string) (*sdk.VCSRelease, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

// UploadReleaseFile Attach a file into the release
func (client *bitbucketcloudClient) UploadReleaseFile(ctx context.Context, repo string, releaseName string, uploadURL string, artifactName string, r io.ReadCloser) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}
