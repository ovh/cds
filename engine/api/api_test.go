package api

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/authentication/local"
	authdrivertest "github.com/ovh/cds/engine/api/authentication/test"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
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
	api.AuthenticationDrivers = make(map[sdk.AuthConsumerType]sdk.AuthDriver)
	api.AuthenticationDrivers[sdk.ConsumerLocal] = local.NewDriver(false, "http://localhost:4200", "")
	api.AuthenticationDrivers[sdk.ConsumerBuiltin] = builtin.NewDriver()
	api.AuthenticationDrivers[sdk.ConsumerTest] = authdrivertest.NewDriver(t)
	api.AuthenticationDrivers[sdk.ConsumerTest2] = authdrivertest.NewDriver(t)

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
	api.AuthenticationDrivers = make(map[sdk.AuthConsumerType]sdk.AuthDriver)
	api.AuthenticationDrivers[sdk.ConsumerLocal] = local.NewDriver(false, "http://localhost:4200", "")
	api.AuthenticationDrivers[sdk.ConsumerBuiltin] = builtin.NewDriver()

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
