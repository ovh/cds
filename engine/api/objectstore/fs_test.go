package objectstore

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func TestFilesystemStoreReplaceObject(t *testing.T) {
	basedir := t.TempDir()

	fss, err := newFilesystemStore(context.TODO(), sdk.ProjectIntegration{}, ConfigOptionsFilesystem{Basedir: basedir})
	require.NoError(t, err)

	o := sdk.GRPCPluginBinary{Name: "my-plugin", OS: "linux", Arch: "amd64"}

	_, err = fss.Store(o, io.NopCloser(strings.NewReader("v1")))
	require.NoError(t, err)

	_, err = fss.Store(o, io.NopCloser(strings.NewReader("v2-longer-content")))
	require.NoError(t, err)

	f, err := fss.Fetch(context.TODO(), o)
	require.NoError(t, err)
	content, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.Equal(t, "v2-longer-content", string(content))

	// the temporary file used to write the object must not be left behind
	entries, err := os.ReadDir(basedir + "/" + o.GetPath())
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, o.GetName(), entries[0].Name())
}
