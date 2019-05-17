package bitbucket

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) GetEvents(ctx context.Context, repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) PushEvents(context.Context, string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) CreateEvents(context.Context, string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) DeleteEvents(context.Context, string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
func (b *bitbucketClient) PullRequestEvents(context.Context, string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
