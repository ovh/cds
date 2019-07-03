package service

import (
	"context"
	"crypto/rsa"
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
	Token                string `toml:"token" default:"************" json:"-"`
	RequestTimeout       int    `toml:"requestTimeout" default:"10" json:"requestTimeout"`
	MaxHeartbeatFailures int    `toml:"maxHeartbeatFailures" default:"10" json:"maxHeartbeatFailures"`
}

// Common is the struct representing a CDS µService
type Common struct {
	Client               cdsclient.Interface
	APIPublicKey         []byte
	ParsedAPIPublicKey   *rsa.PublicKey
	StartupTime          time.Time
	Name                 string
	HTTPURL              string
	Type                 string
	MaxHeartbeatFailures int
	ServiceName          string
	ServiceInstance      *sdk.Service
}

// Service is the interface for a engine service
type Service interface {
	ApplyConfiguration(cfg interface{}) error
	Serve(ctx context.Context) error
	CheckConfiguration(cfg interface{}) error
	Start(ctx context.Context, cfg cdsclient.ServiceConfig) error
	Init(cfg interface{}) (cdsclient.ServiceConfig, error)
	Register(ctx context.Context, cfg sdk.ServiceConfig) error
	Heartbeat(ctx context.Context, status func() sdk.MonitoringStatus) error
	Status() sdk.MonitoringStatus
}

// BeforeStart has to be implemented if you want to run some code after the ApplyConfiguration and before the Serve of a Service
type BeforeStart interface {
	BeforeStart() error
}
