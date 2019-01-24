package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectIntegration(projectKey string, integrationName string, clearPassword bool) (sdk.ProjectIntegration, error) {
	var integration sdk.ProjectIntegration
	if _, err := c.GetJSON(context.Background(), fmt.Sprintf("/project/%s/integrations/%s?clearPassword=%t", projectKey, integrationName, clearPassword), &integration); err != nil {
		return integration, err
	}
	return integration, nil
}
