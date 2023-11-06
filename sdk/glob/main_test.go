package glob_test

import (
	"os"
	"testing"

	"github.com/ovh/cds/sdk/glob"
	"github.com/stretchr/testify/require"
)

func TestGlob(t *testing.T) {
	pattern := "path/to/**/* !path/to/**/*.tmp"
	result, err := glob.Glob(os.DirFS("tests/"), "fixtures", pattern)
	require.NoError(t, err)
	require.Equal(t, "path/to/artifacts/bar, path/to/artifacts/foo, path/to/results/foo.bin", result.String())
}
