package bitbucket

import (
	"context"
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) Release(ctx context.Context, repo, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) UploadReleaseFile(ctx context.Context, repo string, releaseName string, uploadURL string, artifactName string, r io.ReadCloser) error {
	return fmt.Errorf("Not yet implemented")
}
