package grpcplugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

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

	if err := os.MkdirAll(absPath, os.FileMode(0744)); err != nil {
		return false, fmt.Errorf("unable to create destination directory: %v", err)
	}

	// Stream directly: HTTP body → gzip → tar → filesystem (no intermediate file)
	countReader := &countingReader{r: resp.Body}
	t0 := time.Now()
	if err := sdk.UntarGz(afero.NewOsFs(), absPath, countReader); err != nil {
		return false, fmt.Errorf("unable to extract cache: %v", err)
	}
	elapsed := time.Since(t0)

	Successf(c, "Cache restored to %s (%d bytes downloaded and extracted in %.3f seconds).", absPath, countReader.n, elapsed.Seconds())
	return true, nil
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

	if err := os.MkdirAll(absPath, os.FileMode(0744)); err != nil {
		return false, fmt.Errorf("unable to create destination directory: %v", err)
	}

	// Stream directly: HTTP body → gzip → tar → filesystem (no intermediate file)
	countReader := &countingReader{r: resp.Body}
	t0 := time.Now()
	if err := sdk.UntarGz(afero.NewOsFs(), absPath, countReader); err != nil {
		return false, fmt.Errorf("unable to extract cache: %v", err)
	}
	elapsed := time.Since(t0)

	Successf(c, "Cache restored to %s (%d bytes downloaded and extracted in %.3f seconds).", absPath, countReader.n, elapsed.Seconds())
	return true, nil
}

// countingReader wraps an io.Reader and counts bytes read.
type countingReader struct {
	r io.Reader
	n int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}
