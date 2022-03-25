package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/telemetry"
)

// NewMonitoringStatus returns a MonitoringStatus for the current service
func (c *Common) NewMonitoringStatus() *sdk.MonitoringStatus {
	t := time.Now()
	hostname, err := os.Hostname()
	if err != nil {
		log.Error(context.Background(), "NewMonitoringStatus: error on getting hostname")
	}

	s := &sdk.MonitoringStatus{
		Now:             t,
		ServiceType:     c.Type(),
		ServiceName:     c.Name(),
		ServiceHostname: hostname,
	}
	s.AddLine(c.commonMonitoring(t)...)
	return s
}

// CommonMonitoring returns common monitoring status lines
func (c *Common) commonMonitoring(t time.Time) []sdk.MonitoringStatusLine {
	lines := []sdk.MonitoringStatusLine{{
		Component: "Version",
		Value:     sdk.VERSION,
		Status:    sdk.MonitoringStatusOK,
	}, {
		Component: "Uptime",
		Value:     time.Since(c.StartupTime).String(),
		Status:    sdk.MonitoringStatusOK,
	}, {
		Component: "Time",
		Value:     t.Format(time.RFC3339),
		Status:    sdk.MonitoringStatusOK,
	}}

	return append(lines, c.GoRoutines.GetStatus()...)
}

func (c *Common) Type() string {
	return c.ServiceType
}

func (c *Common) Name() string {
	return c.ServiceName
}

func (c *Common) Start(ctx context.Context) error {
	if c.ServiceType == "api" {
		return nil
	}

	ctx = telemetry.ContextWithTag(ctx,
		telemetry.TagServiceType, c.Type(),
		telemetry.TagServiceName, c.Name(),
	)
	c.RegisterCommonMetricsView(ctx)

	return nil
}

// Signin a new service on API
func (c *Common) Signin(ctx context.Context, cdsclientConfig cdsclient.ServiceConfig, srvConfig interface{}) error {
	if c.ServiceType == "api" {
		return nil
	}
	log.Info(ctx, "Init CDS client for service %s(%T) %s", c.Type(), c, c.Name())
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	serviceConfig, err := ParseServiceConfig(srvConfig)
	if err != nil {
		return err
	}

	var pubKey []byte
	if c.PrivateKey != nil {
		pubKey, err = jws.ExportPublicKey(c.PrivateKey)
		if err != nil {
			return err
		}
	}

	registerPayload := sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:      c.Name(),
			Type:      c.Type(),
			HTTPURL:   c.HTTPURL,
			Config:    serviceConfig,
			PublicKey: pubKey,
		},
	}

	if c.Region != "" {
		registerPayload.CanonicalService.Region = &c.Region
	}

	initClient := func(ctx context.Context) error {
		var err error
		// The call below should return the sdk.Service from the signin
		c.Client, c.ServiceInstance, c.APIPublicKey, err = cdsclient.NewServiceClient(ctx, cdsclientConfig, registerPayload)
		if err != nil {
			fmt.Printf("Waiting for CDS API (%v)...\n", err)
		}
		return err
	}

	var lasterr error
	if err := initClient(ctxTimeout); err != nil {
		lasterr = err
	loop:
		for {
			select {
			case <-ctxTimeout.Done():
				if lasterr != nil {
					fmt.Printf("Timeout after 5min - last error: %v\n", lasterr)
				}
				return ctxTimeout.Err()
			case <-ticker.C:
				if err := initClient(ctxTimeout); err == nil {
					lasterr = err //lint:ignore SA4006 false positive
					break loop
				}
			}
		}
	}

	c.ParsedAPIPublicKey, err = jws.NewPublicKeyFromPEM(c.APIPublicKey)
	if err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// ParseServiceConfig parse any object to craft a sdk.ServiceConfig
func ParseServiceConfig(cfg interface{}) (sdk.ServiceConfig, error) {
	var sdkConfig sdk.ServiceConfig
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if err := sdk.JSONUnmarshal(b, &sdkConfig); err != nil {
		return nil, sdk.WithStack(err)
	}
	return sdkConfig, nil
}

// Unregister logout the service
func (c *Common) Unregister(ctx context.Context) error {
	// no logout needed for api
	if c.ServiceType == "api" {
		return nil
	}

	// check if client not nil, can happen when service is waiting for api
	if c.Client == nil {
		return nil
	}

	log.Info(ctx, "Unregisting service %s(%T) %s", c.Type(), c, c.Name())
	return c.Client.AuthConsumerSignout()
}

// Heartbeat have to be launch as a goroutine, call DoHeartBeat each 30s
func (c *Common) Heartbeat(ctx context.Context, status func(ctx context.Context) *sdk.MonitoringStatus) error {
	// no heartbeat for api
	if c.ServiceType == "api" {
		return nil
	}

	var heartbeatFailures int
	execHeartbeat := func(ctx context.Context) error {
		if err := c.Client.ServiceHeartbeat(status(ctx)); err != nil {
			if sdk.ErrorIs(err, sdk.ErrForbidden) {
				return sdk.WrapError(err, "%s> Heartbeat failed with forbidden error", c.Name())
			}
			heartbeatFailures++
			log.Warn(ctx, "%s> Heartbeat failure %d/%d: %v", c.Name(), heartbeatFailures, c.MaxHeartbeatFailures, err)

			// if register failed too many time, stop heartbeat
			if heartbeatFailures > c.MaxHeartbeatFailures {
				return sdk.WithStack(fmt.Errorf("%s> Heartbeat failed excedeed", c.Name()))
			}
			return nil
		}
		heartbeatFailures = 0
		return nil
	}

	// exec first heartbeat immediately
	if err := execHeartbeat(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return sdk.WrapError(ctx.Err(), "%s> Heartbeat> Cancelled", c.Name())
		case <-ticker.C:
			if err := execHeartbeat(ctx); err != nil {
				return err
			}
		}
	}
}
