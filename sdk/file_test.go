package sdk

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/pgzip"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// buildTarGz writes the given entries the way the cache plugin does, so the test
// exercises the archives Untar actually receives.
func buildTarGz(t *testing.T, entries []tar.Header, contents map[string][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := pgzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	for i := range entries {
		h := entries[i]
		if body, ok := contents[h.Name]; ok {
			h.Size = int64(len(body))
		}
		require.NoError(t, tw.WriteHeader(&h))
		if body, ok := contents[h.Name]; ok {
			_, err := tw.Write(body)
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	return buf.Bytes()
}

func TestUntarGz(t *testing.T) {
	// One file smaller than the copy buffer, one empty, and one larger so that
	// the reused buffer is filled more than once.
	small := []byte("hello cache")
	big := bytes.Repeat([]byte("0123456789abcdef"), 64*1024) // 1MB, > the 256kB buffer

	entries := []tar.Header{
		{Name: "mod", Typeflag: tar.TypeDir, Mode: 0755},
		{Name: "mod/nested", Typeflag: tar.TypeDir, Mode: 0755},
		{Name: "mod/nested/small.txt", Typeflag: tar.TypeReg, Mode: 0644},
		{Name: "mod/nested/empty.txt", Typeflag: tar.TypeReg, Mode: 0600},
		{Name: "mod/big.bin", Typeflag: tar.TypeReg, Mode: 0444},
		{Name: "mod/link.txt", Typeflag: tar.TypeSymlink, Linkname: "nested/small.txt", Mode: 0777},
	}
	contents := map[string][]byte{
		"mod/nested/small.txt": small,
		"mod/nested/empty.txt": {},
		"mod/big.bin":          big,
	}
	archive := buildTarGz(t, entries, contents)

	dst := t.TempDir()
	fs := afero.NewOsFs()
	require.NoError(t, UntarGz(fs, dst, bytes.NewReader(archive)))

	// Directories
	for _, d := range []string{"mod", "mod/nested"} {
		fi, err := fs.Stat(filepath.Join(dst, d))
		require.NoError(t, err, d)
		require.True(t, fi.IsDir(), d)
	}

	// Contents, including the file larger than the copy buffer
	for name, want := range contents {
		got, err := afero.ReadFile(fs, filepath.Join(dst, name))
		require.NoError(t, err, name)
		require.Equal(t, want, got, name)
	}

	// Modes
	fi, err := fs.Stat(filepath.Join(dst, "mod/nested/small.txt"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0644), fi.Mode().Perm())
	fi, err = fs.Stat(filepath.Join(dst, "mod/big.bin"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0444), fi.Mode().Perm())

	// Symlink
	target, err := os.Readlink(filepath.Join(dst, "mod/link.txt"))
	require.NoError(t, err)
	require.Equal(t, "nested/small.txt", target)
}

// Restoring twice in a row must succeed: a cache is often extracted over a
// directory that already holds part of it.
func TestUntarGzOverExistingContent(t *testing.T) {
	entries := []tar.Header{
		{Name: "mod", Typeflag: tar.TypeDir, Mode: 0755},
		{Name: "mod/f.txt", Typeflag: tar.TypeReg, Mode: 0644},
	}
	contents := map[string][]byte{"mod/f.txt": []byte("second")}
	archive := buildTarGz(t, entries, contents)

	dst := t.TempDir()
	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(filepath.Join(dst, "mod"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(dst, "mod/f.txt"), []byte("first"), 0644))

	require.NoError(t, UntarGz(fs, dst, bytes.NewReader(archive)))

	got, err := afero.ReadFile(fs, filepath.Join(dst, "mod/f.txt"))
	require.NoError(t, err)
	require.Equal(t, []byte("second"), got, "an existing file must be truncated, not appended to")
}

func TestUntarGzInvalidArchive(t *testing.T) {
	err := UntarGz(afero.NewOsFs(), t.TempDir(), io.LimitReader(bytes.NewReader([]byte("not a gzip")), 10))
	require.Error(t, err)
}
