package hooks

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/service"
)

// Task execution status
const (
	TaskExecutionDoing     = "DOING"
	TaskExecutionDone      = "DONE"
	TaskExecutionScheduled = "SCHEDULED"
)

// Service is the stuct representing a hooks µService
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
	Cache  cache.Store
	Dao    dao
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name string `toml:"name" comment:"Name of this CDS Hooks Service"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1"`
		Port int    `toml:"port" default:"8083"`
	} `toml:"http" comment:"######################\n CDS Hooks HTTP Configuration \n######################"`
	URL              string                          `default:"http://localhost:8083"`
	URLPublic        string                          `toml:"urlPublic" comment:"Public url for external call (webhook)"`
	RetryDelay       int64                           `toml:"retryDelay" default:"120" comment:"Execution retry delay in seconds"`
	RetryError       int64                           `toml:"retryError" default:"3" comment:"Retry execution while this number of error is not reached"`
	ExecutionHistory int                             `toml:"executionHistory" default:"10" comment:"Number of execution to keep"`
	Disable          bool                            `toml:"disable" default:"false" comment:"Disable all hooks executions"`
	API              service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################"`
	Cache            struct {
		TTL   int `toml:"ttl" default:"60"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax! <clustername>@sentinel1:26379,sentinel2:26379,sentinel3:26379"`
			Password string `toml:"password"`
		} `toml:"redis" comment:"Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup"`
	} `toml:"cache" comment:"######################\n CDS Hooks Cache Settings \n######################\n"`
}
