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
