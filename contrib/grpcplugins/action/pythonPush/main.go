package main

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

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
}

// Run implements actionplugin.ActionPluginServer.
func (actPlugin *pythonPushPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	res := &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}

	pkg := q.GetOptions()["package"]
	version := q.GetOptions()["version"]
	directory := q.GetOptions()["directory"]
	wheelString := q.GetOptions()["wheel"]
	urlRepo := q.GetOptions()["url"]
	username := q.GetOptions()["username"]
	password := q.GetOptions()["password"]

	if pkg == "" {
		res.Status = sdk.StatusFail
		res.Details = "'package' input must not be empty"
		return res, nil
	}
	if version == "" {
		res.Status = sdk.StatusFail
		res.Details = "'version' input must not be empty"
		return res, nil
	}
	if directory == "" {
		res.Status = sdk.StatusFail
		res.Details = "'directory' input must not be empty"
		return res, nil
	}
	wheel, err := strconv.ParseBool(wheelString)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Details = "'wheel' input must be a boolean"
		return res, nil
	}

	opts := pythonOpts{
		packageName: pkg,
		version:     version,
		directory:   directory,
		wheel:       wheel,
	}

	var integ *sdk.ProjectIntegration

	// If not url provided, check integration
	if urlRepo == "" {
		jobCtx, err := grpcplugins.GetJobContext(ctx, &actPlugin.Common)
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = "'wheel' input must be a boolean"
			return res, nil
		}
		if jobCtx == nil || jobCtx.Integrations == nil || jobCtx.Integrations.ArtifactManager == "" {
			res.Status = sdk.StatusFail
			res.Details = "unable to upload package, no integration found on the current job"
			return res, nil
		}
		integ, err = grpcplugins.GetIntegrationByName(ctx, &actPlugin.Common, jobCtx.Integrations.ArtifactManager)
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to get integration %s", jobCtx.Integrations.ArtifactManager)
			return res, nil
		}
		completeURL := fmt.Sprintf("%sapi/pypi/%s", integ.Config[sdk.ArtifactoryConfigURL].Value, integ.Config[sdk.ArtifactoryConfigRepositoryPrefix].Value+"-pypi")
		opts.url = completeURL
		opts.username = integ.Config[sdk.ArtifactoryConfigTokenName].Value
		opts.password = integ.Config[sdk.ArtifactoryConfigToken].Value
	} else {
		opts.url = urlRepo
		if username == "" {
			res.Status = sdk.StatusFail
			res.Details = "'username' input must not be empty"
			return res, nil
		}
		if password == "" {
			res.Status = sdk.StatusFail
			res.Details = "'password' input must not be empty"
			return res, nil
		}
		opts.username = username
		opts.password = password
	}

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &actPlugin.Common)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Details = fmt.Sprintf("unable to get working directory: %v", err)
		return res, nil
	}

	if err := actPlugin.perform(ctx, workDirs.WorkingDir, opts, integ); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return res, err
	}
	return res, nil
}

func (actPlugin *pythonPushPlugin) perform(ctx context.Context, workerWorkspaceDir string, opts pythonOpts, integ *sdk.ProjectIntegration) error {
	grpcplugins.Logf("Pushing %s on version %s", opts.packageName, opts.version)

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
cat <<EOF >> .pypirc
[distutils]
index-servers = artifactory
[artifactory]
repository: %s
username: %s
password: %s
EOF

pythonBinary="python"
if [[ -e venv/bin/python ]]; then
	pythonBinary="venv/bin/python"
fi
`, opts.url, opts.username, opts.password)
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
		if err := grpcplugins.RunScript(ctx, chanRes, scriptWorkDir, pullScript); err != nil {
			fmt.Printf("%+v\n", err)
		}
	})

	select {
	case <-ctx.Done():
		fmt.Printf("CDS Worker execution canceled: %v", ctx.Err())
		return errors.New("CDS Worker execution canceled")
	case res := <-chanRes:
		if res.Status != sdk.StatusSuccess {
			return errors.New(res.Details)
		}
	}

	if integ != nil {
		repository := integ.Config[sdk.ArtifactoryConfigRepositoryPrefix].Value + "-pypi"
		maturity := integ.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value
		localRepository := repository + "-" + maturity

		rtConfig := grpcplugins.ArtifactoryConfig{
			URL:   integ.Config[sdk.ArtifactoryConfigURL].Value,
			Token: integ.Config[sdk.ArtifactoryConfigToken].Value,
		}
		if !strings.HasSuffix(rtConfig.URL, "/") {
			rtConfig.URL = rtConfig.URL + "/"
		}

		folderInfo, err := grpcplugins.GetArtifactoryFolderInfo(ctx, &actPlugin.Common, rtConfig, repository, path.Join(opts.packageName, opts.version))
		if err != nil {
			return err
		}
		for _, c := range folderInfo.Children {
			if c.Folder {
				continue
			}
			fi, err := grpcplugins.GetArtifactoryFileInfo(context.TODO(), &actPlugin.Common, rtConfig, repository, path.Join(opts.packageName, opts.version, strings.TrimPrefix(c.URI, "/")))
			if err != nil {
				return fmt.Errorf("unable to get Artifactory file info %s: %v", c.URI, err)
			}
			grpcplugins.Logf("Get info ok for %s", c.URI)

			// Python can upload a tar.gz file + a wheel file
			var runResult *sdk.V2WorkflowRunResult
			if c.URI == fmt.Sprintf("/%s-%s.%s", opts.packageName, opts.version, "tar.gz") {
				runResult = result.RunResult
			} else {
				// Create a new run result
				runResult = &sdk.V2WorkflowRunResult{
					IssuedAt: time.Now(),
					Type:     sdk.V2WorkflowRunResultTypePython,
					Status:   sdk.V2WorkflowRunResultStatusPending,
					Detail: sdk.V2WorkflowRunResultDetail{
						Data: sdk.V2WorkflowRunResultPythonDetail{
							Name:      strings.TrimPrefix(c.URI, "/"),
							Version:   opts.version,
							Extension: strings.TrimPrefix(filepath.Ext(c.URI), "."),
						},
					},
				}
			}
			grpcplugins.ExtractFileInfoIntoRunResult(runResult, *fi, opts.packageName, "python", localRepository, repository, maturity)
			runResult.Status = sdk.V2WorkflowRunResultStatusCompleted

			var runResultRequest = workerruntime.V2RunResultRequest{
				RunResult: runResult,
			}

			if c.URI == fmt.Sprintf("/%s-%s.%s", opts.packageName, opts.version, "tar.gz") {
				grpcplugins.Logf("Updating run result: %s", runResultRequest.RunResult.Name())
				if _, err := grpcplugins.UpdateRunResult(ctx, &actPlugin.Common, &runResultRequest); err != nil {
					return err
				}
			} else {
				if _, err := grpcplugins.CreateRunResult(ctx, &actPlugin.Common, &runResultRequest); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
