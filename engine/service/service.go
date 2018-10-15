package service

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// APIServiceConfiguration is an exposed type for CDS API
type APIServiceConfiguration struct {
	HTTP struct {
		URL      string `toml:"url" default:"http://localhost:8081" json:"url"`
		Insecure bool   `toml:"insecure" commented:"true" json:"insecure"`
	} `toml:"http" json:"http"`
	GRPC struct {
		URL      string `toml:"url" default:"http://localhost:8082" json:"url"`
		Insecure bool   `toml:"insecure" commented:"true" json:"insecure"`
	} `toml:"grpc" json:"grpc"`
	Token                string `toml:"token" default:"************" json:"-"`
	RequestTimeout       int    `toml:"requestTimeout" default:"10" json:"requestTimeout"`
	MaxHeartbeatFailures int    `toml:"maxHeartbeatFailures" default:"10" json:"maxHeartbeatFailures"`
}

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
	ServiceName          string
}

// Service is the interface for a engine service
type Service interface {
	ApplyConfiguration(cfg interface{}) error
	Serve(ctx context.Context) error
	CheckConfiguration(cfg interface{}) error
	Heartbeat(ctx context.Context, status func() sdk.MonitoringStatus, cfg interface{}) error
	Register(status func() sdk.MonitoringStatus, cfg interface{}) error
	Status() sdk.MonitoringStatus
}

// BeforeStart has to be implemented if you want to run some code after the ApplyConfiguration and before the Serve of a Service
type BeforeStart interface {
	BeforeStart() error
}
