package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *hatcheryClient) Heartbeat(ctx context.Context, mon *sdk.MonitoringStatus) error {
	if _, err := c.PostJSON(ctx, "/v2/hatchery/heartbeat", &mon, nil); err != nil {
		return err
	}
	return nil
}
