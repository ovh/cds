package cache

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"

	testConfig "github.com/ovh/cds/engine/test/config"
	"github.com/ovh/cds/sdk"
)

func TestSortedSet(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	cfg := testConfig.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]
	redisDbIndex, err := strconv.ParseInt(cfg["redisDbIndex"], 10, 64)
	require.NoError(t, err, "error when unmarshal config")

	s, err := NewRedisStore(sdk.RedisConf{Host: redisHost, Password: redisPassword, DbIndex: int(redisDbIndex)}, 60)
	require.NoError(t, err)

	s.Delete("test")

	require.NoError(t, s.ScoredSetAdd(context.TODO(), "test", "value", 1.0))
	var res []string
	require.NoError(t, s.ScoredSetScan(context.TODO(), "test", 0.1, 2.0, &res))
	require.EqualValues(t, []string{"value"}, res)
}

func TestDequeueJSONRawMessagesWithContext(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	cfg := testConfig.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]
	redisDbIndex, err := strconv.ParseInt(cfg["redisDbIndex"], 10, 64)
	require.NoError(t, err, "error when unmarshal config")
	s, err := NewRedisStore(sdk.RedisConf{Host: redisHost, Password: redisPassword, DbIndex: int(redisDbIndex)}, 60)
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
	log.Factory = log.NewTestingWrapper(t)
	cfg := testConfig.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]
	redisDbIndex, err := strconv.ParseInt(cfg["redisDbIndex"], 10, 64)
	require.NoError(t, err, "error when unmarshal config")
	s, err := NewRedisStore(sdk.RedisConf{Host: redisHost, Password: redisPassword, DbIndex: int(redisDbIndex)}, 60)
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

// TestKeysWalksWholeKeyspace covers the SCAN-based Keys(). SCAN differs from
// KEYS in ways that matter to callers: it pages through a cursor, it can return
// the same key more than once, and it needs several round trips when the
// keyspace is larger than one batch.
//
// The batch is deliberately tiny here so a small keyspace still forces many
// cursor steps; production uses keysScanBatch. The result must be complete,
// free of duplicates, and correctly filtered by pattern.
func TestKeysWalksWholeKeyspace(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	cfg := testConfig.LoadTestingConf(t, sdk.TypeAPI)
	redisDbIndex, err := strconv.ParseInt(cfg["redisDbIndex"], 10, 64)
	require.NoError(t, err, "error when unmarshal config")

	s, err := NewRedisStore(sdk.RedisConf{Host: cfg["redisHost"], Password: cfg["redisPassword"], DbIndex: int(redisDbIndex)}, 60)
	require.NoError(t, err)

	const total = 250
	prefix := "test:keys:" + sdk.RandomString(8)
	expected := make(map[string]struct{}, total)
	for i := 0; i < total; i++ {
		k := prefix + ":" + strconv.Itoa(i)
		require.NoError(t, s.SetWithTTL(k, "v", 120))
		expected[k] = struct{}{}
	}
	// A key that must not match, so the pattern is proven to still filter.
	other := "test:other:" + sdk.RandomString(8)
	require.NoError(t, s.SetWithTTL(other, "v", 120))

	t.Cleanup(func() {
		for k := range expected {
			s.Delete(k)
		}
		s.Delete(other)
	})

	// A batch far smaller than the keyspace: the walk cannot complete in one
	// step, so the cursor loop itself is under test.
	found, err := s.scanKeys(prefix+":*", 10)
	require.NoError(t, err)

	require.Len(t, found, total, "Keys must return the whole keyspace matching the pattern, across cursor pages")
	seen := make(map[string]struct{}, len(found))
	for _, k := range found {
		_, dup := seen[k]
		require.False(t, dup, "Keys must not return a key twice even though SCAN can repeat across cursor steps: %s", k)
		seen[k] = struct{}{}
		_, want := expected[k]
		require.True(t, want, "Keys returned a key outside the requested pattern: %s", k)
	}

	// And the exported entry point, on the production batch, agrees.
	viaKeys, err := s.Keys(prefix + ":*")
	require.NoError(t, err)
	require.Len(t, viaKeys, total, "Keys must return the same set whatever the batch size")
}

func TestKeysReturnsEmptyForUnmatchedPattern(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	cfg := testConfig.LoadTestingConf(t, sdk.TypeAPI)
	redisDbIndex, err := strconv.ParseInt(cfg["redisDbIndex"], 10, 64)
	require.NoError(t, err, "error when unmarshal config")

	s, err := NewRedisStore(sdk.RedisConf{Host: cfg["redisHost"], Password: cfg["redisPassword"], DbIndex: int(redisDbIndex)}, 60)
	require.NoError(t, err)

	// A cursor walk that matches nothing must terminate and return empty rather
	// than nil-with-error or spinning on a non-zero cursor.
	found, err := s.Keys("test:nothing:" + sdk.RandomString(12) + ":*")
	require.NoError(t, err)
	require.Empty(t, found)
}
