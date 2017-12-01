package bitbucket

import (
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) GetEvents(repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) PushEvents(string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) CreateEvents(string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) DeleteEvents(string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) PullRequestEvents(string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
