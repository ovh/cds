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
	m := sdk.MonitoringStatus{Now: t}

	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Version", Value: sdk.VERSION, Status: sdk.MonitoringStatusOK})
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Uptime", Value: fmt.Sprintf("%s", time.Since(c.StartupTime)), Status: sdk.MonitoringStatusOK})
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Time", Value: fmt.Sprintf("%dh%dm%ds", t.Hour(), t.Minute(), t.Second()), Status: sdk.MonitoringStatusOK})

	return m
}

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
	// no heartbeat for api
	if c.Type == "api" {
		return 0, nil
	}

	srv := sdk.Service{
		Name:             c.Name,
		HTTPURL:          c.HTTPURL,
		LastHeartbeat:    time.Time{},
		Token:            c.Token,
		Type:             c.Type,
		MonitoringStatus: status(),
	}
	hash, err := c.Client.ServiceRegister(srv)
	if err != nil {
		return 0, sdk.WrapError(err, "DoHeartbeat>")
	}
	c.Hash = hash
	return c.MaxHeartbeatFailures, nil
}
