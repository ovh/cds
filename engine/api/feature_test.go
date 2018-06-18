package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/test"
)

func newTestAPIWithIzanamiToken(t *testing.T, token string, bootstrapFunc ...test.Bootstrapf) (*API, *gorp.DbMap, *Router) {
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
	api.Config.Features.Izanami.Token = token
	_ = event.Initialize(event.KafkaConfig{}, api.Cache)
	api.InitRouter()
	return api, db, router
}

func TestFeatureClean(t *testing.T) {
	api, _, router := newTestAPIWithIzanamiToken(t, "mytoken", bootstrap.InitiliazeDB)

	vars := map[string]string{}
	uri := router.GetRoute("POST", api.cleanFeatureHandler, vars)
	req, err := http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	req.Header.Set("X-Izanami-Token", "666")

	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 401, w.Code)

	w = httptest.NewRecorder()
	req.Header.Set("X-Izanami-Token", "mytoken")
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)
}
