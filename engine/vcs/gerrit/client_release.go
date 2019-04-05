package gerrit

import (
	"context"
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
)

// Release on gerrit
func (c *gerritClient) Release(ctx context.Context, repo string, tagName string, title string, releaseNote string) (*sdk.VCSRelease, error) {
	return nil, fmt.Errorf("not implemented")
}

// UploadReleaseFile upload a release file project
func (c *gerritClient) UploadReleaseFile(ctx context.Context, repo string, releaseName string, uploadURL string, artifactName string, r io.ReadCloser) error {
	return fmt.Errorf("not implemented")
}
