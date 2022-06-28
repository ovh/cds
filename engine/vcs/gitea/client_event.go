package gitea

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) GetEvents(ctx context.Context, repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0, sdk.WithStack(sdk.ErrNotImplemented)
}
func (g *giteaClient) PushEvents(context.Context, string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
func (g *giteaClient) CreateEvents(context.Context, string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
func (g *giteaClient) DeleteEvents(context.Context, string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
func (g *giteaClient) PullRequestEvents(context.Context, string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}
