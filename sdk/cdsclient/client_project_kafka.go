package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectPlatform(projectKey string, platformName string, clearPassword bool) (sdk.ProjectPlatform, error) {
	var platform sdk.ProjectPlatform
	if _, err := c.GetJSON(context.Background(), fmt.Sprintf("/project/%s/platforms/%s?clearPassword=%t", projectKey, platformName, clearPassword), &platform); err != nil {
		return platform, err
	}
	return platform, nil
}
