package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (c *client) ServiceHeartbeat(s *sdk.MonitoringStatus) error {
	_, err := c.PostJSON(context.Background(), "/services/heartbeat", s, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) ServiceRegister(ctx context.Context, s sdk.Service) (*sdk.Service, error) {
	code, err := c.PostJSON(context.Background(), "/services/register", &s, &s)
	if code != 201 && code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}

	if !s.Uptodate {
		log.Warning(ctx, "-=-=-=-=- Please update your cds engine binary - current version:%s -=-=-=-=-", sdk.VersionString())
	}
	return &s, nil
}

func (c *client) ServiceConfigurationGet(ctx context.Context, t string) ([]sdk.ServiceConfiguration, error) {
	var servicesConf []sdk.ServiceConfiguration
	_, err := c.GetJSON(ctx, fmt.Sprintf("/services/%s", t), &servicesConf)
	if err != nil {
		return nil, err
	}
	return servicesConf, nil
}
