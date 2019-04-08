package gerrit

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

//GetEvents is not implemented
func (c *gerritClient) GetEvents(ctx context.Context, repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0.0, fmt.Errorf("Not implemented on Gerrit")
}

//PushEvents is not implemented
func (c *gerritClient) PushEvents(context.Context, string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gerrit")
}

//CreateEvents is not implemented
func (c *gerritClient) CreateEvents(context.Context, string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gerrit")
}

//DeleteEvents is not implemented
func (c *gerritClient) DeleteEvents(context.Context, string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gerrit")
}

//PullRequestEvents is not implemented
func (c *gerritClient) PullRequestEvents(context.Context, string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gerrit")
}
