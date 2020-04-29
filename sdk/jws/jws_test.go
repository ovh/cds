package jws

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRandomRSAKey(t *testing.T) {
	k, err := NewRandomRSAKey()
	require.NoError(t, err)
	btes, err := ExportPrivateKey(k)
	require.NoError(t, err)
	t.Log(string(btes))
}

func TestHMacSignAndVerify(t *testing.T) {
	secret, err := NewRandomSymmetricKey(32)
	require.NoError(t, err)
	signer, err := NewHMacSigner(secret)
	require.NoError(t, err)

	message := "coucou"
	messageSigned, err := Sign(signer, message)
	require.NoError(t, err)
	require.NotEqual(t, message, messageSigned)

	var unsigned string
	require.NoError(t, Verify(secret, messageSigned, &unsigned))
	require.Equal(t, message, unsigned)
}
