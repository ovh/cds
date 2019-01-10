package gerrit

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *gerritClient) GetHook(ctx context.Context, repo, id string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, fmt.Errorf("Not yet implemented")
}
func (c *gerritClient) UpdateHook(ctx context.Context, repo, id string, hook sdk.VCSHook) error {
	return fmt.Errorf("Not yet implemented")
}

//CreateHook enables the defaut HTTP POST Hook in Gitlab
func (c *gerritClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	return nil
}

//DeleteHook disables the defaut HTTP POST Hook in Gitlab
func (c *gerritClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	return nil
}
