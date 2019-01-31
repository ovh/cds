package workflow

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
)

func getNodeJobRunParameters(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, stage *sdk.Stage) ([]sdk.Parameter, *sdk.MultiError) {
	params := run.BuildParameters
	tmp := map[string]string{
		"cds.stage": stage.Name,
		"cds.job":   j.Action.Name,
	}
	errm := &sdk.MultiError{}

	for k, v := range tmp {
		s, err := interpolate.Do(v, tmp)
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
func GetNodeBuildParameters(proj *sdk.Project, w *sdk.Workflow, runContext nodeRunContext, pipelineParameters []sdk.Parameter, payload interface{}, hookEvent *sdk.WorkflowNodeRunHookEvent) ([]sdk.Parameter, error) {
	tmpProj := sdk.ParametersFromProjectVariables(*proj)
	vars := make(map[string]string, len(tmpProj))
	for k, v := range tmpProj {
		vars[k] = v
	}

	// compute application variables
	if runContext.Application.ID != 0 {
		vars["cds.application"] = runContext.Application.Name
		tmp := sdk.ParametersFromApplicationVariables(runContext.Application)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// compute environment variables
	if runContext.Environment.ID != 0 {
		vars["cds.environment"] = runContext.Environment.Name
		tmp := sdk.ParametersFromEnvironmentVariables(runContext.Environment)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// compute parameters variables
	if runContext.ProjectIntegration.ID != 0 {
		vars["cds.integration"] = runContext.ProjectIntegration.Name
		tmp := sdk.ParametersFromIntegration(runContext.ProjectIntegration.Config)
		for k, v := range tmp {
			vars[k] = v
		}

		// Process deployment strategy of the chosen integration
		if runContext.Application.ID != 0 {
			for pfName, pfConfig := range runContext.Application.DeploymentStrategies {
				if pfName == runContext.ProjectIntegration.Name {
					tmp := sdk.ParametersFromIntegration(pfConfig)
					for k, v := range tmp {
						vars[k] = v
					}
				}
			}
		}
	}

	// compute pipeline parameters
	tmpPip := sdk.ParametersFromPipelineParameters(pipelineParameters)
	for k, v := range tmpPip {
		vars[k] = v
	}

	// compute payload
	e := dump.NewDefaultEncoder(new(bytes.Buffer))
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	e.ExtraFields.DetailedMap = false
	e.ExtraFields.DetailedStruct = false
	e.ExtraFields.Len = false
	e.ExtraFields.Type = false
	tmpVars, errdump := e.ToStringMap(payload)
	if errdump != nil {
		return nil, sdk.WrapError(errdump, "GetNodeBuildParameters> do-dump error")
	}

	//Merge the dumped payload with vars
	vars = sdk.ParametersMapMerge(vars, tmpVars, sdk.MapMergeOptions.ExcludeGitParams)

	// TODO Update suggest.go  with new variable

	vars["cds.project"] = w.ProjectKey
	vars["cds.workflow"] = w.Name
	vars["cds.pipeline"] = runContext.Pipeline.Name

	if runContext.Application.RepositoryStrategy.ConnectionType != "" {
		vars["git.connection.type"] = runContext.Application.RepositoryStrategy.ConnectionType
		if runContext.Application.RepositoryStrategy.SSHKey != "" {
			vars["git.ssh.key"] = runContext.Application.RepositoryStrategy.SSHKey
		}
		if runContext.Application.RepositoryStrategy.PGPKey != "" {
			vars["git.pgp.key"] = runContext.Application.RepositoryStrategy.PGPKey
		}
		if runContext.Application.RepositoryStrategy.User != "" {
			vars["git.http.user"] = runContext.Application.RepositoryStrategy.User
		}
		if runContext.Application.VCSServer != "" {
			vars["git.server"] = runContext.Application.VCSServer
		}
	} else {
		// remove vcs strategy variable
		delete(vars, "git.ssh.key")
		delete(vars, "git.pgp.key")
		delete(vars, "git.http.user")
	}

	if hookEvent != nil {
		vars["parent.project"] = hookEvent.ParentWorkflow.Key
		vars["parent.run"] = fmt.Sprintf("%d", hookEvent.ParentWorkflow.Run)
		vars["parent.workflow"] = hookEvent.ParentWorkflow.Name
		vars["parent.outgoinghook"] = hookEvent.WorkflowNodeHookUUID
	}

	params := []sdk.Parameter{}
	for k, v := range vars {
		sdk.AddParameter(&params, k, sdk.StringParameter, v)
	}

	return params, nil
}

func getParentParameters(w *sdk.WorkflowRun, nodeRuns []*sdk.WorkflowNodeRun, payload map[string]string) ([]sdk.Parameter, error) {
	repos := w.Workflow.GetRepositories()
	params := make([]sdk.Parameter, 0, len(nodeRuns))
	for _, parentNodeRun := range nodeRuns {
		var nodeName string

		node := w.Workflow.WorkflowData.NodeByID(parentNodeRun.WorkflowNodeID)
		if node == nil {
			return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", parentNodeRun.WorkflowNodeID), "getParentParameters>")
		}
		nodeName = node.Name

		for i := range parentNodeRun.BuildParameters {
			p := &parentNodeRun.BuildParameters[i]

			if p.Name == "" || p.Name == "cds.semver" || p.Name == "cds.release.version" ||
				strings.HasPrefix(p.Name, "cds.proj") || strings.HasPrefix(p.Name, "workflow.") ||
				strings.HasPrefix(p.Name, "cds.version") || strings.HasPrefix(p.Name, "cds.run.number") ||
				strings.HasPrefix(p.Name, "cds.workflow") || strings.HasPrefix(p.Name, "job.requirement") {
				continue
			}

			// Do not duplicate variable from payload
			if _, ok := payload[p.Name]; ok {
				if !strings.HasPrefix(p.Name, "git.") {
					continue
				}
			}

			// We inherite git variables is there is more than one repositories in the whole workflow
			if strings.HasPrefix(p.Name, "git.") && len(repos) == 1 {
				continue
			}

			prefix := "workflow." + nodeName + "."

			if p.Name == "payload" {
				// keep p.Name as is
			} else if strings.HasPrefix(p.Name, "cds.") {
				p.Name = strings.Replace(p.Name, "cds.", prefix, 1)
			} else {
				p.Name = prefix + p.Name
			}
		}
		params = append(params, parentNodeRun.BuildParameters...)
	}
	return params, nil
}

func getNodeRunBuildParameters(ctx context.Context, proj *sdk.Project, wr *sdk.WorkflowRun, run *sdk.WorkflowNodeRun, runContext nodeRunContext) ([]sdk.Parameter, error) {
	ctx, end := observability.Span(ctx, "workflow.getNodeRunBuildParameters",
		observability.Tag(observability.TagWorkflow, wr.Workflow.Name),
		observability.Tag(observability.TagWorkflowRun, wr.Number),
		observability.Tag(observability.TagWorkflowNodeRun, run.ID),
	)
	defer end()

	//Get node build parameters
	params, errparam := GetNodeBuildParameters(proj, &wr.Workflow, runContext, run.PipelineParameters, run.Payload, run.HookEvent)
	if errparam != nil {
		return nil, sdk.WrapError(errparam, "getNodeRunParameters> Unable to compute node build parameters")
	}

	errm := &sdk.MultiError{}
	//override default parameters value
	tmp := sdk.ParametersToMap(params)
	tmp["cds.version"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run.number"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run.subnumber"] = fmt.Sprintf("%d", run.SubNumber)

	_, next := observability.Span(ctx, "workflow.interpolate")
	params = make([]sdk.Parameter, 0, len(tmp))
	for k, v := range tmp {
		s, err := interpolate.Do(v, tmp)
		if err != nil {
			errm.Append(err)
			continue
		}
		sdk.AddParameter(&params, k, sdk.StringParameter, s)
	}
	next()

	if errm.IsEmpty() {
		return params, nil
	}

	return params, errm
}
