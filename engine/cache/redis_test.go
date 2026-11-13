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

// TestKeysWalksWholeKeyspace checks a walk spanning many cursor steps returns
// every matching key and nothing else. The batch is far smaller than the
// keyspace so the cursor loop, rather than a single round trip, is what runs.
//
// De-duplication is covered by TestAppendUnseenDropsKeysRepeatedAcrossCursorSteps.
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
	// A key outside the pattern, which must not come back.
	other := "test:other:" + sdk.RandomString(8)
	require.NoError(t, s.SetWithTTL(other, "v", 120))

	t.Cleanup(func() {
		for k := range expected {
			s.Delete(k)
		}
		s.Delete(other)
	})

	// A batch far smaller than the keyspace, so the walk takes many steps.
	found, err := s.scanKeys(prefix+":*", 10)
	require.NoError(t, err)

	require.Len(t, found, total, "Keys must return the whole keyspace matching the pattern, across cursor pages")
	for _, k := range found {
		_, want := expected[k]
		require.True(t, want, "Keys returned a key outside the requested pattern: %s", k)
	}

	// The exported entry point, on the production batch, returns the same set.
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

	// A pattern matching nothing must terminate and return empty.
	found, err := s.Keys("test:nothing:" + sdk.RandomString(12) + ":*")
	require.NoError(t, err)
	require.Empty(t, found)
}

// TestAppendUnseenDropsKeysRepeatedAcrossCursorSteps checks a key returned by
// more than one cursor step, or twice within one, appears once in the result.
// SCAN returns that when the keyspace is resized while the cursor is open.
func TestAppendUnseenDropsKeysRepeatedAcrossCursorSteps(t *testing.T) {
	seen := make(map[string]struct{})
	var keys []string

	keys = appendUnseen(seen, keys, []string{"a", "b"})
	// Second cursor step repeats a key from the first and adds a new one.
	keys = appendUnseen(seen, keys, []string{"b", "c"})
	// Third repeats within its own batch.
	keys = appendUnseen(seen, keys, []string{"c", "c", "d"})

	require.Equal(t, []string{"a", "b", "c", "d"}, keys, "each key must appear exactly once, in the order it was first seen")
}

// TestDeleteAllRemovesTheWholeMatchingKeyspace checks DeleteAll removes every
// key matching the pattern across a keyspace larger than one cursor step, and
// leaves keys outside the pattern alone.
func TestDeleteAllRemovesTheWholeMatchingKeyspace(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	cfg := testConfig.LoadTestingConf(t, sdk.TypeAPI)
	redisDbIndex, err := strconv.ParseInt(cfg["redisDbIndex"], 10, 64)
	require.NoError(t, err, "error when unmarshal config")

	s, err := NewRedisStore(sdk.RedisConf{Host: cfg["redisHost"], Password: cfg["redisPassword"], DbIndex: int(redisDbIndex)}, 60)
	require.NoError(t, err)

	const total = 250
	prefix := "test:deleteall:" + sdk.RandomString(8)
	for i := 0; i < total; i++ {
		require.NoError(t, s.SetWithTTL(prefix+":"+strconv.Itoa(i), "v", 120))
	}
	// A key outside the pattern, which must survive.
	other := "test:deleteall-other:" + sdk.RandomString(8)
	require.NoError(t, s.SetWithTTL(other, "v", 120))
	t.Cleanup(func() { s.Delete(other) })

	require.NoError(t, s.DeleteAll(prefix+":*"))

	left, err := s.Keys(prefix + ":*")
	require.NoError(t, err)
	require.Empty(t, left, "DeleteAll must remove every key matching the pattern")

	survived, err := s.Keys(other)
	require.NoError(t, err)
	require.Len(t, survived, 1, "DeleteAll must not touch keys outside the pattern")
}
