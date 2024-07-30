package grpcplugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/archiver/v3"
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
		cacheFound, err = performFromArtifactory(ctx, c, jobCtx, cacheKey, workDirs, path, failOnMiss)
	} else {
		cacheFound, err = performFromCDN(c, jobCtx, cacheKey, workDirs, absPath)
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
	t0 := time.Now()
	downloadURI := BuildCacheURL(jobCtx.Integrations.ArtifactManager, jobCtx.CDS.ProjectKey, cacheKey)
	destinationTarFile, n, err := DownloadFromArtifactory(ctx, c, jobCtx.Integrations.ArtifactManager, *workDirs, absPath, "cache.tar.gz", os.FileMode(0755), downloadURI)
	if err != nil {
		if !strings.Contains(err.Error(), "(HTTP 404)") || failOnMiss {
			return false, err
		}
		Warn(c, "no cache found")
		return false, nil
	}
	if err := archiver.Unarchive(destinationTarFile, absPath); err != nil {
		return false, err
	}
	if err := afero.NewOsFs().Remove(destinationTarFile); err != nil {
		return false, fmt.Errorf("unable to remove archive cache file %v", err)
	}
	Successf(c, "Cache was downloaded to %s (%d bytes downloaded in %.3f seconds).", absPath, n, time.Since(t0).Seconds())
	return true, nil
}

func performFromCDN(c *actionplugin.Common, jobCtx sdk.WorkflowRunJobsContext, cacheKey string, workDirs *sdk.WorkerDirectories, absPath string) (bool, error) {
	return false, sdk.ErrNotImplemented
}
