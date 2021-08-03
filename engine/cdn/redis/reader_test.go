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

func TestReader_EOF(t *testing.T) {
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	c, err := cache.New(cfg["redisHost"], cfg["redisPassword"], -1)
	require.NoError(t, err)

	cacheKey := cache.Key("test:cdn:item")

	itemID := sdk.RandomString(10)

	w := &redis.Writer{
		Store:     c,
		ItemID:    itemID,
		PrefixKey: cacheKey,
	}

	r := &redis.Reader{
		Store:     c,
		ItemID:    itemID,
		PrefixKey: cacheKey,
		Format:    sdk.CDNReaderFormatJSON,
	}

	t.Logf("writings logs for fake item with id: %s", itemID)

	// Insert lines in redis
	_, err = io.Copy(w, strings.NewReader("this is the first line\nthis is the second line\n"))
	require.NoError(t, err)

	// Read all lines without closing the reader
	buf1 := new(strings.Builder)
	_, err = io.Copy(buf1, r)
	require.NoError(t, err)
	require.Equal(t, "[{\"number\":0,\"value\":\"this is the first line\\n\",\"api_ref_hash\":\"\"},{\"number\":1,\"value\":\"this is the second line\\n\",\"api_ref_hash\":\"\"}]", buf1.String())

	// Insert a new line in redis
	_, err = io.Copy(w, strings.NewReader("this is the third line\n"))
	require.NoError(t, err)

	// Try to read again from same reader should return an error
	buf := make([]byte, 1)
	_, err = r.Read(buf)
	require.Error(t, err, "read should return EOF")
}
