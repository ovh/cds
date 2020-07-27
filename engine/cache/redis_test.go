package cache

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/require"
)

func TestSortedSet(t *testing.T) {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]
	s, err := NewRedisStore(redisHost, redisPassword, 60)
	require.NoError(t, err)

	s.Delete("test")

	require.NoError(t, s.ScoredSetAdd(context.TODO(), "test", "value", 1.0))
	var res []string
	require.NoError(t, s.ScoredSetScan(context.TODO(), "test", 0.1, 2.0, &res))
	require.EqualValues(t, []string{"value"}, res)
}
