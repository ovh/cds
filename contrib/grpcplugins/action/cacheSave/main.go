package main

import (
	"archive/tar"
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/pgzip"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type cacheSavePlugin struct {
	actionplugin.Common
}

func (actPlugin *cacheSavePlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-cacheSave",
		Author:      "Steven GUIHEUX <steven.guiheux@ovhcloud.com>",
		Description: `This action allow you save a cache`,
		Version:     sdk.VERSION,
	}, nil
}

func (p *cacheSavePlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	cacheKey := q.GetOptions()["key"]
	path := q.GetOptions()["path"]

	jobCtx, err := grpcplugins.GetJobContext(ctx, &p.Common)
	if err != nil {
		err := fmt.Errorf("unable to retrieve job context: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
	if err != nil {
		err := fmt.Errorf("unable to get working directory: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	if err := p.perform(ctx, *jobCtx, cacheKey, workDirs, path); err != nil {
		err := fmt.Errorf("unable to save cache: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	return stream.Send(res)
}

func (p *cacheSavePlugin) perform(ctx context.Context, jobCtx sdk.WorkflowRunJobsContext, cacheKey string, workDirs *sdk.WorkerDirectories, path string) error {
	fullPath := path
	if !sdk.PathIsAbs(path) {
		var err error
		fullPath, err = filepath.Abs(filepath.Join(workDirs.WorkingDir, path))
		if err != nil {
			return fmt.Errorf("unable to compute absolute path: %v", err)
		}
	}

	itemsToArchive := make([]string, 0)
	// Check directory
	if fileInfo, err := os.Stat(fullPath); err == nil && fileInfo.IsDir() {
		itemsToArchive = append(itemsToArchive, fullPath)
	} else {
		// Try to manage files
		results, err := glob.Glob(workDirs.WorkingDir, fullPath)
		if err != nil {
			return err
		}
		for _, r := range results.Results {
			resultPath := filepath.Join(fmt.Sprintf("%s", results.DirFS), r.Path)
			itemsToArchive = append(itemsToArchive, resultPath)
			grpcplugins.Success(&p.Common, resultPath+" will be cached")
		}
	}

	if len(itemsToArchive) == 0 {
		return sdk.Errorf("there is nothing to cache")
	}

	archivePath := workDirs.WorkingDir + "/cache.tar.gz"

	// Check if a file cache.tar.gz already exist and remove it
	if _, err := os.Stat(archivePath); err == nil {
		if err := os.Remove(archivePath); err != nil {
			return sdk.Errorf("unable to remove previous cache: %s: %v", archivePath, err)
		}
	}

	t0 := time.Now()
	if err := createTarGz(itemsToArchive, archivePath); err != nil {
		return fmt.Errorf("unable to create cache archive: %v", err)
	}
	grpcplugins.Logf(&p.Common, "Cache archive created in %.3fs", time.Since(t0).Seconds())

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("unable to open cache archive: %v", err)
	}
	defer f.Close()

	// Check if file or directory exist
	if jobCtx.Integrations != nil && jobCtx.Integrations.ArtifactManager.Name != "" {
		return p.performFromArtifactory(ctx, jobCtx, cacheKey, f)
	} else {
		return p.performFromCDN(ctx, cacheKey, f)
	}
}

func (p *cacheSavePlugin) performFromArtifactory(ctx context.Context, jobCtx sdk.WorkflowRunJobsContext, cacheKey string, reader io.ReadSeeker) error {
	uploadURL := grpcplugins.BuildCacheURL(jobCtx.Integrations.ArtifactManager, jobCtx.CDS.ProjectKey, cacheKey)

	_, d, err := grpcplugins.ArtifactoryItemUpload(ctx, &p.Common, jobCtx.Integrations.ArtifactManager, reader, map[string]string{}, uploadURL)
	if err != nil {
		return err
	}

	grpcplugins.Successf(&p.Common, "Cache uploaded in %.3fs", d.Seconds())
	return nil
}

func (p *cacheSavePlugin) performFromCDN(ctx context.Context, cacheKey string, reader io.ReadSeeker) error {
	sign, err := grpcplugins.GetV2CacheSignature(ctx, &p.Common, cacheKey)
	if err != nil {
		return err
	}

	_, d, err := grpcplugins.CDNItemUpload(ctx, &p.Common, sign.CDNAddress, sign.Signature, reader)
	if err != nil {
		return err
	}
	grpcplugins.Successf(&p.Common, "Cache uploaded in %.3fs", d.Seconds())
	return nil
}

func (actPlugin *cacheSavePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

// createTarGz creates a tar.gz archive using parallel gzip compression.
func createTarGz(sources []string, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	// Buffer disk writes to reduce syscalls
	bw := bufio.NewWriterSize(f, 1<<20) // 1MB buffer
	defer bw.Flush()

	gzw, err := pgzip.NewWriterLevel(bw, gzip.BestSpeed)
	if err != nil {
		return err
	}
	defer gzw.Close()

	// Use 1MB blocks for better parallelization across all CPUs
	if err := gzw.SetConcurrency(1<<20, runtime.GOMAXPROCS(0)); err != nil {
		return err
	}

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	copyBuf := make([]byte, 256*1024) // reusable 256KB copy buffer

	for _, source := range sources {
		info, err := os.Lstat(source)
		if err != nil {
			return fmt.Errorf("unable to stat %s: %v", source, err)
		}

		var baseDir string
		if info.IsDir() {
			baseDir = filepath.Dir(source)
		} else {
			baseDir = filepath.Dir(source)
		}

		err = filepath.Walk(source, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Use Lstat to detect symlinks (Walk follows them)
			lfi, err := os.Lstat(path)
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(lfi, "")
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}
			header.Name = relPath

			if lfi.Mode()&os.ModeSymlink != 0 {
				link, err := os.Readlink(path)
				if err != nil {
					return err
				}
				header.Linkname = link
				header.Typeflag = tar.TypeSymlink
				return tw.WriteHeader(header)
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !lfi.Mode().IsRegular() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.CopyBuffer(tw, file, copyBuf)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	actPlugin := cacheSavePlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}
