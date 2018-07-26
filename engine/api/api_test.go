package api

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/warning"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func newTestAPI(t *testing.T, bootstrapFunc ...test.Bootstrapf) (*API, *gorp.DbMap, *Router) {
	bootstrapFunc = append(bootstrapFunc, bootstrap.InitiliazeDB)
	db, cache := test.SetupPG(t, bootstrapFunc...)
	router := newRouter(auth.TestLocalAuth(t, db, sessionstore.Options{Cache: cache, TTL: 30}), mux.NewRouter(), "/"+test.GetTestName(t))
	api := &API{
		StartupTime:         time.Now(),
		Router:              router,
		DBConnectionFactory: test.DBConnectionFactory,
		Config:              Configuration{},
		Cache:               cache,
	}
	_ = event.Initialize(event.KafkaConfig{}, api.Cache)
	api.InitRouter()
	api.warnChan = make(chan sdk.Event)
	event.Subscribe(api.warnChan)

	sdk.GoRoutine("workflow.ComputeAudit", func() { workflow.ComputeAudit(context.Background(), api.DBConnectionFactory.GetDBMap) })
	sdk.GoRoutine("warning.Start", func() { warning.Start(context.Background(), api.DBConnectionFactory.GetDBMap, api.warnChan) })

	return api, db, router
}

func newTestServer(t *testing.T, bootstrapFunc ...test.Bootstrapf) (*API, string, func()) {
	bootstrapFunc = append(bootstrapFunc, bootstrap.InitiliazeDB)
	db, cache := test.SetupPG(t, bootstrapFunc...)
	router := newRouter(auth.TestLocalAuth(t, db, sessionstore.Options{Cache: cache, TTL: 30}), mux.NewRouter(), "")
	api := &API{
		StartupTime:         time.Now(),
		Router:              router,
		DBConnectionFactory: test.DBConnectionFactory,
		Config:              Configuration{},
		Cache:               cache,
	}
	_ = event.Initialize(event.KafkaConfig{}, api.Cache)
	api.InitRouter()
	ts := httptest.NewServer(router.Mux)
	url, _ := url.Parse(ts.URL)
	return api, url.String(), ts.Close
}
