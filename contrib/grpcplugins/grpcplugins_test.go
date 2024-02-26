package grpcplugins

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_checksums(t *testing.T) {
	c, err := checksums(context.TODO(), os.DirFS("."), "main.go")
	require.NoError(t, err)
	t.Log(c)
}
