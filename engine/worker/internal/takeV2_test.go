package internal

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

// The signing key given at take time is the only thing that makes the CDN accept the logs of the
// job. Decoding it into a fixed size buffer keeps the trailing zeros of the buffer when the key
// is shorter than the buffer: the worker then signs every line with a key nobody can verify and
// the whole job loses its logs. So a key must always be decoded at its own length. Both take
// paths (Take for v1 and V2Take) go through this helper, which is the only place where the
// decoding happens.
func TestDecodeSigningKey(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	t.Run("a key shorter than 32 bytes is not zero padded", func(t *testing.T) {
		shortKey := []byte("a-signing-key-shorter-than-32b")
		require.Less(t, len(shortKey), signingKeyLength)

		decoded, err := decodeSigningKey(context.TODO(), base64.StdEncoding.EncodeToString(shortKey))
		require.NoError(t, err)
		require.Equal(t, shortKey, decoded)
	})

	t.Run("a 32 bytes key is decoded as is", func(t *testing.T) {
		key := make([]byte, signingKeyLength)
		for i := range key {
			key[i] = byte(i)
		}

		decoded, err := decodeSigningKey(context.TODO(), base64.StdEncoding.EncodeToString(key))
		require.NoError(t, err)
		require.Equal(t, key, decoded)
	})

	t.Run("an empty key fails the take instead of silently signing with zeros", func(t *testing.T) {
		_, err := decodeSigningKey(context.TODO(), "")
		require.Error(t, err)
	})

	t.Run("an invalid base64 key fails the take", func(t *testing.T) {
		_, err := decodeSigningKey(context.TODO(), "not-a-base64-key!")
		require.Error(t, err)
	})
}
