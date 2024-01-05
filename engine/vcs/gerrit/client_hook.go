package gerrit

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *gerritClient) GetHook(ctx context.Context, repo, id string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, sdk.WithStack(sdk.ErrNotImplemented)
}

// CreateHook enables the default HTTP POST Hook in Gerrit
func (c *gerritClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	if len(hook.Events) == 0 {
		hook.Events = sdk.GerritEventsDefault
	}
	return nil
}

// UpdateHook enables the default HTTP POST Hook in Gerrit
func (c *gerritClient) UpdateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	if len(hook.Events) == 0 {
		hook.Events = sdk.GerritEventsDefault
	}
	return nil
}

// DeleteHook disables the default HTTP POST Hook in Gerrit
func (c *gerritClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	return nil
}
