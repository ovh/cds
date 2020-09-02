package cdsclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *client) FeatureEnabled(name string, params map[string]string) (sdk.FeatureEnabledResponse, error) {
	var response sdk.FeatureEnabledResponse
	code, err := c.PostJSON(context.Background(), "/feature/enabled/"+name, params, &response)
	if code != http.StatusOK {
		if err == nil {
			return response, fmt.Errorf("HTTP Code %d", code)
		}
	}
	return response, err
}
