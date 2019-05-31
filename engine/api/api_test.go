package api

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/localauthentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
)

func newTestAPI(t *testing.T, bootstrapFunc ...test.Bootstrapf) (*API, *gorp.DbMap, *Router, context.CancelFunc) {
	bootstrapFunc = append(bootstrapFunc, bootstrap.InitiliazeDB)
	db, cache, end := test.SetupPG(t, bootstrapFunc...)
	router := newRouter(mux.NewRouter(), "/"+test.GetTestName(t))
	var cancel context.CancelFunc
	router.Background, cancel = context.WithCancel(context.Background())
	api := &API{
		StartupTime:         time.Now(),
		Router:              router,
		DBConnectionFactory: test.DBConnectionFactory,
		Config:              Configuration{},
		Cache:               cache,
	}
	api.AuthenticationDrivers = make(map[string]authentication.Driver)
	api.AuthenticationDrivers["local"] = localauthentication.New()
	// TODO
	//auth.Store, _ = sessionstore.Get(context.Background(), cache, 60)
	api.InitRouter()
	f := func() {
		cancel()
		end()
	}
	return api, db, router, f
}

func newTestServer(t *testing.T, bootstrapFunc ...test.Bootstrapf) (*API, string, func()) {
	bootstrapFunc = append(bootstrapFunc, bootstrap.InitiliazeDB)
	_, cache, end := test.SetupPG(t, bootstrapFunc...)
	router := newRouter(mux.NewRouter(), "")
	var cancel context.CancelFunc
	router.Background, cancel = context.WithCancel(context.Background())
	api := &API{
		StartupTime:         time.Now(),
		Router:              router,
		DBConnectionFactory: test.DBConnectionFactory,
		Config:              Configuration{},
		Cache:               cache,
	}
	api.AuthenticationDrivers = make(map[string]authentication.Driver)
	api.AuthenticationDrivers["local"] = localauthentication.New()
	// TODO
	//auth.Store, _ = sessionstore.Get(context.Background(), cache, 60)
	api.InitRouter()
	ts := httptest.NewServer(router.Mux)
	url, _ := url.Parse(ts.URL)
	f := func() {
		end()
		cancel()
		ts.Close()
	}
	return api, url.String(), f
}
