package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/telemetry"
)

// GetCommon returns a pointer to the Common struct.
// This is automatically available on all services that embed Common.
func (c *Common) GetCommon() *Common {
	return c
}

// ListenAndServeOrWait starts the HTTP server, or if the service is in gateway
// mode, simply waits for ctx cancellation without starting a listener.
func ListenAndServeOrWait(ctx context.Context, c *Common, server *http.Server) error {
	if c.GatewayServiceMode {
		log.Info(ctx, "%s> Gateway mode: skipping HTTP listener on %s", c.ServiceType, server.Addr)
		<-ctx.Done()
		return ctx.Err()
	}
	return server.ListenAndServe()
}

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
	if c.ServiceType == "hatchery" && cdsclientConfig.Token == "" {
		log.Info(ctx, "No token v1 provided. The hatchery will not handle job v1")
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
	if c.IgnoreJobWithNoRegion {
		registerPayload.CanonicalService.IgnoreJobWithNoRegion = &c.IgnoreJobWithNoRegion
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

// LocalSigninRegisterFunc is the function signature for registering a local service.
// It is provided by the API when services are co-located in the same process.
type LocalSigninRegisterFunc func(ctx context.Context, srv sdk.Service) error

// LocalSignin registers a co-located service directly with the API in-process,
// without HTTP calls or authentication tokens. The apiHandler is the API's
// http.Handler used for in-process communication. The registerFunc creates
// the consumer and service entries in the database.
func (c *Common) LocalSignin(ctx context.Context, apiHandler http.Handler, registerFunc LocalSigninRegisterFunc, srvConfig interface{}) error {
	if c.ServiceType == "api" {
		return nil
	}

	log.Info(ctx, "LocalSignin> registering local service %s(%s)", c.Name(), c.Type())

	serviceConfig, err := ParseServiceConfig(srvConfig)
	if err != nil {
		return err
	}

	// Build service payload
	var pubKey []byte
	if c.PrivateKey != nil {
		pubKey, err = jws.ExportPublicKey(c.PrivateKey)
		if err != nil {
			return sdk.WithStack(err)
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
	if c.IgnoreJobWithNoRegion {
		registerPayload.CanonicalService.IgnoreJobWithNoRegion = &c.IgnoreJobWithNoRegion
	}

	// Register directly in DB via the API
	err = registerFunc(ctx, registerPayload)
	if err != nil {
		return sdk.WrapError(err, "LocalSignin> cannot register service %s", c.Name())
	}

	// Create the local cdsclient that calls the API handler in-process
	c.Client = cdsclient.NewLocalServiceClient(apiHandler, c.Name(), c.Type())

	log.Info(ctx, "LocalSignin> local service %s registered", c.Name())
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

	log.Info(ctx, "Unregistering service %s(%T) %s", c.Type(), c, c.Name())
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
				return sdk.WithStack(fmt.Errorf("%s> Heartbeat failed exceeded", c.Name()))
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
