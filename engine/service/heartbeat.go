package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Heartbeat have to be launch as a goroutine, call DoHeartBeat each 30s
func (c *Common) Heartbeat(ctx context.Context, status func() sdk.MonitoringStatus) error {
	ticker := time.NewTicker(30 * time.Second)
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	var heartbeatFailures int
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if maxHeartbeatFailures, err := c.DoHeartbeat(status); err != nil {
				log.Error("%s> Heartbeat> Heartbeat failed", c.Name)
				heartbeatFailures++
				if heartbeatFailures > maxHeartbeatFailures {
					return fmt.Errorf("%s> Heartbeat> failed excedeed", c.Name)
				}
			}
			heartbeatFailures = 0
		}
	}
}

// DoHeartbeat registers the service for heartbeat
func (c *Common) DoHeartbeat(status func() sdk.MonitoringStatus) (int, error) {
	srv := sdk.Service{
		Name:             c.Name,
		HTTPURL:          c.HTTPURL,
		LastHeartbeat:    time.Time{},
		Token:            c.Token,
		Type:             c.Type,
		MonitoringStatus: status(),
	}
	log.Debug("%s> DoHeartbeat> %+v", c.Name, srv)
	hash, err := c.Client.ServiceRegister(srv)
	if err != nil {
		return 0, sdk.WrapError(err, "DoHeartbeat>")
	}
	c.Hash = hash
	return c.MaxHeartbeatFailures, nil
}
