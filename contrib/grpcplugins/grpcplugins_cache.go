package grpcplugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/klauspost/pgzip"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/spf13/afero"
)

func PerformGetCache(ctx context.Context, c *actionplugin.Common, jobCtx sdk.WorkflowRunJobsContext, cacheKey string, workDirs *sdk.WorkerDirectories, path string, failOnMiss bool) error {
	absPath := path
	if !sdk.PathIsAbs(path) {
		var err error
		absPath, err = filepath.Abs(filepath.Join(workDirs.WorkingDir, path))
		if err != nil {
			return fmt.Errorf("unable to compute absolute path: %v", err)
		}
	}

	// Check if file or directory exist
	cacheFound := false
	var err error
	if jobCtx.Integrations != nil && jobCtx.Integrations.ArtifactManager.Name != "" {
		cacheFound, err = performFromArtifactory(ctx, c, jobCtx, cacheKey, workDirs, absPath, failOnMiss)
	} else {
		cacheFound, err = performFromCDN(ctx, c, cacheKey, workDirs, absPath)
	}
	if err != nil {
		return err
	}
	out := workerruntime.OutputRequest{
		Name:  "cache-hit",
		Value: strconv.FormatBool(cacheFound),
	}
	return CreateOutput(ctx, c, out)
}

func performFromArtifactory(ctx context.Context, c *actionplugin.Common, jobCtx sdk.WorkflowRunJobsContext, cacheKey string, workDirs *sdk.WorkerDirectories, absPath string, failOnMiss bool) (bool, error) {
	downloadURI := BuildCacheURL(jobCtx.Integrations.ArtifactManager, jobCtx.CDS.ProjectKey, cacheKey)
	if downloadURI == "" {
		return false, sdk.Errorf("no downloadURI specified")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return false, err
	}
	rtToken := jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigToken)
	req.Header.Set("Authorization", "Bearer "+rtToken)

	Logf(c, "Downloading cache from %s...", downloadURI)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		if failOnMiss {
			return false, sdk.Errorf("cache not found (HTTP 404)")
		}
		Warn(c, "no cache found")
		return false, nil
	}
	if resp.StatusCode > 200 {
		return false, sdk.Errorf("unable to download cache (HTTP %d)", resp.StatusCode)
	}

	return extractCache(c, resp.Body, absPath)
}

func performFromCDN(ctx context.Context, c *actionplugin.Common, cacheKey string, workDirs *sdk.WorkerDirectories, absPath string) (bool, error) {
	items, err := GetV2CacheLink(ctx, c, cacheKey)
	if err != nil {
		return false, err
	}
	if len(items.Items) == 0 {
		Warn(c, "no cache found")
		return false, nil
	}
	if len(items.Items) != 1 {
		return false, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to get one cache with key %s. Got %d", cacheKey, len(items.Items))
	}

	cdnSig, err := GetV2CacheSignature(ctx, c, cacheKey)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/item/%s/%s/download", items.CDNHttpURL, string(items.Items[0].Type), items.Items[0].APIRefHash), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("X-CDS-WORKER-SIGNATURE", cdnSig.Signature)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 200 {
		return false, sdk.Errorf("unable to download cache (HTTP %d)", resp.StatusCode)
	}

	return extractCache(c, resp.Body, absPath)
}

const (
	// Read ahead of the decompression. The pgzip default is 4MB, too small to
	// keep the download busy while the current blocks are being written to disk.
	cacheGzipBlockSize = 1 << 20
	cacheGzipBlocks    = 16
)

// extractCache streams a cache archive into absPath, without an intermediate
// file, and reports how long the download held the extraction up. It returns
// whether the cache could be used: an archive that cannot be read is reported
// as a miss rather than as an error.
func extractCache(c *actionplugin.Common, body io.Reader, absPath string) (bool, error) {
	if err := os.MkdirAll(absPath, os.FileMode(0744)); err != nil {
		return false, fmt.Errorf("unable to create destination directory: %v", err)
	}

	// HTTP body → gzip → tar → filesystem
	countReader := &countingReader{r: body}
	gzr, err := pgzip.NewReaderN(countReader, cacheGzipBlockSize, cacheGzipBlocks)
	if err != nil {
		warnCacheUnusable(c, absPath, err)
		return false, nil
	}
	defer gzr.Close()

	t0 := time.Now()
	if err := sdk.Untar(afero.NewOsFs(), absPath, gzr); err != nil {
		warnCacheUnusable(c, absPath, err)
		return false, nil
	}
	elapsed := time.Since(t0)

	// The decompression reads ahead in its own goroutine, so downloading and
	// extracting overlap and the two cannot be told apart in wall clock time.
	// Reporting how long the reads blocked is enough to know which of the two
	// the restore is waiting on: close to the total means the network, much
	// lower means the filesystem.
	read := countReader.bytes()
	blocked := countReader.blocked()
	var throughput float64
	if elapsed > 0 {
		throughput = float64(read) / (1024 * 1024) / elapsed.Seconds()
	}
	Successf(c, "Cache restored to %s (%d bytes in %.3fs, %.3fs of which blocked on the download, %.1f MB/s).",
		absPath, read, elapsed.Seconds(), blocked.Seconds(), throughput)
	return true, nil
}

// warnCacheUnusable reports a cache that could not be read as a miss. A cache is
// optimisation and must never take a build down: whatever the reason, from an
// archive left truncated by an interrupted or concurrent upload to a full disk,
// the job can still do the work itself. A job that saves its cache on a miss
// then replaces the unusable copy, so the key repairs itself on the next run.
//
// The reason is not guessed: a truncated gzip stream, a bad checksum and a
// failure to write a file all surface the same way here, so the error is
// reported as is and the caller carries on without a cache.
func warnCacheUnusable(c *actionplugin.Common, absPath string, err error) {
	Warnf(c, "ignoring the cache, it could not be extracted to %s: %v", absPath, err)
	Warnf(c, "%s may hold part of it, continuing as if there was no cache", absPath)
}

// countingReader counts the bytes read and the time spent waiting for them. It
// is read from the decompression goroutine and reported from another one, hence
// the atomics.
type countingReader struct {
	r       io.Reader
	n       atomic.Int64
	waitedN atomic.Int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	t0 := time.Now()
	n, err := cr.r.Read(p)
	cr.waitedN.Add(int64(time.Since(t0)))
	cr.n.Add(int64(n))
	return n, err
}

func (cr *countingReader) bytes() int64           { return cr.n.Load() }
func (cr *countingReader) blocked() time.Duration { return time.Duration(cr.waitedN.Load()) }
