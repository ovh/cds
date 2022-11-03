package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) HatcheryAdd(ctx context.Context, h *sdk.Hatchery) error {
	if _, err := c.PostJSON(ctx, "/v2/hatchery", &h, &h); err != nil {
		return err
	}
	return nil
}

func (c *client) HatcheryGet(ctx context.Context, hatcheryIdentifier string) (sdk.Hatchery, error) {
	var reg sdk.Hatchery
	if _, err := c.GetJSON(ctx, "/v2/hatchery/"+hatcheryIdentifier, &reg, nil); err != nil {
		return reg, err
	}
	return reg, nil
}

func (c *client) HatcheryList(ctx context.Context) ([]sdk.Hatchery, error) {
	var hatcheries []sdk.Hatchery
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
