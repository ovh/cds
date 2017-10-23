package gitlab

import (
	"bytes"
	"fmt"

	"github.com/ovh/cds/sdk"
)

// Release on gitlab
// TODO: https://docs.gitlab.com/ee/api/tags.html#create-a-new-release
func (c *gitlabClient) Release(repo string, tagName string, title string, releaseNote string) (*sdk.VCSRelease, error) {
	return nil, fmt.Errorf("not implemented")
}

// UploadReleaseFile upload a release file project
func (c *gitlabClient) UploadReleaseFile(repo string, release *sdk.VCSRelease, runArtifact sdk.WorkflowNodeRunArtifact, buf *bytes.Buffer) error {
	return fmt.Errorf("not implemented")
}
