package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) Download() ([]sdk.DownloadableResource, error) {
	var res []sdk.DownloadableResource
	if _, err := c.GetJSON(context.Background(), "/download", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *client) DownloadURLFromAPI(name, os, arch, variant string) string {
	return fmt.Sprintf("%s/download/%s/%s/%s?variant=%s", c.APIURL(), name, os, arch, variant)
}
