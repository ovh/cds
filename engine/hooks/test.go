package hooks

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"

	"github.com/ovh/cds/engine/api/cache"
)

func setupTestHookService(t *testing.T) (Service, func()) {
	s := Service{}
	cfg := test.LoadTestingConf(t)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	store, err := cache.NewRedisStore(redisHost, redisPassword, 60)
	if err != nil {
		t.Fatalf("Unable to connect to redis: %v", err)
	}
	s.Dao = dao{
		store: store,
	}

	cancel := func() {
		store.Client.Close()
		store.Client = nil
	}

	return s, cancel
}
