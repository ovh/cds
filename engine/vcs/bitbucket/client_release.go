package bitbucket

import (
	"bytes"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) Release(repo, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) UploadReleaseFile(repo string, release *sdk.VCSRelease, runArtifact sdk.WorkflowNodeRunArtifact, file *bytes.Buffer) error {
	return fmt.Errorf("Not yet implemented")
}
