package cdn

import (
	"context"
	"testing"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func newRouter(m *mux.Router, p string) *api.Router {
	r := &api.Router{
		Mux:        m,
		Prefix:     p,
		URL:        "",
		Background: context.Background(),
	}
	return r
}

func newTestService(t *testing.T) (*Service, *test.FakeTransaction) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, end := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(end)

	router := newRouter(mux.NewRouter(), "/"+test.GetTestName(t))
	var cancel context.CancelFunc
	router.Background, cancel = context.WithCancel(context.Background())
	s := &Service{
		Router:              router,
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}
	s.initRouter(context.TODO())

	t.Cleanup(func() { cancel() })
	return s, db
}
