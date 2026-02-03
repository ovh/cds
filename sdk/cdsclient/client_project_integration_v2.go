package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectV2IntegrationWorkerHookGet(projectKey string, integrationName string) (*sdk.WorkerHookProjectIntegrationModel, error) {
	path := fmt.Sprintf("/v2/project/%s/integrations/%s/workerhooks", projectKey, integrationName)
	var res sdk.WorkerHookProjectIntegrationModel
	if _, err := c.GetJSON(context.Background(), path, &res); err != nil {
		return &res, err
	}
	return &res, nil
}
