package api

import (
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/test"
)

func newTestAPI(t *testing.T, bootstrapFunc ...test.Bootstrapf) (*API, *gorp.DbMap, *Router) {
	db := test.SetupPG(t, bootstrapFunc...)
	router := newRouter(auth.TestLocalAuth(t, db), mux.NewRouter(), "/"+test.GetTestName(t))
	api := &API{
		StartupTime:         time.Now(),
		Router:              router,
		DBConnectionFactory: test.DBConnectionFactory,
		Config:              Configuration{},
	}
	api.InitRouter()
	return api, db, router
}
