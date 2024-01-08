package gerrit

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk"
)

// GetEvents is not implemented
func (c *gerritClient) GetEvents(ctx context.Context, repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0.0, sdk.WithStack(sdk.ErrNotImplemented)
}

// PushEvents is not implemented
func (c *gerritClient) PushEvents(context.Context, string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

// CreateEvents is not implemented
func (c *gerritClient) CreateEvents(context.Context, string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

// DeleteEvents is not implemented
func (c *gerritClient) DeleteEvents(context.Context, string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

// PullRequestEvents is not implemented
func (c *gerritClient) PullRequestEvents(context.Context, string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
