package vcs

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/cdsclient"
)

// Service is the stuct representing a vcs µService
type Service struct {
	Cfg    Configuration
	Router *api.Router
	Cache  cache.Store
	cds    cdsclient.Interface
	hash   string
}

// Configuration is the vcs configuration structure
type Configuration struct {
	Name string `toml:"name" comment:"Name of this CDS VCS Service"`
	HTTP struct {
		Port int `toml:"port" default:"8083" toml:"name"`
	} `toml:"http" comment:"######################\n CDS VCS HTTP Configuration \n######################"`
	URL string `default:"http://localhost:8084"`
	API struct {
		HTTP struct {
			URL      string `toml:"url" default:"http://localhost:8081"`
			Insecure bool   `toml:"insecure" commented:"true"`
		} `toml:"http"`
		GRPC struct {
			URL      string `toml:"url" default:"http://localhost:8082"`
			Insecure bool   `toml:"insecure" commented:"true"`
		} `toml:"grpc"`
		Token                string `toml:"token" default:"************"`
		RequestTimeout       int    `toml:"requestTimeout" default:"10"`
		MaxHeartbeatFailures int    `toml:"maxHeartbeatFailures" default:"10"`
	} `toml:"api" comment:"######################\n CDS API Settings \n######################`
	Cache struct {
		TTL   int `toml:"ttl" default:"60"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379"`
			Password string `toml:"password"`
		} `toml:"cache" comment:"######################\n CDS VCS Cache Settings \n######################"`
	}
}
