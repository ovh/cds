package hooks

import (
	"reflect"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/sdk/cdsclient"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
)

// Service is the stuct representing a hooks µService
type Service struct {
	Cfg    Configuration
	Router *api.Router
	Cache  cache.Store
	cds    cdsclient.Interface
	Dao    dao
}

// Configuration is
type Configuration struct {
	Name string
	HTTP struct {
		Port int `default:"8083"`
	}
	URL string `default:"http://localhost:8083"`
	API struct {
		HTTP struct {
			URL      string `default:"http://localhost:8081"`
			Insecure bool
		}
		GRPC struct {
			URL      string `default:"http://localhost:8082"`
			Insecure bool
		}
		Token                string `default:"************"`
		RequestTimeout       int    `default:"10"`
		MaxHeartbeatFailures int    `default:"10"`
	}
	Cache struct {
		Mode  string `default:"redis"`
		TTL   int    `default:"60"`
		Redis struct {
			Host     string `default:"localhost:6379"`
			Password string
		}
	}
}

type LongRunningTask struct {
	UUID      string
	Type      string
	Config    sdk.WorkflowNodeHookConfig
	LastError string
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
	DateEffectiveExecution string
	Error                  string
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
