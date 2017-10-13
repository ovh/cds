package bitbucket

import (
	"bytes"
	"time"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) Repos() ([]sdk.VCSRepo, error) {
	return nil, nil
}
func (b *bitbucketClient) RepoByFullname(fullname string) (sdk.VCSRepo, error) {
	return sdk.VCSRepo{}, nil
}
func (b *bitbucketClient) Branches(string) ([]sdk.VCSBranch, error) {
	return nil, nil
}
func (b *bitbucketClient) Branch(string, string) (*sdk.VCSBranch, error) {
	return nil, nil
}
func (b *bitbucketClient) Commits(repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	return nil, nil
}
func (b *bitbucketClient) Commit(repo, hash string) (sdk.VCSCommit, error) {
	return sdk.VCSCommit{}, nil
}
func (b *bitbucketClient) PullRequests(string) ([]sdk.VCSPullRequest, error) {
	return nil, nil
}
func (b *bitbucketClient) CreateHook(repo, url string) error {
	return nil
}
func (b *bitbucketClient) DeleteHook(repo, url string) error {
	return nil
}
func (b *bitbucketClient) GetEvents(repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0, nil
}
func (b *bitbucketClient) PushEvents(string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, nil
}
func (b *bitbucketClient) CreateEvents(string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, nil
}
func (b *bitbucketClient) DeleteEvents(string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, nil
}
func (b *bitbucketClient) PullRequestEvents(string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, nil
}
func (b *bitbucketClient) SetStatus(event sdk.Event) error {
	return nil
}
func (b *bitbucketClient) Release(repo, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error) {
	return nil, nil
}
func (b *bitbucketClient) UploadReleaseFile(repo string, release *sdk.VCSRelease, runArtifact sdk.WorkflowNodeRunArtifact, file *bytes.Buffer) error {
	return nil
}
