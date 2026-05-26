package forgejo

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk"
)

func (f *forgejoClient) GetEvents(ctx context.Context, repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0, sdk.WithStack(sdk.ErrNotImplemented)
}
func (f *forgejoClient) PushEvents(context.Context, string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
func (f *forgejoClient) CreateEvents(context.Context, string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
func (f *forgejoClient) DeleteEvents(context.Context, string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
func (f *forgejoClient) PullRequestEvents(context.Context, string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
