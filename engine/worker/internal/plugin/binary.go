package plugin

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

const (
	binaryDownloadRetry = 20
	// a checksum mismatch is only deterministic as long as the published descriptor does not
	// change: retrying more than once with the very same checksum only delays a failure that
	// will not resolve itself
	binaryChecksumRetry = 2
)

// binaryDownloadRetryDelay is a variable so that tests do not have to wait for it
var binaryDownloadRetryDelay = 3 * time.Second

// DownloadBinary makes sure the plugin binary is available in the worker basedir and that
// its content matches the checksum currently published by the API. It returns the validated
// binary descriptor, which callers must use to start the plugin: it is read from the API and
// may be more recent than the one carried by the job payload.
func DownloadBinary(ctx context.Context, w workerruntime.Runtime, pluginName, currentOS, currentARCH string) (*sdk.GRPCPluginBinary, error) {
	if b := w.GetCheckedPluginBinary(pluginName, currentOS, currentARCH); b != nil {
		return b, nil
	}

	// The descriptor carried by the job payload can be older than the binary actually served
	// by the API: read the checksum from the same source as the binary itself.
	b, err := readBinaryDescriptor(w, pluginName, currentOS, currentARCH)
	if err != nil {
		return nil, err
	}

	if _, err := w.BaseDir().Stat(b.Name); err == nil {
		errCache := checkBinaryChecksum(w.BaseDir(), b)
		if errCache == nil {
			log.Debug(ctx, "plugin %q: binary %q is in cache", pluginName, b.Name)
			w.SetCheckedPluginBinary(b)
			return b, nil
		}
		log.Warn(ctx, "plugin %q: cached binary %q rejected (%v), downloading it again", pluginName, b.Name, errCache)
		if err := w.BaseDir().Remove(b.Name); err != nil {
			return nil, sdk.WrapError(err, "unable to remove invalid plugin binary %q", b.Name)
		}
	}

	var lastErr error
	var checksumFailures int
	for retry := 0; retry < binaryDownloadRetry; retry++ {
		if retry > 0 {
			time.Sleep(binaryDownloadRetryDelay)

			// A binary can be replaced while the job runs: its checksum, and the name it is
			// published under, are only valid for the content served when the descriptor was read.
			// Reading the descriptor again is what makes a download that raced with an upload
			// recoverable, instead of failing on a checksum that will never match again.
			previous := b
			refreshed, err := readBinaryDescriptor(w, pluginName, currentOS, currentARCH)
			if err != nil {
				log.Warn(ctx, "plugin %q: unable to read the descriptor of binary %q again (try %d): %v", pluginName, b.Name, retry+1, err)
			} else {
				b = refreshed
				if !strings.EqualFold(b.SHA512sum, previous.SHA512sum) {
					log.Info(ctx, "plugin %q: binary %q has been replaced, retrying with the checksum now published", pluginName, b.Name)
					checksumFailures = 0
				}
			}
		}

		lastErr = downloadBinary(w, b)
		if lastErr == nil {
			log.Info(ctx, "plugin %q: binary %q successfully downloaded and checked", pluginName, b.Name)
			w.SetCheckedPluginBinary(b)
			return b, nil
		}

		log.Warn(ctx, "plugin %q: unable to get binary %q (try %d): %v", pluginName, b.Name, retry+1, lastErr)

		if sdk.ErrorIs(lastErr, sdk.ErrPluginInvalid) {
			checksumFailures++
			if checksumFailures >= binaryChecksumRetry {
				return nil, lastErr
			}
		}
	}

	return nil, sdk.NewErrorFrom(sdk.ErrPluginInvalid, "unable to get a valid binary %q for plugin %q after %d tries: %v", b.Name, pluginName, binaryDownloadRetry, lastErr)
}

// readBinaryDescriptor reads from the API the binary currently published for the given os and arch.
func readBinaryDescriptor(w workerruntime.Runtime, pluginName, currentOS, currentARCH string) (*sdk.GRPCPluginBinary, error) {
	p, err := w.PluginGet(pluginName)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get plugin %q", pluginName)
	}
	b := p.GetBinary(currentOS, currentARCH)
	if b == nil {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find plugin %q for %s/%s", pluginName, currentOS, currentARCH)
	}
	if b.SHA512sum == "" {
		return nil, sdk.NewErrorFrom(sdk.ErrPluginInvalid, "plugin %q: binary %q has no published sha512sum, refusing to run it", pluginName, b.Name)
	}
	return b, nil
}

// downloadBinary writes the plugin binary in the worker basedir and checks its integrity
// while it is streamed, so that the content is never hashed from a second read.
func downloadBinary(w workerruntime.Runtime, b *sdk.GRPCPluginBinary) error {
	fi, err := w.BaseDir().OpenFile(b.Name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(b.Perm))
	if err != nil {
		return sdk.WrapError(err, "unable to create the file %q", b.Name)
	}

	sha512Hash := sha512.New()
	if err := w.PluginGetBinary(b.PluginName, b.OS, b.Arch, io.MultiWriter(fi, sha512Hash)); err != nil {
		_ = fi.Close()
		return sdk.WrapError(err, "unable to download the binary plugin %q", b.PluginName)
	}
	if err := fi.Close(); err != nil {
		return sdk.WrapError(err, "unable to close the file %q", b.Name)
	}

	return compareChecksum(b, sha512Hash)
}

// checkBinaryChecksum hashes a binary already present in the worker basedir.
func checkBinaryChecksum(fs afero.Fs, b *sdk.GRPCPluginBinary) error {
	f, err := fs.Open(b.Name)
	if err != nil {
		return sdk.WrapError(err, "unable to open the file %q", b.Name)
	}
	defer f.Close() // nolint

	sha512Hash := sha512.New()
	if _, err := io.Copy(sha512Hash, f); err != nil {
		return sdk.WrapError(err, "unable to read the file %q", b.Name)
	}

	return compareChecksum(b, sha512Hash)
}

func compareChecksum(b *sdk.GRPCPluginBinary, sha512Hash hash.Hash) error {
	sum := hex.EncodeToString(sha512Hash.Sum(nil))
	if !strings.EqualFold(sum, b.SHA512sum) {
		return sdk.NewErrorFrom(sdk.ErrPluginInvalid, "sha512 mismatch for binary %q: expected %s, got %s", b.Name, b.SHA512sum, sum)
	}
	return nil
}
