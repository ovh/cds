package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/observability"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
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
			Value:     time.Since(c.StartupTime).String(),
			Status:    sdk.MonitoringStatusOK,
		}, {
			Component: "Time",
			Value:     fmt.Sprintf("%dh%dm%ds", t.Hour(), t.Minute(), t.Second()),
			Status:    sdk.MonitoringStatusOK,
		}},
	}
}

func (c *Common) Type() string {
	return c.ServiceType
}

func (c *Common) Name() string {
	return c.ServiceName
}

func (c *Common) Start(ctx context.Context, cfg cdsclient.ServiceConfig) error {
	// no register for api
	if c.ServiceType == "api" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var err error
	var firstAttempt = true
loop:
	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			return err
		default:
			c.Client, c.APIPublicKey, err = cdsclient.NewServiceClient(cfg)
			if err == nil {
				fmt.Println()
				break loop
			}
			if firstAttempt {
				fmt.Print("Waiting for CDS API..")
				firstAttempt = false
			}
			fmt.Print(".")
			time.Sleep(10 * time.Second)
		}
	}

	c.ParsedAPIPublicKey, err = jws.NewPublicKeyFromPEM(c.APIPublicKey)
	if err != nil {
		return sdk.WithStack(err)
	}

	ctx = observability.ContextWithTag(ctx,
		observability.TagServiceType, c.Type(),
		observability.TagServiceName, c.Name(),
	)

	RegisterCommonMetricsView(ctx)

	return nil
}

func (c *Common) Register(ctx context.Context, cfg sdk.ServiceConfig) error {
	log.Info("Registing service %s(%T) %s", c.Type(), c, c.Name())

	// no register for api
	if c.ServiceType == "api" {
		return nil
	}

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:    c.ServiceName,
			HTTPURL: c.HTTPURL,
			Type:    c.ServiceType,
			Config:  cfg,
		},
		LastHeartbeat: time.Time{},
		Version:       sdk.VERSION,
	}

	if c.PrivateKey != nil {
		pubKeyPEM, err := jws.ExportPublicKey(c.PrivateKey)
		if err != nil {
			return fmt.Errorf("unable get public key from private key: %v", err)
		}
		srv.PublicKey = pubKeyPEM
	}

	srv2, err := c.Client.ServiceRegister(ctx, srv)
	if err != nil {
		return sdk.WrapError(err, "Register>")
	}
	c.ServiceInstance = srv2
	return nil
}

// Heartbeat have to be launch as a goroutine, call DoHeartBeat each 30s
func (c *Common) Heartbeat(ctx context.Context, status func(ctx context.Context) sdk.MonitoringStatus) error {
	// no heartbeat for api
	if c.ServiceType == "api" {
		return nil
	}

	ticker := time.NewTicker(30 * time.Second)

	var heartbeatFailures int
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.Client.ServiceHeartbeat(status(ctx)); err != nil {
				log.Warning(ctx, "%s> Heartbeat failure: %v", c.Name(), err)
				heartbeatFailures++
			}

			// if register failed too many time, stop heartbeat
			if heartbeatFailures > c.MaxHeartbeatFailures {
				return fmt.Errorf("%s> Heartbeat> Register failed excedeed", c.Name())
			}
		}
	}
}
