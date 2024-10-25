package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) HatcheryRegenToken(ctx context.Context, hatcheryIdentifier string) (*sdk.HatcheryGetResponse, error) {
	path := fmt.Sprintf("/v2/hatchery/%s/regen", hatcheryIdentifier)
	var h sdk.HatcheryGetResponse
	if _, err := c.PostJSON(ctx, path, nil, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

func (c *client) HatcheryAdd(ctx context.Context, h *sdk.Hatchery) (*sdk.HatcheryGetResponse, error) {
	var hgr sdk.HatcheryGetResponse
	if _, err := c.PostJSON(ctx, "/v2/hatchery", &h, &hgr); err != nil {
		return nil, err
	}
	return &hgr, nil
}

func (c *client) HatcheryGet(ctx context.Context, hatcheryIdentifier string) (sdk.HatcheryGetResponse, error) {
	var reg sdk.HatcheryGetResponse
	if _, err := c.GetJSON(ctx, "/v2/hatchery/"+hatcheryIdentifier, &reg, nil); err != nil {
		return reg, err
	}
	return reg, nil
}

func (c *client) HatcheryList(ctx context.Context) ([]sdk.HatcheryGetResponse, error) {
	var hatcheries []sdk.HatcheryGetResponse
	if _, err := c.GetJSON(ctx, "/v2/hatchery", &hatcheries, nil); err != nil {
		return nil, err
	}
	return hatcheries, nil
}

func (c *client) HatcheryDelete(ctx context.Context, hatcheryIdentifier string) error {
	if _, err := c.DeleteJSON(ctx, "/v2/hatchery/"+hatcheryIdentifier, nil, nil); err != nil {
		return err
	}
	return nil
}
