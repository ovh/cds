package cdsclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *client) FeatureEnabled(name sdk.FeatureName, params map[string]string) (sdk.FeatureEnabledResponse, error) {
	var response sdk.FeatureEnabledResponse
	code, err := c.PostJSON(context.Background(), fmt.Sprintf("/feature/enabled/%s", name), params, &response)
	if code != http.StatusOK {
		if err == nil {
			return response, newAPIError(fmt.Errorf("HTTP Code %d", code))
		}
	}
	return response, err
}
