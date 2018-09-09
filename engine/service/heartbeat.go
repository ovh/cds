package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// CommonMonitoring returns common part of MonitoringStatus
func (c *Common) CommonMonitoring() sdk.MonitoringStatus {
	t := time.Now()
	return sdk.MonitoringStatus{
		Now: t,
		Lines: []sdk.MonitoringStatusLine{{
			Component: "Version",
			Value:     sdk.VERSION,
			Status:    sdk.MonitoringStatusOK,
		}, {
			Component: "Uptime",
			Value:     fmt.Sprintf("%s", time.Since(c.StartupTime)),
			Status:    sdk.MonitoringStatusOK,
		}, {
			Component: "Time",
			Value:     fmt.Sprintf("%dh%dm%ds", t.Hour(), t.Minute(), t.Second()),
			Status:    sdk.MonitoringStatusOK,
		}},
	}
}

// Heartbeat have to be launch as a goroutine, call DoHeartBeat each 30s
func (c *Common) Heartbeat(ctx context.Context, status func() sdk.MonitoringStatus) error {
	// no heartbeat for api
	if c.Type == "api" {
		return nil
	}

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
			// try to register, on success reset the failure count
			if err := c.Register(status); err != nil {
				heartbeatFailures++
				log.Error("%s> Heartbeat> Register failed %d/%d", c.Name,
					heartbeatFailures, c.MaxHeartbeatFailures)
			} else {
				heartbeatFailures = 0
			}

			// if register failed too many time, stop heartbeat
			if heartbeatFailures > c.MaxHeartbeatFailures {
				return fmt.Errorf("%s> Heartbeat> Register failed excedeed", c.Name)
			}
		}
	}
}

// Register the service to CDS api and store session hash.
func (c *Common) Register(status func() sdk.MonitoringStatus) error {
	// no need to register for api
	if c.Type == "api" {
		return nil
	}

	hash, err := c.Client.ServiceRegister(sdk.Service{
		Name:             c.Name,
		HTTPURL:          c.HTTPURL,
		LastHeartbeat:    time.Time{},
		Token:            c.Token,
		Type:             c.Type,
		MonitoringStatus: status(),
	})
	if err != nil {
		return sdk.WrapError(err, "Register>")
	}

	c.Hash = hash

	return nil
}
