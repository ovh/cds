package action

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func RunArtifactDownload(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, params []sdk.Parameter, secrets []sdk.Variable) (sdk.Result, error) {
	res := sdk.Result{Status: sdk.StatusSuccess}

	project := sdk.ParameterValue(params, "cds.project")
	workflow := sdk.ParameterValue(params, "cds.workflow")
	number := sdk.ParameterValue(params, "cds.run.number")
	enabled := sdk.ParameterValue(params, "enabled") != "false"
	pattern := sdk.ParameterValue(a.Parameters, "pattern")

	destPath := sdk.ParameterValue(a.Parameters, "path")
	tag := sdk.ParameterValue(a.Parameters, "tag")

	if destPath == "" {
		destPath = "."
	}

	// TODO: we should remove this
	if !enabled {
		wk.SendLog(workerruntime.LevelDebug, "Artifact Download is disabled")
		return res, nil
	}

	if err := os.MkdirAll(destPath, os.FileMode(0744)); err != nil {
		return res, fmt.Errorf("Unable to create %s: %v", destPath, err)
	}

	wk.SendLog(workerruntime.LevelInfo, fmt.Sprintf("Downloading artifacts from workflow into '%s'...", destPath))

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
		wk.SendLog(workerruntime.LevelInfo, res.Reason)
		return res, err
	}

	wg := new(sync.WaitGroup)
	wg.Add(len(artifacts))

	for i := range artifacts {
		a := &artifacts[i]

		if pattern != "" && !regexp.MatchString(a.Name) {
			wk.SendLog(workerruntime.LevelInfo, fmt.Sprintf("%s does not match pattern %s - skipped", a.Name, pattern))
			wg.Done()
			continue
		}

		if tag != "" && a.Tag != tag {
			wk.SendLog(workerruntime.LevelInfo, fmt.Sprintf("%s does not match tag %s - skipped", a.Name, tag))
			wg.Done()
			continue
		}

		go func(a *sdk.WorkflowNodeRunArtifact) {
			defer wg.Done()

			destFile := path.Join(destPath, a.Name)
			f, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(a.Perm))
			if err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warning("Cannot download artifact (OpenFile) %s: %s", destFile, err)
				wk.SendLog(workerruntime.LevelError, res.Reason)
				return
			}
			wk.SendLog(workerruntime.LevelInfo, fmt.Sprintf("downloading artifact %s from workflow %s/%s on run %d...", destFile, project, workflow, n))
			if err := wk.Client().WorkflowNodeRunArtifactDownload(project, workflow, *a, f); err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warning("Cannot download artifact %s: %s", destFile, err)
				wk.SendLog(workerruntime.LevelError, res.Reason)
				return
			}
			if err := f.Close(); err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warning("Cannot download artifact %s: %s", destFile, err)
				wk.SendLog(workerruntime.LevelError, res.Reason)
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
