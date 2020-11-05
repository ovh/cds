package cdn

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"gopkg.in/spacemonkeygo/httpsig.v0"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

func newRouter(m *mux.Router, p string) *api.Router {
	r := &api.Router{
		Mux:    m,
		Prefix: p,
		URL:    "",
	}
	return r
}

func newTestService(t *testing.T) (*Service, *test.FakeTransaction) {
	fakeAPIPrivateKey.Lock()
	defer fakeAPIPrivateKey.Unlock()

	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, end := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(end)

	router := newRouter(mux.NewRouter(), "/"+test.GetTestName(t))
	s := &Service{
		Router:              router,
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}
	s.GoRoutines = sdk.NewGoRoutines()
	if fakeAPIPrivateKey.key == nil {
		fakeAPIPrivateKey.key, _ = jws.NewRandomRSAKey()
	}
	s.Common.GoRoutines = sdk.NewGoRoutines()
	s.ParsedAPIPublicKey = &fakeAPIPrivateKey.key.PublicKey

	ctx, cancel := context.WithCancel(context.Background())
	s.initRouter(ctx)
	t.Cleanup(cancel)

	return s, db
}

func newRequest(t *testing.T, method, uri string, i interface{}, opts ...cdsclient.RequestModifier) *http.Request {
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

func newRunningStorageUnits(t *testing.T, m *gorpmapper.Mapper, dbMap *gorp.DbMap, ctx context.Context, maxStepSize int64) *storage.RunningStorageUnits {
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)
	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	t.Cleanup(cancel)

	cdnUnits, err := storage.Init(ctx, m, dbMap, sdk.NewGoRoutines(), storage.Configuration{
		HashLocatorSalt: "thisismysalt",
		Buffer: storage.BufferConfiguration{
			Name: "redis_buffer",
			Redis: storage.RedisBufferConfiguration{
				Host:     cfg["redisHost"],
				Password: cfg["redisPassword"],
			},
		},
		Storages: []storage.StorageConfiguration{
			{
				Name: "local_storage",
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
				},
			},
		},
	}, storage.LogConfig{NbJobLogsGoroutines: 0, NbServiceLogsGoroutines: 0, StepMaxSize: maxStepSize, ServiceMaxSize: 10000, StepLinesRateLimit: 10000})
	require.NoError(t, err)
	cdnUnits.Start(ctx, sdk.NewGoRoutines())
	return cdnUnits
}
