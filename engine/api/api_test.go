package api

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/authentication/local"
	authdrivertest "github.com/ovh/cds/engine/api/authentication/test"
	"github.com/ovh/cds/engine/api/bootstrap"
	apiTest "github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func newTestAPI(t *testing.T, bootstrapFunc ...test.Bootstrapf) (*API, *test.FakeTransaction, *Router) {
	bootstrapFunc = append(bootstrapFunc, bootstrap.InitiliazeDB)
	db, factory, cache := apiTest.SetupPGWithFactory(t, bootstrapFunc...)
	router := newRouter(mux.NewRouter(), "/"+test.GetTestName(t))
	var cancel context.CancelFunc
	router.Background, cancel = context.WithCancel(context.Background())
	api := &API{
		StartupTime:         time.Now(),
		Router:              router,
		DBConnectionFactory: factory,
		Config:              Configuration{},
		Cache:               cache,
	}
	api.AuthenticationDrivers = make(map[sdk.AuthConsumerType]sdk.AuthDriver)
	api.AuthenticationDrivers[sdk.ConsumerLocal] = local.NewDriver(context.TODO(), false, "http://localhost:8080", "")
	api.AuthenticationDrivers[sdk.ConsumerBuiltin] = builtin.NewDriver()
	api.AuthenticationDrivers[sdk.ConsumerTest] = authdrivertest.NewDriver(t)
	api.AuthenticationDrivers[sdk.ConsumerTest2] = authdrivertest.NewDriver(t)
	api.GoRoutines = sdk.NewGoRoutines()

	api.InitRouter()
	t.Cleanup(func() { cancel() })
	return api, db, router
}

func newRouter(m *mux.Router, p string) *Router {
	r := &Router{
		Mux:              m,
		Prefix:           p,
		URL:              "",
		mapRouterConfigs: map[string]*service.RouterConfig{},
		Background:       context.Background(),
	}
	return r
}

func newTestServer(t *testing.T, bootstrapFunc ...test.Bootstrapf) (*API, *test.FakeTransaction, string) {
	bootstrapFunc = append(bootstrapFunc, bootstrap.InitiliazeDB)
	db, factory, cache := apiTest.SetupPGWithFactory(t, bootstrapFunc...)
	router := newRouter(mux.NewRouter(), "")
	var cancel context.CancelFunc
	router.Background, cancel = context.WithCancel(context.Background())
	api := &API{
		StartupTime:         time.Now(),
		Router:              router,
		DBConnectionFactory: factory,
		Config:              Configuration{},
		Cache:               cache,
	}
	api.AuthenticationDrivers = make(map[sdk.AuthConsumerType]sdk.AuthDriver)
	api.AuthenticationDrivers[sdk.ConsumerLocal] = local.NewDriver(context.TODO(), false, "http://localhost:8080", "")
	api.AuthenticationDrivers[sdk.ConsumerBuiltin] = builtin.NewDriver()
	api.GoRoutines = sdk.NewGoRoutines()

	api.InitRouter()
	ts := httptest.NewServer(router.Mux)
	url, _ := url.Parse(ts.URL)
	t.Cleanup(func() {
		cancel()
		ts.Close()
	})
	return api, db, url.String()
}
