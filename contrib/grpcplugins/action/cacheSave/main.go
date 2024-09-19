package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mholt/archiver/v3"

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
		err := fmt.Errorf("unable to retrieve cache: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	return stream.Send(res)
}

func (p *cacheSavePlugin) perform(ctx context.Context, jobCtx sdk.WorkflowRunJobsContext, cacheKey string, workDirs *sdk.WorkerDirectories, path string) error {
	itemsToArchive := make([]string, 0)
	// Check directory
	dirFS := os.DirFS(workDirs.WorkingDir)
	fullPath := workDirs.WorkingDir + "/" + path
	if fileInfo, err := os.Stat(fullPath); err == nil && fileInfo.IsDir() {
		itemsToArchive = append(itemsToArchive, workDirs.WorkingDir+"/"+path)
	} else {
		// Try to manage files
		results, err := glob.Glob(workDirs.WorkingDir, path)
		if err != nil {
			return err
		}
		for _, r := range results.Results {
			itemsToArchive = append(itemsToArchive, r.Path)
			grpcplugins.Success(&p.Common, r.Path+" will be cached")
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
	if err := archiver.Archive(itemsToArchive, archivePath); err != nil {
		return fmt.Errorf("unable to create cache archive: %v", err)
	}

	f, err := dirFS.Open("cache.tar.gz")
	if err != nil {
		return fmt.Errorf("unable to open cache archive: %v", err)
	}
	reader, ok := f.(io.ReadSeeker)
	if !ok {
		// unable to cast the file
		return fmt.Errorf("unable to cast reader")
	}

	// Check if file or directory exist
	if jobCtx.Integrations != nil && jobCtx.Integrations.ArtifactManager.Name != "" {
		return p.performFromArtifactory(ctx, jobCtx, cacheKey, reader)
	} else {
		return p.performFromCDN(ctx, cacheKey, reader)
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

func main() {
	actPlugin := cacheSavePlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}
