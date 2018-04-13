package service

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// Common is the struct representing a CDS µService
type Common struct {
	Client               cdsclient.Interface
	Hash                 string
	StartupTime          time.Time
	API                  string
	Name                 string
	HTTPURL              string
	Token                string
	Type                 string
	MaxHeartbeatFailures int
}

type Service interface {
	ApplyConfiguration(cfg interface{}) error
	Serve(ctx context.Context) error
	CheckConfiguration(cfg interface{}) error
	Heartbeat(ctx context.Context, status func() sdk.MonitoringStatus) error
	DoHeartbeat(status func() sdk.MonitoringStatus) (int, error)
	Status() sdk.MonitoringStatus
}
