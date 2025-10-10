package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) V2ConfigCDN() (sdk.CDNConfig, error) {
	var res sdk.CDNConfig
	if _, err := c.GetJSON(context.Background(), "/v2/config/cdn", &res); err != nil {
		return res, err
	}
	return res, nil
}
