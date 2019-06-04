package bitbucketcloud

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

// ErrNoNewEvents for no new events
var (
	ErrNoNewEvents = fmt.Errorf("No new events")
)

//GetEvents returns events from bitbucket cloud
func (client *bitbucketcloudClient) GetEvents(ctx context.Context, fullname string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0, fmt.Errorf("Not yet implemented")
}

//PushEvents returns push events as commits
func (client *bitbucketcloudClient) PushEvents(ctx context.Context, fullname string, iEvents []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}

//CreateEvents checks create events from a event list
func (client *bitbucketcloudClient) CreateEvents(ctx context.Context, fullname string, iEvents []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}

//DeleteEvents checks delete events from a event list
func (client *bitbucketcloudClient) DeleteEvents(ctx context.Context, fullname string, iEvents []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}

//PullRequestEvents checks pull request events from a event list
func (client *bitbucketcloudClient) PullRequestEvents(ctx context.Context, fullname string, iEvents []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, fmt.Errorf("Not yet implemented")
}
