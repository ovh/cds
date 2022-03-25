package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ServiceHeartbeat(s *sdk.MonitoringStatus) error {
	_, err := c.PostJSON(context.Background(), "/services/heartbeat", s, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) ServiceConfigurationGet(ctx context.Context, t string) ([]sdk.ServiceConfiguration, error) {
	var servicesConf []sdk.ServiceConfiguration
	_, err := c.GetJSON(ctx, fmt.Sprintf("/services/%s", t), &servicesConf)
	if err != nil {
		return nil, err
	}
	return servicesConf, nil
}
