package gitlab

import (
	"context"
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
)

// Release on gitlab
// TODO: https://docs.gitlab.com/ee/api/tags.html#create-a-new-release
func (c *gitlabClient) Release(ctx context.Context, repo string, tagName string, title string, releaseNote string) (*sdk.VCSRelease, error) {
	return nil, fmt.Errorf("not implemented")
}

// UploadReleaseFile upload a release file project
func (c *gitlabClient) UploadReleaseFile(ctx context.Context, repo string, releaseName string, uploadURL string, artifactName string, r io.ReadCloser) error {
	return fmt.Errorf("not implemented")
}
