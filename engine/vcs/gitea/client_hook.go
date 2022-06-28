package gitea

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) GetHook(ctx context.Context, repo, url string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) UpdateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	return sdk.WithStack(sdk.ErrNotImplemented)
}
