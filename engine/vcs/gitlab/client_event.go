package gitlab

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

//GetEvents is not implemented
func (c *gitlabClient) GetEvents(ctx context.Context, repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0.0, fmt.Errorf("Not implemented on Gitlab")
}

//PushEvents is not implemented
func (c *gitlabClient) PushEvents(context.Context, string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

//CreateEvents is not implemented
func (c *gitlabClient) CreateEvents(context.Context, string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

//DeleteEvents is not implemented
func (c *gitlabClient) DeleteEvents(context.Context, string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

//PullRequestEvents is not implemented
func (c *gitlabClient) PullRequestEvents(context.Context, string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}
