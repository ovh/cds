package cdn

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ovh/symmecrypt/keyloader"

	"github.com/ovh/cds/engine/cache"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
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
	cdslog "github.com/ovh/cds/sdk/log"
)

func init() {
	cdslog.Initialize(context.TODO(), &cdslog.Conf{Level: "debug"})
}

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

	log.Factory = log.NewTestingWrapper(t)
	db, factory, cache, end := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(end)

	router := newRouter(mux.NewRouter(), "/"+test.GetTestName(t))
	s := &Service{
		Router:              router,
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}
	s.GoRoutines = sdk.NewGoRoutines(context.TODO())
	if fakeAPIPrivateKey.key == nil {
		fakeAPIPrivateKey.key, _ = jws.NewRandomRSAKey()
	}
	s.Common.GoRoutines = sdk.NewGoRoutines(context.TODO())
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

func newRunningStorageUnits(t *testing.T, m *gorpmapper.Mapper, dbMap *gorp.DbMap, ctx context.Context, store cache.Store) *storage.RunningStorageUnits {
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)
	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	tmpDirBuf, err := os.MkdirTemp("", t.Name()+"-cdn-1-buf-*")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	t.Cleanup(cancel)

	cdnUnits, err := storage.Init(ctx, m, store, dbMap, sdk.NewGoRoutines(ctx), storage.Configuration{
		SyncSeconds:     2,
		SyncNbElements:  100,
		PurgeSeconds:    30,
		PurgeNbElements: 100,
		HashLocatorSalt: "thisismysalt",
		Buffers: map[string]storage.BufferConfiguration{
			"redis_buffer": {
				Redis: &storage.RedisBufferConfiguration{
					Host:     cfg["redisHost"],
					Password: cfg["redisPassword"],
				},
				BufferType: storage.CDNBufferTypeLog,
			},
			"fs_buffer": {
				Local: &storage.LocalBufferConfiguration{
					Path: tmpDirBuf,
					Encryption: []*keyloader.KeyConfig{
						{
							Identifier: "fs-buf-id",
							Cipher:     "aes-gcm",
							Key:        "12345678901234567890123456789012",
						},
					},
				},
				BufferType: storage.CDNBufferTypeFile,
			},
		},
		Storages: map[string]storage.StorageConfiguration{
			"local_storage": {
				SyncParallel: 10,
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
				},
			},
		},
	})
	require.NoError(t, err)
	cdnUnits.Start(ctx, sdk.NewGoRoutines(ctx))
	return cdnUnits
}
