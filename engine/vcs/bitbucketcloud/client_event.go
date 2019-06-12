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
	return nil, 0, sdk.WithStack(sdk.ErrNotImplemented)
}

//PushEvents returns push events as commits
func (client *bitbucketcloudClient) PushEvents(ctx context.Context, fullname string, iEvents []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

//CreateEvents checks create events from a event list
func (client *bitbucketcloudClient) CreateEvents(ctx context.Context, fullname string, iEvents []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

//DeleteEvents checks delete events from a event list
func (client *bitbucketcloudClient) DeleteEvents(ctx context.Context, fullname string, iEvents []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

//PullRequestEvents checks pull request events from a event list
func (client *bitbucketcloudClient) PullRequestEvents(ctx context.Context, fullname string, iEvents []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
