package action

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func RunArtifactDownload(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, secrets []sdk.Variable) (sdk.Result, error) {
	res := sdk.Result{Status: sdk.StatusSuccess}

	project := sdk.ParameterValue(wk.Parameters(), "cds.project")
	workflow := sdk.ParameterValue(wk.Parameters(), "cds.workflow")
	number := sdk.ParameterValue(wk.Parameters(), "cds.run.number")
	pattern := sdk.ParameterValue(a.Parameters, "pattern")

	destPath := sdk.ParameterValue(a.Parameters, "path")
	tag := sdk.ParameterValue(a.Parameters, "tag")

	if destPath == "" {
		destPath = "."
	}

	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		return res, err
	}

	var abs string
	if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
		abs, _ = x.RealPath(workdir.Name())
	} else {
		abs = workdir.Name()
	}

	if !sdk.PathIsAbs(destPath) {
		destPath = filepath.Join(abs, destPath)
	}

	wkDirFS := afero.NewOsFs()
	if err := wkDirFS.MkdirAll(destPath, os.FileMode(0744)); err != nil {
		return res, fmt.Errorf("Unable to create %s: %v", destPath, err)
	}

	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Downloading artifacts from workflow into '%s'...", destPath))

	n, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return res, fmt.Errorf("cds.run.number variable is not valid. aborting")
	}

	artifacts, err := wk.Client().WorkflowRunArtifacts(project, workflow, n)
	if err != nil {
		return res, err
	}

	regexp, err := regexp.Compile(pattern)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Reason = fmt.Sprintf("Invalid pattern %s, must be a regex : %v", pattern, err)
		wk.SendLog(ctx, workerruntime.LevelInfo, res.Reason)
		return res, err
	}

	wg := new(sync.WaitGroup)
	wg.Add(len(artifacts))

	for i := range artifacts {
		a := &artifacts[i]

		if pattern != "" && !regexp.MatchString(a.Name) {
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("%s does not match pattern %s - skipped", a.Name, pattern))
			wg.Done()
			continue
		}

		if tag != "" && a.Tag != tag {
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("%s does not match tag %s - skipped", a.Name, tag))
			wg.Done()
			continue
		}

		go func(a *sdk.WorkflowNodeRunArtifact) {
			defer wg.Done()

			destFile := path.Join(destPath, a.Name)
			f, err := wkDirFS.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(a.Perm))
			if err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warn(ctx, "Cannot download artifact (OpenFile) %s: %s", destFile, err)
				wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
				return
			}
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Downloading artifact %s from workflow %s/%s on run %d...", destFile, project, workflow, n))
			if err := wk.Client().WorkflowNodeRunArtifactDownload(project, workflow, *a, f); err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warn(ctx, "Cannot download artifact %s: %s", destFile, err)
				wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
				return
			}
			if err := f.Close(); err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warn(ctx, "Cannot download artifact %s: %s", destFile, err)
				wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
				return
			}
		}(a)
		// TODO: write here a reason why we are waiting 3 seconds
		if len(artifacts) > 1 {
			time.Sleep(3 * time.Second)
		}
	}

	wg.Wait()
	return res, nil
}
