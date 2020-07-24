package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) ConfigUser() (sdk.ConfigUser, error) {
	var res sdk.ConfigUser
	if _, err := c.GetJSON(context.Background(), "/config/user", &res); err != nil {
		return res, err
	}
	return res, nil
}

func (c *client) ConfigCDN() (sdk.CDNConfig, error) {
	var res sdk.CDNConfig
	if _, err := c.GetJSON(context.Background(), "/config/cdn", &res); err != nil {
		return res, err
	}
	return res, nil
}
