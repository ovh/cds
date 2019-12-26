package gerrit

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *gerritClient) GetHook(ctx context.Context, repo, id string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, fmt.Errorf("Not implemented")
}

//CreateHook enables the default HTTP POST Hook in Gerrit
func (c *gerritClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	return nil
}

//UpdateHook enables the default HTTP POST Hook in Gerrit
func (c *gerritClient) UpdateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	return nil
}

//DeleteHook disables the default HTTP POST Hook in Gerrit
func (c *gerritClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	return nil
}
