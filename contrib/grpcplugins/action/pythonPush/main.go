package main

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type pythonPushPlugin struct {
	actionplugin.Common
}

func main() {
	actPlugin := pythonPushPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}

func (actPlugin *pythonPushPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "pythonPush",
		Author:      "Steven GUIHEUX <steven.guiheux@corp.ovh.com>",
		Description: "Push a package on a python repository",
		Version:     sdk.VERSION,
	}, nil
}

type pythonOpts struct {
	packageName string
	version     string
	directory   string
	url         string
	username    string
	password    string
	wheel       bool
	binary      string
}

func (p *pythonPushPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	pkg := q.GetOptions()["package"]
	version := q.GetOptions()["version"]
	directory := q.GetOptions()["directory"]
	wheelString := q.GetOptions()["wheel"]
	urlRepo := q.GetOptions()["url"]
	username := q.GetOptions()["username"]
	password := q.GetOptions()["password"]
	pythonBinary := q.GetOptions()["pythonBinary"]

	if pythonBinary == "" {
		pythonBinary = "python"
	}
	if pkg == "" {
		res.Status = sdk.StatusFail
		res.Details = "'package' input must not be empty"
		return stream.Send(res)
	}
	if version == "" {
		res.Status = sdk.StatusFail
		res.Details = "'version' input must not be empty"
		return stream.Send(res)
	}
	if directory == "" {
		res.Status = sdk.StatusFail
		res.Details = "'directory' input must not be empty"
		return stream.Send(res)
	}
	wheel, err := strconv.ParseBool(wheelString)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Details = "'wheel' input must be a boolean"
		return stream.Send(res)
	}

	opts := pythonOpts{
		packageName: pkg,
		version:     version,
		directory:   directory,
		wheel:       wheel,
		binary:      pythonBinary,
	}

	var integ *sdk.JobIntegrationsContext

	// If not url provided, check integration
	if urlRepo == "" {
		jobCtx, err := grpcplugins.GetJobContext(ctx, &p.Common)
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = "unable to retrieve job context"
			return stream.Send(res)
		}
		if jobCtx == nil || jobCtx.Integrations == nil || jobCtx.Integrations.ArtifactManager.Name == "" {
			res.Status = sdk.StatusFail
			res.Details = "unable to upload package, no integration found on the current job"
			return stream.Send(res)
		}
		integ = &jobCtx.Integrations.ArtifactManager
		completeURL := fmt.Sprintf("%sapi/pypi/%s-pypi", integ.Get(sdk.ArtifactoryConfigURL), integ.Get(sdk.ArtifactoryConfigRepositoryPrefix))
		opts.url = completeURL
		opts.username = integ.Get(sdk.ArtifactoryConfigTokenName)
		opts.password = integ.Get(sdk.ArtifactoryConfigToken)
	} else {
		opts.url = urlRepo
		if username == "" {
			res.Status = sdk.StatusFail
			res.Details = "'username' input must not be empty"
			return stream.Send(res)
		}
		if password == "" {
			res.Status = sdk.StatusFail
			res.Details = "'password' input must not be empty"
			return stream.Send(res)
		}
		opts.username = username
		opts.password = password
	}

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Details = fmt.Sprintf("unable to get working directory: %v", err)
		return stream.Send(res)
	}

	if err := p.perform(ctx, workDirs.WorkingDir, opts, integ); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}
	return stream.Send(res)
}

// Run implements actionplugin.ActionPluginServer.
func (actPlugin *pythonPushPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (actPlugin *pythonPushPlugin) perform(ctx context.Context, workerWorkspaceDir string, opts pythonOpts, integ *sdk.JobIntegrationsContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
			fmt.Println(string(debug.Stack()))
			err = errors.Errorf("Internal server error: panic")
		}
	}()

	grpcplugins.Logf(&actPlugin.Common, "Pushing %s on version %s", opts.packageName, opts.version)

	var runResultRequest = workerruntime.V2RunResultRequest{
		RunResult: &sdk.V2WorkflowRunResult{
			IssuedAt: time.Now(),
			Type:     sdk.V2WorkflowRunResultTypePython,
			Status:   sdk.V2WorkflowRunResultStatusPending,
			Detail: sdk.V2WorkflowRunResultDetail{
				Data: sdk.V2WorkflowRunResultPythonDetail{
					Name:      opts.packageName,
					Version:   opts.version,
					Extension: "tar.gz",
				},
			},
		},
	}
	result, err := grpcplugins.CreateRunResult(ctx, &actPlugin.Common, &runResultRequest)
	if err != nil {
		return err
	}

	pullScript := fmt.Sprintf(`#!/bin/bash
# write .pypirc file
cat <<EOF >> ${HOME}/.pypirc
[distutils]
index-servers = artifactory
[artifactory]
repository: %s
username: %s
password: %s
EOF

pythonBinary="%s"
if [[ -e venv/bin/python ]]; then
	pythonBinary="venv/bin/python"
fi
`, opts.url, opts.username, opts.password, opts.binary)
	if opts.wheel {
		pullScript += "$pythonBinary setup.py sdist bdist_wheel upload -r artifactory"
	} else {
		pullScript += "$pythonBinary setup.py sdist upload -r artifactory"
	}

	chanRes := make(chan *actionplugin.ActionResult)
	goRoutines := sdk.NewGoRoutines(ctx)

	scriptWorkDir := opts.directory
	if !strings.HasPrefix(scriptWorkDir, "/") {
		scriptWorkDir = strings.TrimSuffix(workerWorkspaceDir, "/") + "/" + opts.directory
	}

	goRoutines.Exec(ctx, "runActionPythonPushPlugin-runScript", func(ctx context.Context) {
		if err := grpcplugins.RunScript(ctx, &actPlugin.Common, chanRes, scriptWorkDir, pullScript); err != nil {
			grpcplugins.Errorf(&actPlugin.Common, "%+v\n", err)
		}
	})

	select {
	case <-ctx.Done():
		grpcplugins.Errorf(&actPlugin.Common, "CDS Worker execution canceled: %v", ctx.Err())
		return errors.New("CDS Worker execution canceled")
	case res := <-chanRes:
		if res.Status != sdk.StatusSuccess {
			return errors.New(res.Details)
		}
	}

	if integ != nil {
		repository := integ.Get(sdk.ArtifactoryConfigRepositoryPrefix) + "-pypi"
		maturity := integ.Get(sdk.ArtifactoryConfigPromotionLowMaturity)
		localRepository := repository + "-" + maturity

		rtConfig := grpcplugins.ArtifactoryConfig{
			URL:   integ.Get(sdk.ArtifactoryConfigURL),
			Token: integ.Get(sdk.ArtifactoryConfigToken),
		}
		if !strings.HasSuffix(rtConfig.URL, "/") {
			rtConfig.URL = rtConfig.URL + "/"
		}

		folderInfo, err := grpcplugins.GetArtifactoryFolderInfo(ctx, &actPlugin.Common, rtConfig, repository, filepath.Join(opts.packageName, opts.version))
		if err != nil {
			return err
		}
		for _, c := range folderInfo.Children {
			if c.Folder {
				continue
			}
			fi, err := grpcplugins.GetArtifactoryFileInfo(context.TODO(), &actPlugin.Common, rtConfig, repository, filepath.Join(opts.packageName, opts.version, strings.TrimPrefix(c.URI, "/")))
			if err != nil {
				return fmt.Errorf("unable to get Artifactory file info %s: %v", c.URI, err)
			}
			grpcplugins.Logf(&actPlugin.Common, "Get info ok for %s", c.URI)

			// Python can upload a tar.gz file + a wheel file
			var runResult *sdk.V2WorkflowRunResult
			if c.URI == fmt.Sprintf("/%s-%s.%s", opts.packageName, opts.version, "tar.gz") {
				runResult = result.RunResult
				_, fileName := filepath.Split(fi.Path)
				grpcplugins.ExtractFileInfoIntoRunResult(runResult, *fi, fileName, "pypi", localRepository, repository, maturity)
			} else {
				// Create a new run result
				runResult = &sdk.V2WorkflowRunResult{
					IssuedAt:                       time.Now(),
					Type:                           sdk.V2WorkflowRunResultTypePython,
					Status:                         sdk.V2WorkflowRunResultStatusPending,
					ArtifactManagerIntegrationName: &integ.Name,
					Detail:                         grpcplugins.ComputeRunResultPythonDetail(strings.TrimPrefix(c.URI, "/"), opts.version, strings.TrimPrefix(filepath.Ext(c.URI), ".")),
				}
				grpcplugins.ExtractFileInfoIntoRunResult(runResult, *fi, strings.TrimPrefix(c.URI, "/"), "pypi", localRepository, repository, maturity)
			}
			runResult.Status = sdk.V2WorkflowRunResultStatusCompleted
			var runResultRequest = workerruntime.V2RunResultRequest{
				RunResult: runResult,
			}

			if c.URI == fmt.Sprintf("/%s-%s.%s", opts.packageName, opts.version, "tar.gz") {
				if _, err := grpcplugins.UpdateRunResult(ctx, &actPlugin.Common, &runResultRequest); err != nil {
					return err
				}
			} else {
				if _, err := grpcplugins.CreateRunResult(ctx, &actPlugin.Common, &runResultRequest); err != nil {
					return err
				}
			}
		}
	} else {
		result.RunResult.Status = sdk.V2WorkflowRunResultStatusCompleted
		runResultRequest := workerruntime.V2RunResultRequest{
			RunResult: result.RunResult,
		}
		if _, err := grpcplugins.UpdateRunResult(ctx, &actPlugin.Common, &runResultRequest); err != nil {
			return err
		}
	}

	return nil
}
