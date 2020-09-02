package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	testConfig "github.com/ovh/cds/engine/test/config"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func TestSortedSet(t *testing.T) {
	log.SetLogger(t)
	cfg := testConfig.LoadTestingConf(t, sdk.TypeAPI)
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

func TestDequeueJSONRawMessagesWithContext(t *testing.T) {
	log.SetLogger(t)
	cfg := testConfig.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]
	s, err := NewRedisStore(redisHost, redisPassword, 60)
	require.NoError(t, err)

	s.Delete("test")

	msgs := make([]string, 100)
	for i := 0; i < 100; i++ {
		msgs[i] = sdk.RandomString(10)
		require.NoError(t, s.Enqueue("test", msgs[i]))
	}

	l, err := s.QueueLen("test")
	require.NoError(t, err)
	require.Equal(t, 100, l)

	data, err := s.DequeueJSONRawMessagesWithContext(context.TODO(), "test", 30*time.Millisecond, 50)
	require.NoError(t, err)
	require.Equal(t, 50, len(data))

	data2, err := s.DequeueJSONRawMessagesWithContext(context.TODO(), "test", 30*time.Millisecond, 50)
	require.NoError(t, err)
	require.Equal(t, 50, len(data2))

	ctx := context.TODO()
	ctx2, _ := context.WithTimeout(ctx, 200*time.Millisecond)
	data3, _ := s.DequeueJSONRawMessagesWithContext(ctx2, "test", 30*time.Millisecond, 50)
	require.Equal(t, 00, len(data3))

	ctx3, _ := context.WithTimeout(ctx, 100*time.Millisecond)
	data4, _ := s.DequeueJSONRawMessagesWithContext(ctx3, "test", 30*time.Millisecond, 50)
	require.Equal(t, 00, len(data4))
}

func TestDequeueJSONRawMessagesWithContextMaxTimeout(t *testing.T) {
	log.SetLogger(t)
	cfg := testConfig.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]
	s, err := NewRedisStore(redisHost, redisPassword, 60)
	require.NoError(t, err)

	s.Delete("test")

	msgs := make([]string, 100)
	for i := 0; i < 100; i++ {
		msgs[i] = sdk.RandomString(10)
		require.NoError(t, s.Enqueue("test", msgs[i]))
	}

	l, err := s.QueueLen("test")
	require.NoError(t, err)
	require.Equal(t, 100, l)

	ctx := context.TODO()

	ctx2, _ := context.WithTimeout(ctx, 10*time.Millisecond)
	data, err := s.DequeueJSONRawMessagesWithContext(ctx2, "test", 30*time.Millisecond, 50)
	require.NoError(t, err)
	require.Equal(t, 0, len(data))

	data2, err := s.DequeueJSONRawMessagesWithContext(context.TODO(), "test", 30*time.Millisecond, 5)
	require.NoError(t, err)
	require.Equal(t, 5, len(data2))

	l2, err := s.QueueLen("test")
	require.NoError(t, err)
	require.Equal(t, 95, l2)
}
