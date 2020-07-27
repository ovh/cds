package hooks

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/test"
)

func setupTestHookService(t *testing.T) (Service, func()) {
	s := Service{}
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	s.Cfg.RetryError = 1

	store, err := cache.NewRedisStore(redisHost, redisPassword, 60)
	if err != nil {
		t.Fatalf("Unable to connect to redis: %v", err)
	}
	s.Dao = dao{
		store: store,
	}
	s.Cache = store

	ctrl := gomock.NewController(t)
	s.Client = mock_cdsclient.NewMockInterface(ctrl)

	t.Cleanup(func() {
		store.Client.Close()
		store.Client = nil
		ctrl.Finish()
	})

	return s, func() {}
}
