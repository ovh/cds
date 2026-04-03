package forgejo

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (f *forgejoClient) GetHook(ctx context.Context, repo, url string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, sdk.WithStack(sdk.ErrNotImplemented)
}

func (f *forgejoClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}

func (f *forgejoClient) UpdateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}

func (f *forgejoClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}
