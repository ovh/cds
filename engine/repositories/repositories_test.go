package repositories

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/spacemonkeygo/httpsig.v0"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

var (
	RedisHost     string
	RedisPassword string
)

func init() {
	log.Initialize(context.TODO(), &log.Conf{Level: "debug"})
}

func newTestService(t *testing.T) (*Service, error) {
	//Read the test config file
	if RedisHost == "" {
		cfg := test.LoadTestingConf(t, sdk.TypeAPI)
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
	service.GoRoutines = sdk.NewGoRoutines()
	if fakeAPIPrivateKey.key == nil {
		fakeAPIPrivateKey.key, _ = jws.NewRandomRSAKey()
	}
	service.ParsedAPIPublicKey = &fakeAPIPrivateKey.key.PublicKey
	service.Router = r
	service.initRouter(ctx)
	service.Cfg = cfg

	//Init the cache
	var errCache error
	service.Cache, errCache = cache.New(service.Cfg.Cache.Redis.Host, service.Cfg.Cache.Redis.Password, service.Cfg.Cache.TTL)
	if errCache != nil {
		log.Error(ctx, "Unable to init cache (%s): %v", service.Cfg.Cache.Redis.Host, errCache)
		return nil, errCache
	}

	service.dao = dao{
		store: service.Cache,
	}

	return service, nil
}

var fakeAPIPrivateKey = struct {
	sync.Mutex
	key *rsa.PrivateKey
}{}

func newRequest(t *testing.T, s *Service, method, uri string, i interface{}) *http.Request {
	fakeAPIPrivateKey.Lock()
	defer fakeAPIPrivateKey.Unlock()

	t.Logf("Request: %s %s", method, uri)
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.FailNow()
	}

	HTTPSigner := httpsig.NewRSASHA256Signer("test", fakeAPIPrivateKey.key, []string{"(request-target)", "host", "date"})
	require.NoError(t, HTTPSigner.Sign(req))

	return req
}

func newMultiPartTarRequest(t *testing.T, s *Service, method, uri string, in interface{}, buffer *bytes.Buffer) *http.Request {
	t.Logf("Request: %s %s", method, uri)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create tar part
	tarPartMH := make(textproto.MIMEHeader)
	tarPartMH.Set("Content-Type", "application/tar")
	tarPartMH.Set("Content-Disposition", "form-data; name=\"dataFiles\"; filename=\"data.tar\"")
	dataPart, err := writer.CreatePart(tarPartMH)
	if err != nil {
		t.Errorf("Unable to create data part: %v", err)
		return nil
	}
	if _, err := io.Copy(dataPart, buffer); err != nil {
		t.Errorf("unable to write into data part: %v", err)
		return nil
	}

	// Create json part
	jsonData, errM := json.Marshal(in)
	if errM != nil {
		t.Errorf("unable to marshal data: %v", errM)
		t.FailNow()
	}
	assert.NoError(t, writer.WriteField("dataJSON", string(jsonData)))

	// Close writer
	if err := writer.Close(); err != nil {
		t.Errorf("unable to close writer: %v", err)
		t.FailNow()
	}

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		t.FailNow()
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	HTTPSigner := httpsig.NewRSASHA256Signer("test", fakeAPIPrivateKey.key, []string{"(request-target)", "host", "date"})
	require.NoError(t, HTTPSigner.Sign(req))

	return req
}
