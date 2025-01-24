package grpcplugins

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		cacheFound, err = performFromArtifactory(ctx, c, jobCtx, cacheKey, workDirs, path, failOnMiss)
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
	if err := Untar(absPath, destinationTarFile); err != nil {
		return false, err
	}
	if err := afero.NewOsFs().Remove(destinationTarFile); err != nil {
		return false, fmt.Errorf("unable to remove archive cache file %v", err)
	}
	Successf(c, "Cache was downloaded to %s (%d bytes downloaded in %.3f seconds).", absPath, n, time.Since(t0).Seconds())
	return true, nil
}

func performFromCDN(ctx context.Context, c *actionplugin.Common, cacheKey string, workDirs *sdk.WorkerDirectories, absPath string) (bool, error) {
	t0 := time.Now()
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

	destinationTarFile, n, err := DownloadFromCDN(ctx, c, cdnSig.Signature, *workDirs, items.Items[0].APIRefHash, string(items.Items[0].Type), items.CDNHttpURL, absPath, "cache.tar.gz", os.FileMode(0755))
	if err != nil {
		return false, err
	}

	if err := Untar(absPath, destinationTarFile); err != nil {
		return false, err
	}
	if err := afero.NewOsFs().Remove(destinationTarFile); err != nil {
		return false, fmt.Errorf("unable to remove archive cache file %v", err)
	}
	Successf(c, "Cache was downloaded to %s (%d bytes downloaded in %.3f seconds).", absPath, n, time.Since(t0).Seconds())
	return true, nil
}

func Untar(dst string, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	gzr, err := gzip.NewReader(bufio.NewReader(file))
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil
		// return any other error
		case err != nil:
			return err
		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)
		// check the file type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
}
