package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func runArtifactDownload(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
		res := &sdk.Result{Status: sdk.StatusSuccess.String()}

		project := sdk.ParameterValue(*params, "cds.project")
		workflow := sdk.ParameterValue(*params, "cds.workflow")
		number := sdk.ParameterValue(*params, "cds.run.number")
		enabled := sdk.ParameterValue(*params, "enabled") != "false"
		pattern := sdk.ParameterValue(a.Parameters, "pattern")

		destPath := sdk.ParameterValue(a.Parameters, "path")
		tag := sdk.ParameterValue(a.Parameters, "tag")

		if destPath == "" {
			destPath = "."
		}

		if !enabled {
			sendLog("Artifact Download is disabled.")
			return *res
		}

		if err := os.MkdirAll(destPath, os.FileMode(0744)); err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("Unable to create %s: %v", destPath, err)
			sendLog(res.Reason)
			return *res
		}

		sendLog(fmt.Sprintf("Downloading artifacts from workflow into '%s'...", destPath))

		n, err := strconv.ParseInt(number, 10, 64)
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("cds.run.number variable is not valid. aborting")
			sendLog(res.Reason)
			return *res
		}
		artifacts, err := w.client.WorkflowRunArtifacts(project, workflow, n)
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = err.Error()
			log.Warning("Cannot download artifacts: %s", err)
			sendLog(res.Reason)
			return *res
		}

		regexp := regexp.MustCompile(pattern)
		wg := new(sync.WaitGroup)
		wg.Add(len(artifacts))

		for i := range artifacts {
			a := &artifacts[i]

			if pattern != "" && !regexp.MatchString(a.Name) {
				sendLog(fmt.Sprintf("%s does not match pattern %s - skipped", a.Name, pattern))
				wg.Done()
				continue
			}

			if tag != "" && a.Tag != tag {
				sendLog(fmt.Sprintf("%s does not match tag %s - skipped", a.Name, tag))
				wg.Done()
				continue
			}

			go func(a *sdk.WorkflowNodeRunArtifact) {
				defer wg.Done()

				destFile := path.Join(destPath, a.Name)
				f, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(a.Perm))
				if err != nil {
					res.Status = sdk.StatusFail.String()
					res.Reason = err.Error()
					log.Warning("Cannot download artifact (OpenFile) %s: %s", destFile, err)
					sendLog(res.Reason)
					return
				}
				sendLog(fmt.Sprintf("downloading artifact %s from workflow %s/%s on run %d...", destFile, project, workflow, n))
				if err := w.client.WorkflowNodeRunArtifactDownload(project, workflow, *a, f); err != nil {
					res.Status = sdk.StatusFail.String()
					res.Reason = err.Error()
					log.Warning("Cannot download artifact %s: %s", destFile, err)
					sendLog(res.Reason)
					return
				}
				if err := f.Close(); err != nil {
					res.Status = sdk.StatusFail.String()
					res.Reason = err.Error()
					log.Warning("Cannot download artifact %s: %s", destFile, err)
					sendLog(res.Reason)
					return
				}
			}(a)
			if len(artifacts) > 1 {
				time.Sleep(3 * time.Second)
			}
		}

		wg.Wait()
		return *res
	}
}
