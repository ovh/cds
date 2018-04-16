package vcs

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk/log"
)

var (
	RedisHost     string
	RedisPassword string
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

func newTestService(t *testing.T) (*Service, error) {
	//Read the test config file
	if RedisHost == "" {
		cfg := test.LoadTestingConf(t)
		RedisHost = cfg["redisHost"]
		RedisPassword = cfg["redisPassword"]
	}
	log.SetLogger(t)

	//Prepare the configuration
	cfg := Configuration{}
	cfg.Cache.TTL = 30
	cfg.Cache.Redis.Host = RedisHost
	cfg.Cache.Redis.Password = RedisPassword

	ctx := context.Background()
	r := &api.Router{
		Mux:        mux.NewRouter(),
		Prefix:     "/" + test.GetTestName(t),
		Background: ctx,
	}
	service := new(Service)
	service.Router = r
	service.initRouter(ctx)
	service.Cfg = cfg

	//Init the cache
	var errCache error
	service.Cache, errCache = cache.New(service.Cfg.Cache.Redis.Host, service.Cfg.Cache.Redis.Password, service.Cfg.Cache.TTL)
	if errCache != nil {
		log.Error("Unable to init cache (%s): %v", service.Cfg.Cache.Redis.Host, errCache)
		return nil, errCache
	}

	return service, nil
}

func newRequest(t *testing.T, s *Service, method, uri string, i interface{}) *http.Request {
	t.Logf("Request: %s %s", method, uri)
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		if err != nil {
			t.FailNow()
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.FailNow()
	}

	assets.AuthentifyRequestFromService(t, req, s.Hash)

	return req
}
