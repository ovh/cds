package service

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"

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

type HTTPRouterConfiguration struct {
	Addr                string `toml:"addr" default:"" commented:"true" comment:"Listen HTTP address without port, example: 127.0.0.1" json:"addr"`
	Port                int    `toml:"port" default:"8081" json:"port"`
	HeaderXForwardedFor string `toml:"headerXForwardedFor" commented:"true" comment:"Forward source addr from given header, let empty to use request addr." default:"X-Forwarded-For" json:"header_w_forwarded_for"`
}

// HatcheryCommonConfiguration is the base configuration for all hatcheries
type HatcheryCommonConfiguration struct {
	Name          string                  `toml:"name" default:"" comment:"Name of Hatchery" json:"name"`
	RSAPrivateKey string                  `toml:"rsaPrivateKey" default:"" comment:"The RSA Private Key used by the hatchery.\nThis is mandatory." json:"-"`
	HTTP          HTTPRouterConfiguration `toml:"http" comment:"######################\n CDS Hatchery HTTP Configuration \n######################" json:"http"`
	URL           string                  `toml:"url" default:"http://localhost:8086" comment:"URL of this Hatchery" json:"url"`
	API           struct {
		HTTP struct {
			URL      string `toml:"url" default:"http://localhost:8081" comment:"CDS API URL" json:"url"`
			Insecure bool   `toml:"insecure" default:"false" commented:"true" comment:"sslInsecureSkipVerify, set to true if you use a self-signed SSL on CDS API" json:"insecure"`
		} `toml:"http" json:"http"`
		Token                string `toml:"token" default:"" comment:"CDS Token to reach CDS API. See https://ovh.github.io/cds/docs/components/cdsctl/token/ " json:"-"`
		TokenV2              string `toml:"tokenV2" default:"" comment:"Hatchery consumer Token. Allow to reach CDS API on /v2 routes" json:"-"`
		RequestTimeout       int    `toml:"requestTimeout" default:"10" comment:"Request CDS API: timeout in seconds" json:"requestTimeout"`
		MaxHeartbeatFailures int    `toml:"maxHeartbeatFailures" default:"10" comment:"Maximum allowed consecutives failures on heatbeat routine" json:"maxHeartbeatFailures"`
	} `toml:"api" json:"api"`
	CDN struct {
		URL string `toml:"url" default:"http://localhost:8089" commented:"true" comment:"Address to access CDN HTTP server, let empty or commented to use the public URL that is returned by the CDS API." json:"url"`
		TCP struct {
			EnableTLS bool   `toml:"enableTLS" commented:"true" comment:"Enable TLS for CDN TCP connection" json:"enable_tls"`
			URL       string `toml:"url" default:"localhost:8090" commented:"true" comment:"Address to access CDN TCP server, let empty or commented to use the public URL that is returned by the CDS API." json:"url"`
		} `toml:"tcp" json:"tcp"`
	} `toml:"cdn" json:"cdn"`
	Provision struct {
		InjectEnvVars             []string `toml:"injectEnvVars" commented:"true" comment:"Inject env variables in workers" json:"-" mapstructure:"injectEnvVars"`
		MaxWorker                 int      `toml:"maxWorker" default:"10" comment:"Maximum allowed simultaneous workers" json:"maxWorker"`
		MaxConcurrentProvisioning int      `toml:"maxConcurrentProvisioning" default:"10" comment:"Maximum allowed simultaneous workers provisioning" json:"maxConcurrentProvisioning"`
		MaxConcurrentRegistering  int      `toml:"maxConcurrentRegistering" default:"2" comment:"Maximum allowed simultaneous workers registering. -1 to disable registering on this hatchery" json:"maxConcurrentRegistering"`
		RegisterFrequency         int      `toml:"registerFrequency" default:"60" comment:"Check if some worker model have to be registered each n Seconds" json:"registerFrequency"`
		Region                    string   `toml:"region" default:"" comment:"region of this hatchery - optional. With a free text as 'myregion', user can set a prerequisite 'region' with value 'myregion' on CDS Job" json:"region"`
		IgnoreJobWithNoRegion     bool     `toml:"ignoreJobWithNoRegion" default:"false" comment:"Ignore job without a region prerequisite if ignoreJobWithNoRegion=true" json:"ignoreJobWithNoRegion"`
		WorkerAPIHTTP             struct {
			URL      string `toml:"url" default:"" commented:"false" comment:"CDS API URL for worker, let empty or commented to use the same URL that is used by the Hatchery. Example: http://localhost:8081" json:"url"`
			Insecure bool   `toml:"insecure" default:"false" commented:"true" comment:"sslInsecureSkipVerify, set to true if you use a self-signed SSL on CDS API" json:"insecure"`
		} `toml:"workerApiHttp" json:"workerApiHttp"`
		WorkerCDN struct {
			URL string `toml:"url" default:"" commented:"true" comment:"Address to access CDN HTTP server for worker, let empty or commented to use the same URL that is used by the Hatchery." json:"url"`
			TCP struct {
				EnableTLS bool   `toml:"enableTLS" commented:"true" comment:"Enable TLS for CDN TCP connection" json:"enable_tls"`
				URL       string `toml:"url" default:"" commented:"true" comment:"Address to access CDN TCP server, let empty or commented to use the same URL that is used by the Hatchery." json:"url"`
			} `toml:"tcp" json:"tcp"`
		} `toml:"workerCdn" json:"workerCdn"`
		WorkerBasedir     string `toml:"workerBasedir" commented:"true" comment:"Worker Basedir" json:"workerBasedir"`
		WorkerLogsOptions struct {
			Level   string `toml:"level" comment:"Worker log level" json:"level"`
			Graylog struct {
				Host       string `toml:"host" comment:"Example: thot.ovh.com" json:"host"`
				Port       int    `toml:"port" comment:"Example: 12202" json:"port"`
				Protocol   string `toml:"protocol" default:"tcp" comment:"tcp or udp" json:"protocol"`
				ExtraKey   string `toml:"extraKey" comment:"Example: X-OVH-TOKEN. You can use many keys: aaa,bbb" json:"extraKey"`
				ExtraValue string `toml:"extraValue" comment:"value for extraKey field. For many keys: valueaaa,valuebbb" json:"-"`
			} `toml:"graylog" json:"graylog"`
		} `toml:"workerLogsOptions" comment:"Worker Log Configuration" json:"workerLogsOptions"`
		MaxAttemptsNumberBeforeFailure int `toml:"maxAttemptsNumberBeforeFailure" default:"5" commented:"true" comment:"Maximum attempts to start a same job. -1 to disable failing jobs when to many attempts" json:"maxAttemptsNumberBeforeFailure"`
	} `toml:"provision" json:"provision"`
	LogOptions struct {
		SpawnOptions struct {
			ThresholdCritical int `toml:"thresholdCritical" default:"480" comment:"log critical if spawn take more than this value (in seconds)" json:"thresholdCritical"`
			ThresholdWarning  int `toml:"thresholdWarning" default:"360" comment:"log warning if spawn take more than this value (in seconds)" json:"thresholdWarning"`
		} `toml:"spawnOptions" json:"spawnOptions"`
	} `toml:"logOptions" comment:"Hatchery Log Configuration" json:"logOptions"`
}

func (hcc HatcheryCommonConfiguration) Check() error {
	if hcc.Provision.MaxConcurrentProvisioning > hcc.Provision.MaxWorker {
		return fmt.Errorf("maxConcurrentProvisioning (value: %d) cannot be less than maxWorker (value: %d) ",
			hcc.Provision.MaxConcurrentProvisioning, hcc.Provision.MaxWorker)
	}

	if hcc.Provision.MaxConcurrentRegistering > hcc.Provision.MaxWorker {
		return fmt.Errorf("maxConcurrentRegistering (value: %d) cannot be less than maxWorker (value: %d) ",
			hcc.Provision.MaxConcurrentRegistering, hcc.Provision.MaxWorker)
	}

	if hcc.API.HTTP.URL == "" {
		return fmt.Errorf("API HTTP(s) URL is mandatory")
	}

	if hcc.API.Token == "" {
		return fmt.Errorf("API Token URL is mandatory")
	}

	if hcc.Name == "" {
		return fmt.Errorf("please enter a name in your hatchery configuration")
	}

	return nil
}

// Common is the struct representing a CDS µService
type Common struct {
	Client                cdsclient.Interface
	APIPublicKey          []byte
	ParsedAPIPublicKey    *rsa.PublicKey
	StartupTime           time.Time
	HTTPURL               string
	MaxHeartbeatFailures  int
	ServiceName           string
	ServiceType           string
	ServiceInstance       *sdk.Service
	PrivateKey            *rsa.PrivateKey
	Signer                jose.Signer
	CDNConfig             sdk.CDNConfig
	ServiceLogger         *logrus.Logger
	GoRoutines            *sdk.GoRoutines
	Region                string
	IgnoreJobWithNoRegion bool
	ModelType             string
}

// Service is the interface for a engine service
// Lifecycle: ApplyConfiguration->?BeforeStart->Init->Signin->Register->Start->Serve->Heartbeat
type Service interface {
	ApplyConfiguration(cfg interface{}) error
	Serve(ctx context.Context) error
	CheckConfiguration(cfg interface{}) error
	Start(ctx context.Context) error
	Init(cfg interface{}) (cdsclient.ServiceConfig, error)
	Signin(ctx context.Context, clientConfig cdsclient.ServiceConfig, srvConfig interface{}) error
	Unregister(ctx context.Context) error
	Heartbeat(ctx context.Context, status func(ctx context.Context) *sdk.MonitoringStatus) error
	Status(ctx context.Context) *sdk.MonitoringStatus
	NamedService
}

// BeforeStart has to be implemented if you want to run some code after the ApplyConfiguration and before the Serve of a Service
type BeforeStart interface {
	BeforeStart(ctx context.Context) error
}
