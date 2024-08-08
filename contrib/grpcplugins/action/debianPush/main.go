package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"

	"google.golang.org/protobuf/types/known/emptypb"
)

type debianPushPlugin struct {
	actionplugin.Common
}

func main() {
	actPlugin := debianPushPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}

// Manifest implements actionplugin.ActionPluginServer.
func (*debianPushPlugin) Manifest(context.Context, *emptypb.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "debianPush",
		Author:      "Steven GUIHEUX <steven.guiheux@corp.ovh.com>",
		Description: "Push debian package on a repository",
		Version:     sdk.VERSION,
	}, nil
}

type debianPushOptions struct {
	architectures       []string
	components          []string
	distributions       []string
	files               []string
	repositoryURL       string
	label               string
	origin              string
	jobContext          sdk.WorkflowRunJobsContext
	integRepositoryName string
}

func (p *debianPushPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}
	opts := debianPushOptions{}

	// Check inputs
	arch := q.GetOptions()["architectures"]
	compo := q.GetOptions()["components"]
	distr := q.GetOptions()["distributions"]
	files := q.GetOptions()["files"]
	opts.label = q.GetOptions()["label"]
	opts.origin = q.GetOptions()["origin"]

	if arch == "" {
		res.Status = sdk.StatusFail
		res.Details = "'architectures' input cannot be empty"
		return stream.Send(res)
	} else {
		opts.architectures = strings.Split(arch, " ")
	}

	if compo == "" {
		res.Status = sdk.StatusFail
		res.Details = "'components' input cannot be empty"
		return stream.Send(res)
	} else {
		opts.components = strings.Split(compo, " ")
	}
	if distr == "" {
		res.Status = sdk.StatusFail
		res.Details = "'distributions' input cannot be empty"
		return stream.Send(res)
	} else {
		opts.distributions = strings.Split(distr, " ")
	}
	if files == "" {
		res.Status = sdk.StatusFail
		res.Details = "'files' input cannot be empty"
		return stream.Send(res)
	} else {
		opts.files = strings.Split(files, " ")
	}

	jobContext, err := grpcplugins.GetJobContext(ctx, &p.Common)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Details = fmt.Sprintf("Unable to retrieve job integration: %v", err)
		return stream.Send(res)
	}
	if jobContext == nil || jobContext.Integrations == nil || jobContext.Integrations.ArtifactManager.Name == "" {
		res.Status = sdk.StatusFail
		res.Details = "Unable to retrieve artifact manager integration for the current job"
		return stream.Send(res)
	}
	opts.jobContext = *jobContext
	url := opts.jobContext.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigURL)
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	opts.integRepositoryName = jobContext.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigRepositoryPrefix) + "-debian"
	url += opts.integRepositoryName
	opts.repositoryURL = url

	grpcplugins.Logf(&p.Common, "  Debian repository URL: %s", opts.repositoryURL)

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Details = fmt.Sprintf("unable to get working directory: %v", err)
		return stream.Send(res)
	}

	dirFS := os.DirFS(workDirs.WorkingDir)

	if err := p.perform(ctx, dirFS, opts); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}
	return stream.Send(res)

}

// Run implements actionplugin.ActionPluginServer.
func (p *debianPushPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (p *debianPushPlugin) perform(ctx context.Context, dirFS fs.FS, opts debianPushOptions) error {
	for _, pattern := range opts.files {
		results, sizes, _, openFiles, checksums, err := grpcplugins.RetrieveFilesToUpload(ctx, &p.Common, dirFS, pattern, "ERROR")
		if err != nil {
			return err
		}
		for _, r := range results {
			message := fmt.Sprintf("\nStarting upload of file %q as %q \n  Size: %d, MD5: %s, sha1: %s, SHA256: %s", r.Path, r.Result, sizes[r.Path], checksums[r.Path].Md5, checksums[r.Path].Sha1, checksums[r.Path].Sha256)
			grpcplugins.Log(&p.Common, message)

			// Create run result at status "pending"
			var runResultRequest = workerruntime.V2RunResultRequest{
				RunResult: &sdk.V2WorkflowRunResult{
					IssuedAt: time.Now(),
					Type:     sdk.V2WorkflowRunResultTypeDebian,
					Status:   sdk.V2WorkflowRunResultStatusPending,
					Detail: sdk.V2WorkflowRunResultDetail{
						Data: sdk.V2WorkflowRunResultDebianDetail{
							Name:          r.Result,
							Size:          sizes[r.Path],
							MD5:           checksums[r.Path].Md5,
							SHA1:          checksums[r.Path].Sha1,
							SHA256:        checksums[r.Path].Sha256,
							Components:    opts.components,
							Distributions: opts.distributions,
							Architectures: opts.architectures,
						},
					},
				},
			}

			if _, err := p.UploadArtifactoryDebianPackage(ctx, opts, &runResultRequest, r.Result, openFiles[r.Path], sizes[r.Path], checksums[r.Path]); err != nil {
				_ = openFiles[r.Path].Close()
				return err
			}
			_ = openFiles[r.Path].Close()
		}
	}
	return nil
}

func (p *debianPushPlugin) UploadArtifactoryDebianPackage(ctx context.Context, opts debianPushOptions, runresultReq *workerruntime.V2RunResultRequest, fileName string, f fs.File, size int64, fileChecksum grpcplugins.ChecksumResult) (*workerruntime.V2UpdateResultResponse, error) {
	response, err := grpcplugins.CreateRunResult(ctx, &p.Common, runresultReq)
	if err != nil {
		return nil, err
	}

	// Upload the file to an artifactory or CDN
	var d time.Duration
	var runResultRequest workerruntime.V2RunResultRequest

	var distribLayout, componentLayout, archLayout string
	for _, d := range opts.distributions {
		distribLayout += ";deb.distribution=" + d

	}
	for _, c := range opts.components {
		componentLayout += ";deb.component=" + c

	}
	for _, a := range opts.architectures {
		archLayout += ";deb.architecture=" + a

	}

	debInfo := fmt.Sprintf("%s;%s;%s", distribLayout, componentLayout, archLayout)
	cdsInfo := fmt.Sprintf("cds_version=%s;cds_workflow=%s", opts.jobContext.Git.SemverCurrent, opts.jobContext.CDS.Workflow)
	maturity := opts.jobContext.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigPromotionLowMaturity)
	path := fmt.Sprintf("/pool/%s;%s;%s;deb.release.origin=%s;deb.release.label=%s",
		fileName, debInfo, cdsInfo, opts.origin, opts.label)

	response.RunResult.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{}
	response.RunResult.ArtifactManagerMetadata.Set("repository", opts.integRepositoryName) // This is the virtual repository
	response.RunResult.ArtifactManagerMetadata.Set("type", "debian")
	response.RunResult.ArtifactManagerMetadata.Set("maturity", maturity)
	response.RunResult.ArtifactManagerMetadata.Set("name", fileName)
	response.RunResult.ArtifactManagerMetadata.Set("path", path)
	response.RunResult.ArtifactManagerMetadata.Set("md5", fileChecksum.Md5)
	response.RunResult.ArtifactManagerMetadata.Set("sha1", fileChecksum.Sha1)
	response.RunResult.ArtifactManagerMetadata.Set("sha256", fileChecksum.Sha256)

	reader, ok := f.(io.ReadSeeker)
	if !ok {
		// unable to cast the file
		return nil, fmt.Errorf("unable to cast reader")
	}

	var res *grpcplugins.ArtifactoryUploadResult
	res, d, err = grpcplugins.ArtifactoryItemUploadRunResult(ctx, &p.Common, response.RunResult, opts.jobContext.Integrations.ArtifactManager, reader)
	if err != nil {
		grpcplugins.Error(&p.Common, err.Error())
		return nil, err
	}

	response.RunResult.ArtifactManagerMetadata.Set("uri", res.URI)
	response.RunResult.ArtifactManagerMetadata.Set("mimeType", res.MimeType)
	response.RunResult.ArtifactManagerMetadata.Set("downloadURI", res.DownloadURI)
	response.RunResult.ArtifactManagerMetadata.Set("createdBy", res.CreatedBy)
	response.RunResult.ArtifactManagerMetadata.Set("localRepository", res.Repo) // This contains the localrepository
	response.RunResult.ArtifactManagerMetadata.Set("path", res.Path)
	response.RunResult.ArtifactManagerMetadata.Set("name", filepath.Base(res.Path))
	runResultRequest = workerruntime.V2RunResultRequest{RunResult: response.RunResult}

	// Update run result
	runResultRequest.RunResult.Status = sdk.V2WorkflowRunResultStatusCompleted
	updateResponse, err := grpcplugins.UpdateRunResult(ctx, &p.Common, &runResultRequest)
	if err != nil {
		grpcplugins.Error(&p.Common, err.Error())
		return nil, err
	}

	grpcplugins.Successf(&p.Common, "  %d bytes uploaded in %.3fs", size, d.Seconds())

	if _, err := updateResponse.RunResult.GetDetail(); err != nil {
		grpcplugins.Error(&p.Common, err.Error())
		return nil, err
	}
	grpcplugins.Logf(&p.Common, "  Result %s (%s) created", updateResponse.RunResult.Name(), updateResponse.RunResult.ID)
	return updateResponse, nil
}
