package plugin

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime/mock_workerruntime"
	"github.com/ovh/cds/sdk"
)

const (
	testPluginName = "my-plugin"
	testBinaryName = "my-plugin-linux-amd64"
	testOS         = "linux"
	testARCH       = "amd64"
)

// setupBinaryTest returns a mocked runtime backed by an in memory basedir. The retry delay is
// neutralized so that tests exercising the retry loop stay fast.
func setupBinaryTest(t *testing.T) (*mock_workerruntime.MockRuntime, afero.Fs) {
	log.Factory = log.NewTestingWrapper(t)

	previousDelay := binaryDownloadRetryDelay
	binaryDownloadRetryDelay = time.Millisecond
	t.Cleanup(func() { binaryDownloadRetryDelay = previousDelay })

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	fs := afero.NewMemMapFs()
	w := mock_workerruntime.NewMockRuntime(ctrl)
	w.EXPECT().BaseDir().Return(fs).AnyTimes()
	w.EXPECT().GetCheckedPluginBinary(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	w.EXPECT().SetCheckedPluginBinary(gomock.Any()).AnyTimes()

	return w, fs
}

// pluginServing builds the plugin as published by the API for the given binary content.
func pluginServing(t *testing.T, content string) *sdk.GRPCPlugin {
	t.Helper()
	sum, err := sdk.SHA512sum(content)
	require.NoError(t, err)

	return &sdk.GRPCPlugin{
		Name: testPluginName,
		Binaries: sdk.GRPCPluginBinaries{{
			PluginName: testPluginName,
			Name:       testBinaryName,
			OS:         testOS,
			Arch:       testARCH,
			Perm:       0755,
			SHA512sum:  sum,
			Cmd:        "./my-plugin",
		}},
	}
}

// serve makes PluginGetBinary write the given content, as the API would.
func serve(content string) func(name, os, arch string, w io.Writer) error {
	return func(_, _, _ string, w io.Writer) error {
		_, err := w.Write([]byte(content))
		return err
	}
}

func writeCache(t *testing.T, fs afero.Fs, content string) {
	t.Helper()
	require.NoError(t, afero.WriteFile(fs, testBinaryName, []byte(content), os.FileMode(0755)))
}

// A binary published without a sha512sum cannot be authenticated, so it must never reach the
// disk: falling back on the md5sum would silently rest the guarantee on a broken algorithm.
func TestDownloadBinary_RefusesBinaryWithoutSHA512(t *testing.T) {
	w, fs := setupBinaryTest(t)

	p := pluginServing(t, "binary content")
	p.Binaries[0].SHA512sum = ""
	p.Binaries[0].MD5sum = "d41d8cd98f00b204e9800998ecf8427e"

	w.EXPECT().PluginGet(testPluginName).Return(p, nil)
	w.EXPECT().PluginGetBinary(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := DownloadBinary(context.TODO(), w, testPluginName, testOS, testARCH)
	require.Error(t, err)
	require.True(t, sdk.ErrorIs(err, sdk.ErrPluginInvalid), "got %v", err)

	exists, err := afero.Exists(fs, testBinaryName)
	require.NoError(t, err)
	require.False(t, exists, "no binary should have been written")
}

// A binary whose content does not match the published checksum must never be handed to the
// job, and the failure must be fast: retrying a deterministic mismatch 20 times would be
// paid again by every step using the plugin.
func TestDownloadBinary_RefusesTamperedDownload(t *testing.T) {
	w, _ := setupBinaryTest(t)

	w.EXPECT().PluginGet(testPluginName).Return(pluginServing(t, "the published binary"), nil)
	w.EXPECT().
		PluginGetBinary(testPluginName, testOS, testARCH, gomock.Any()).
		DoAndReturn(serve("a tampered binary")).
		Times(binaryChecksumRetry)

	_, err := DownloadBinary(context.TODO(), w, testPluginName, testOS, testARCH)
	require.Error(t, err)
	require.True(t, sdk.ErrorIs(err, sdk.ErrPluginInvalid), "got %v", err)
	require.Contains(t, err.Error(), "sha512 mismatch")
}

// The worker basedir defaults to os.TempDir() and can be shared between workers: a binary
// found there is not trustworthy just because it exists under the expected name.
func TestDownloadBinary_RejectsTamperedCache(t *testing.T) {
	w, fs := setupBinaryTest(t)
	writeCache(t, fs, "a tampered binary planted in the basedir")

	w.EXPECT().PluginGet(testPluginName).Return(pluginServing(t, "the published binary"), nil)
	w.EXPECT().
		PluginGetBinary(testPluginName, testOS, testARCH, gomock.Any()).
		DoAndReturn(serve("the published binary")).
		Times(1)

	b, err := DownloadBinary(context.TODO(), w, testPluginName, testOS, testARCH)
	require.NoError(t, err)
	require.Equal(t, testBinaryName, b.Name)

	content, err := afero.ReadFile(fs, testBinaryName)
	require.NoError(t, err)
	require.Equal(t, "the published binary", string(content), "the planted binary must have been replaced")
}

func TestDownloadBinary_UsesValidCacheWithoutDownloading(t *testing.T) {
	w, fs := setupBinaryTest(t)
	writeCache(t, fs, "the published binary")

	w.EXPECT().PluginGet(testPluginName).Return(pluginServing(t, "the published binary"), nil)
	w.EXPECT().PluginGetBinary(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	b, err := DownloadBinary(context.TODO(), w, testPluginName, testOS, testARCH)
	require.NoError(t, err)
	require.Equal(t, testBinaryName, b.Name)
}

// The returned descriptor must be the one read from the API, not the one the caller holds:
// for integration plugins the caller's descriptor comes from the job payload, which is frozen
// when the run is scheduled and can be older than the binary actually served.
func TestDownloadBinary_ReturnsDescriptorReadFromAPI(t *testing.T) {
	w, _ := setupBinaryTest(t)

	p := pluginServing(t, "the published binary")
	p.Binaries[0].Cmd = "./my-plugin-v2"
	p.Binaries[0].Entrypoints = []string{"entrypoint-v2"}

	w.EXPECT().PluginGet(testPluginName).Return(p, nil)
	w.EXPECT().
		PluginGetBinary(testPluginName, testOS, testARCH, gomock.Any()).
		DoAndReturn(serve("the published binary"))

	b, err := DownloadBinary(context.TODO(), w, testPluginName, testOS, testARCH)
	require.NoError(t, err)
	require.Equal(t, "./my-plugin-v2", b.Cmd)
	require.Equal(t, []string{"entrypoint-v2"}, b.Entrypoints)
	require.Equal(t, p.Binaries[0].SHA512sum, b.SHA512sum)
}

// Once a binary has been checked during the job, neither the API nor the disk are hit again.
// The lookup is keyed on pluginName/os/arch, which stays stable even when a new upload
// changes the binary file name.
func TestDownloadBinary_MemoizesCheckedBinary(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	checked := &sdk.GRPCPluginBinary{
		PluginName: testPluginName,
		Name:       "my-plugin-linux-amd64-renamed",
		OS:         testOS,
		Arch:       testARCH,
		SHA512sum:  "whatever",
	}

	w := mock_workerruntime.NewMockRuntime(ctrl)
	w.EXPECT().GetCheckedPluginBinary(testPluginName, testOS, testARCH).Return(checked)
	w.EXPECT().PluginGet(gomock.Any()).Times(0)
	w.EXPECT().PluginGetBinary(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	w.EXPECT().BaseDir().Times(0)

	b, err := DownloadBinary(context.TODO(), w, testPluginName, testOS, testARCH)
	require.NoError(t, err)
	require.Equal(t, checked, b)
}
