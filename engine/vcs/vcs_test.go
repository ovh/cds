package vcs

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"gopkg.in/spacemonkeygo/httpsig.v0"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
)

var (
	RedisHost     string
	RedisPassword string
)

func init() {
	cdslog.Initialize(context.TODO(), &cdslog.Conf{Level: "debug"})
}

func newTestService(t *testing.T) (*Service, error) {
	fakeAPIPrivateKey.Lock()
	defer fakeAPIPrivateKey.Unlock()
	//Read the test config file
	if RedisHost == "" {
		cfg := test.LoadTestingConf(t, sdk.TypeAPI)
		RedisHost = cfg["redisHost"]
		RedisPassword = cfg["redisPassword"]
	}
	log.Factory = log.NewTestingWrapper(t)

	//Prepare the configuration
	cfg := Configuration{}
	cfg.Cache.TTL = 30
	cfg.Cache.Redis.Host = RedisHost
	cfg.Cache.Redis.Password = RedisPassword
	cfg.Cache.Redis.DbIndex = 0

	ctx := context.Background()
	r := &api.Router{
		Mux:        mux.NewRouter(),
		Prefix:     "/" + test.GetTestName(t),
		Background: ctx,
	}

	service := new(Service)
	service.GoRoutines = sdk.NewGoRoutines(ctx)
	if fakeAPIPrivateKey.key == nil {
		fakeAPIPrivateKey.key, _ = jws.NewRandomRSAKey()
	}
	service.ParsedAPIPublicKey = &fakeAPIPrivateKey.key.PublicKey
	service.Router = r
	service.initRouter(ctx)
	service.Cfg = cfg

	//Init the cache
	var err error
	service.Cache, err = cache.New(context.TODO(), service.Cfg.Cache.Redis, service.Cfg.Cache.TTL)
	if err != nil {
		log.Error(ctx, "Unable to init cache (%s): %v", service.Cfg.Cache.Redis.Host, err)
		return nil, err
	}

	return service, nil
}

func newRequest(t *testing.T, s *Service, method, uri string, i interface{}, opts ...cdsclient.RequestModifier) *http.Request {
	fakeAPIPrivateKey.Lock()
	defer fakeAPIPrivateKey.Unlock()

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

	for _, opt := range opts {
		opt(req)
	}

	HTTPSigner := httpsig.NewRSASHA256Signer("test", fakeAPIPrivateKey.key, []string{"(request-target)", "host", "date"})
	require.NoError(t, HTTPSigner.Sign(req))

	return req
}

var fakeAPIPrivateKey = struct {
	sync.Mutex
	key *rsa.PrivateKey
}{}
