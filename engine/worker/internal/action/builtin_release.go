package action

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func RunRelease(ctx context.Context, wk workerruntime.Runtime, a *sdk.Action, params []sdk.Parameter, secrets []sdk.Variable) (sdk.Result, error) {
	var res sdk.Result

	jobID, err := workerruntime.JobID(ctx)
	if err != nil {
		return res, err
	}

	artifactList := sdk.ParameterFind(a.Parameters, "artifacts")
	tag := sdk.ParameterFind(a.Parameters, "tag")
	title := sdk.ParameterFind(a.Parameters, "title")
	releaseNote := sdk.ParameterFind(a.Parameters, "releaseNote")

	pkey := sdk.ParameterFind(params, "cds.project")
	wName := sdk.ParameterFind(params, "cds.workflow")
	workflowNum := sdk.ParameterFind(params, "cds.run.number")

	if pkey == nil {
		return res, errors.New("cds.project variable not found")
	}

	if wName == nil {
		return res, errors.New("cds.workflow variable not found")
	}

	if workflowNum == nil {
		return res, errors.New("cds.run.number variable not found")
	}

	if tag == nil || tag.Value == "" {
		return res, errors.New("tag name is not set. Nothing to perform")
	}

	if title == nil || title.Value == "" {
		return res, errors.New("release title is not set")
	}

	if releaseNote == nil || releaseNote.Value == "" {
		return res, errors.New("release note is not set")
	}

	wRunNumber, errI := strconv.ParseInt(workflowNum.Value, 10, 64)
	if errI != nil {
		return res, fmt.Errorf("Workflow number is not a number. Got %s: %s", workflowNum.Value, errI)
	}

	artSplitted := strings.Split(artifactList.Value, ",")
	req := sdk.WorkflowNodeRunRelease{
		ReleaseContent: releaseNote.Value,
		ReleaseTitle:   title.Value,
		TagName:        tag.Value,
		Artifacts:      artSplitted,
	}

	if err := wk.Client().WorkflowNodeRunRelease(pkey.Value, wName.Value, wRunNumber, jobID, req); err != nil {
		return res, fmt.Errorf("Cannot make workflow node run release: %s", err)
	}

	return sdk.Result{Status: sdk.StatusSuccess}, nil
}
