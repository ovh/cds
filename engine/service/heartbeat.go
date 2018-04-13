package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (c *Common) Heartbeat(ctx context.Context) error {
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
			if maxHeartbeatFailures, err := c.DoHeartbeat(); err != nil {
				log.Error("VCS> heartbeat> Heartbeat failed")
				heartbeatFailures++
				if heartbeatFailures > maxHeartbeatFailures {
					return fmt.Errorf("Heartbeat failed excedeed")
				}
			}
			heartbeatFailures = 0
		}
	}
}

// DoHeartbeat registers the service for heartbeat
func (c *Common) DoHeartbeat() (int, error) {
	srv := sdk.Service{
		Name:          c.Name,
		HTTPURL:       c.HTTPURL,
		LastHeartbeat: time.Time{},
		Token:         c.Token,
		Type:          c.Type,
	}
	log.Debug("%s> doHeartbeat: %+v", c.Name, srv)
	hash, err := c.Client.ServiceRegister(srv)
	if err != nil {
		return 0, sdk.WrapError(err, "doHeartbeat")
	}
	c.Hash = hash
	return c.MaxHeartbeatFailures, nil
}
