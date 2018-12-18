package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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

	service.dao = dao{
		store: service.Cache,
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
	writer.WriteField("dataJSON", string(jsonData))

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

	assets.AuthentifyRequestFromService(t, req, s.Hash)

	return req
}
