package redis_test

import (
	"io"
	"strings"
	"testing"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestWriter_Closed(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)
	c, err := cache.New(sdk.RedisConf{Host: cfg["redisHost"], Password: cfg["redisPassword"], DbIndex: 0}, -1)
	require.NoError(t, err)

	cacheKey := cache.Key("test:cdn:item")
	itemID := sdk.RandomString(10)

	w := &redis.Writer{
		Store:     c,
		ItemID:    itemID,
		PrefixKey: cacheKey,
	}

	_, err = io.Copy(w, strings.NewReader("this is the first line\nthis is the second line\n"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	_, err = io.Copy(w, strings.NewReader("this is the third line\n"))
	require.Error(t, err)
	require.Equal(t, "writer is closed", err.Error())
}
