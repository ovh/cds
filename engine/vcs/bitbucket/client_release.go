package bitbucket

import (
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) Release(repo, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) UploadReleaseFile(repo string, releaseName string, uploadURL string, artifactName string, r io.ReadCloser) error {
	return fmt.Errorf("Not yet implemented")
}
