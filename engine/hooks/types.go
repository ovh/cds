package hooks

import (
	"reflect"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// Service is the stuct representing a hooks µService
type Service struct {
	Cfg    Configuration
	Router *api.Router
	Cache  cache.Store
	cds    cdsclient.Interface
	Dao    dao
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name string `toml:"name" default:"cdshooks" comment:"Name of this CDS Hooks Service"`
	HTTP struct {
		Port int `toml:"port" default:"8083" toml:"name"`
	} `toml:"http" comment:"######################\n# CDS Hooks HTTP Configuration #\n######################\n"`
	URL string `default:"http://localhost:8083"`
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
	} `toml:"api" comment:"######################\n# CDS API Settings #\n######################\n`
	Cache struct {
		Mode  string `toml:"mode" default:"local" comment:"Cache Mode: redis or local"`
		TTL   int    `toml:"ttl" default:"60"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379"`
			Password string `toml:"password"`
		} `toml:"redis" comment:"Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup"`
	} `toml:"cache" comment:"######################\n# CDS Cache Settings #\n######################\nIf your CDS is made of a unique instance, a local cache if enough, but rememeber that all cached data will be lost on startup."`
}

type LongRunningTask struct {
	UUID   string
	Type   string
	Config sdk.WorkflowNodeHookConfig
}

type LongRunningTaskExecution struct {
	UUID                string
	Config              sdk.WorkflowNodeHookConfig
	Type                string
	Timestamp           int64
	RequestURL          string
	RequestBody         []byte
	RequestHeader       map[string][]string
	LastError           string
	ProcessingTimestamp int64
	WorkflowRun         int64
}

type ScheduledTask struct {
	UUID      string
	Type      string
	CronExpr  string
	Config    sdk.WorkflowNodeHookConfig
	LastError string
}

type ScheduledTaskExecution struct {
	UUID                   string
	Type                   string
	DateScheduledExecution string
}

func interfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("interfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}
