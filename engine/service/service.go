package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"

	"github.com/ovh/cds/sdk/cdsclient"

	"github.com/ovh/cds/sdk"
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

func (c *Common) Start(ctx context.Context, cfg cdsclient.ServiceConfig) error {
	// no register for api
	if c.Type == "api" {
		return nil
	}

	var err error
	c.Client, c.APIPublicKey, err = cdsclient.NewServiceClient(cfg)
	if err != nil {
		return sdk.WithStack(err)
	}
	c.ParsedAPIPublicKey, err = jws.NewPublicKeyFromPEM(c.APIPublicKey)
	if err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func (c *Common) Register(ctx context.Context, cfg sdk.ServiceConfig) error {
	// no register for api
	if c.Type == "api" {
		return nil
	}

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:    c.Name,
			HTTPURL: c.HTTPURL,
			Type:    c.Type,
			Config:  cfg,
		},
		LastHeartbeat: time.Time{},
		Version:       sdk.VERSION,
	}

	srv2, err := c.Client.ServiceRegister(srv)
	if err != nil {
		return sdk.WrapError(err, "Register>")
	}
	c.ServiceInstance = srv2
	return nil
}

// Heartbeat have to be launch as a goroutine, call DoHeartBeat each 30s
func (c *Common) Heartbeat(ctx context.Context, status func() sdk.MonitoringStatus) error {
	// no heartbeat for api
	if c.Type == "api" {
		return nil
	}

	ticker := time.NewTicker(30 * time.Second)

	var heartbeatFailures int
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.Client.ServiceHeartbeat(status()); err != nil {
				log.Warning("%s> Heartbeat failure: %v", c.Name, err)
				heartbeatFailures++
			}

			// if register failed too many time, stop heartbeat
			if heartbeatFailures > c.MaxHeartbeatFailures {
				return fmt.Errorf("%s> Heartbeat> Register failed excedeed", c.Name)
			}
		}
	}

}
