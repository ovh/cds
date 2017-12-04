package workflow

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getNodeJobRunParameters(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, stage *sdk.Stage) ([]sdk.Parameter, *sdk.MultiError) {
	params := run.BuildParameters
	tmp := map[string]string{}

	tmp["cds.stage"] = stage.Name
	tmp["cds.job"] = j.Action.Name
	errm := &sdk.MultiError{}

	for k, v := range tmp {
		s, err := sdk.Interpolate(v, tmp)
		if err != nil {
			errm.Append(err)
			continue
		}
		sdk.AddParameter(&params, k, sdk.StringParameter, s)
	}

	if errm.IsEmpty() {
		return params, nil
	}
	return params, errm
}

// GetNodeBuildParameters returns build parameters with default values for cds.version, cds.run, cds.run.number, cds.run.subnumber
func GetNodeBuildParameters(proj *sdk.Project, w *sdk.Workflow, n *sdk.WorkflowNode, pipelineParameters []sdk.Parameter, payload interface{}) ([]sdk.Parameter, error) {
	vars := map[string]string{}
	tmpProj := sdk.ParametersFromProjectVariables(proj)
	for k, v := range tmpProj {
		vars[k] = v
	}

	// compute application variables
	if n.Context != nil && n.Context.Application != nil {
		vars["cds.application"] = n.Context.Application.Name
		tmp := sdk.ParametersFromApplicationVariables(n.Context.Application)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// compute environment variables
	if n.Context != nil && n.Context.Environment != nil {
		vars["cds.environment"] = n.Context.Environment.Name
		tmp := sdk.ParametersFromEnvironmentVariables(n.Context.Environment)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// compute pipeline parameters
	tmpPip := sdk.ParametersFromPipelineParameters(pipelineParameters)
	for k, v := range tmpPip {
		vars[k] = v
	}

	// compute payload
	errm := &sdk.MultiError{}
	e := dump.NewDefaultEncoder(new(bytes.Buffer))
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	e.ExtraFields.DetailedMap = false
	e.ExtraFields.DetailedStruct = false
	e.ExtraFields.Len = false
	e.ExtraFields.Type = false
	tmpVars, errdump := e.ToStringMap(payload)
	if errdump != nil {
		log.Error("GetNodeBuildParameters> do-dump error: %v", errdump)
		errm.Append(errdump)
	}

	//Merge the dumped payload with vars
	vars = sdk.ParametersMapMerge(vars, tmpVars)

	log.Debug("GetNodeBuildParameters> compute payload :%#v", payload)

	// TODO Update suggest.go  with new variable

	vars["cds.project"] = w.ProjectKey
	vars["cds.workflow"] = w.Name
	vars["cds.pipeline"] = n.Pipeline.Name

	params := []sdk.Parameter{}
	for k, v := range vars {
		sdk.AddParameter(&params, k, sdk.StringParameter, v)
	}

	if errm.IsEmpty() {
		return params, nil
	}

	return params, errm
}

func getParentParameters(db gorp.SqlExecutor, run *sdk.WorkflowNodeRun, nodeRunIds []int64, payload map[string]string) ([]sdk.Parameter, error) {
	//Load workflow run
	w, err := LoadRunByID(db, run.WorkflowRunID, false)
	if err != nil {
		return nil, sdk.WrapError(err, "getParentParameters> Unable to load workflow run")
	}

	params := []sdk.Parameter{}
	for _, nodeRunID := range nodeRunIds {
		parentNodeRun, errNR := LoadNodeRunByID(db, nodeRunID, false)
		if errNR != nil {
			return nil, sdk.WrapError(errNR, "getParentParameters> Cannot get parent node run")
		}

		node := w.Workflow.GetNode(parentNodeRun.WorkflowNodeID)
		if node == nil {
			return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", parentNodeRun.WorkflowNodeID), "getParentParameters>")
		}

		for i := range parentNodeRun.BuildParameters {
			p := &parentNodeRun.BuildParameters[i]

			if p.Name == "" || p.Name == "cds.semver" || p.Name == "cds.release.version" ||
				strings.HasPrefix(p.Name, "cds.proj") || strings.HasPrefix(p.Name, "workflow.") ||
				strings.HasPrefix(p.Name, "cds.version") || strings.HasPrefix(p.Name, "cds.run.number") ||
				strings.HasPrefix(p.Name, "cds.workflow") {
				continue
			}

			// Do not duplicate variable from payload
			if _, ok := payload[p.Name]; ok {
				continue
			}

			prefix := "workflow." + node.Name + "."
			if strings.HasPrefix(p.Name, "cds.") {
				p.Name = strings.Replace(p.Name, "cds.", prefix, 1)
			} else {
				p.Name = prefix + p.Name
			}
		}
		params = append(params, parentNodeRun.BuildParameters...)
	}
	return params, nil
}

func getNodeRunBuildParameters(db gorp.SqlExecutor, proj *sdk.Project, run *sdk.WorkflowNodeRun) ([]sdk.Parameter, error) {
	//Load workflow run
	w, err := LoadRunByID(db, run.WorkflowRunID, false)
	if err != nil {
		return nil, sdk.WrapError(err, "getNodeRunParameters> Unable to load workflow run")
	}

	//Load node definition
	n := w.Workflow.GetNode(run.WorkflowNodeID)
	if n == nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", run.WorkflowNodeID), "getNodeRunParameters>")
	}

	//Get node build parameters
	errm := &sdk.MultiError{}
	params, errparam := GetNodeBuildParameters(proj, &w.Workflow, n, run.PipelineParameters, run.Payload)
	if errparam != nil {
		err, ok := errparam.(*sdk.MultiError)
		if ok {
			errm = err
		} else {
			return nil, sdk.WrapError(err, "getNodeRunParameters> Unable to compute node build parameters")
		}
	}

	//override default parameters value
	tmp := sdk.ParametersToMap(params)
	tmp["cds.version"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run.number"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run.subnumber"] = fmt.Sprintf("%d", run.SubNumber)

	params = []sdk.Parameter{}
	for k, v := range tmp {
		s, err := sdk.Interpolate(v, tmp)
		if err != nil {
			errm.Append(err)
			continue
		}
		sdk.AddParameter(&params, k, sdk.StringParameter, s)
	}

	if errm.IsEmpty() {
		return params, nil
	}

	return params, errm
}
