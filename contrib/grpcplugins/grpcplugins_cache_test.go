package grpcplugins

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/pgzip"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func cacheArchive(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := pgzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	// The directory comes first, as filepath.Walk emits it when the archive is
	// built: Untar does not create the parent of a file on its own.
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "mod", Typeflag: tar.TypeDir, Mode: 0755,
	}))
	body := []byte("cached")
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "mod/f.txt", Typeflag: tar.TypeReg, Mode: 0644, Size: int64(len(body)),
	}))
	_, err := tw.Write(body)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	return buf.Bytes()
}

func TestExtractCache(t *testing.T) {
	dst := t.TempDir()
	found, err := extractCache(&actionplugin.Common{}, bytes.NewReader(cacheArchive(t)), dst)
	require.NoError(t, err)
	require.True(t, found)

	got, err := os.ReadFile(filepath.Join(dst, "mod/f.txt"))
	require.NoError(t, err)
	require.Equal(t, []byte("cached"), got)
}

// An archive left truncated, by an interrupted or a concurrent upload, must be
// reported as a miss so that the job carries on and rebuilds what it needs.
func TestExtractCacheTruncated(t *testing.T) {
	archive := cacheArchive(t)
	for _, tc := range []struct {
		name string
		body []byte
	}{
		{"cut in the middle", archive[:len(archive)/2]},
		{"header only", archive[:5]},
		{"empty", nil},
		{"not a gzip stream", []byte("this is not an archive")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			found, err := extractCache(&actionplugin.Common{}, bytes.NewReader(tc.body), t.TempDir())
			require.NoError(t, err, "an unusable cache must not fail the job")
			require.False(t, found, "an unusable cache must be reported as a miss")
		})
	}
}
