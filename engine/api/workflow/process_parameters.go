package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/fsamin/go-dump"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/telemetry"
)

func getNodeJobRunParameters(j sdk.Job, run *sdk.WorkflowNodeRun, stage *sdk.Stage) ([]sdk.Parameter, *sdk.MultiError) {
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

// getBuildParameterFromNodeContext returns the parameters compute from  node context (project, application,  pipeline, pyaload)
func getBuildParameterFromNodeContext(proj sdk.Project, w sdk.Workflow, runContext nodeRunContext, pipelineParameters []sdk.Parameter, payload interface{}, hookEvent *sdk.WorkflowNodeRunHookEvent) ([]sdk.Parameter, error) {
	tmpProj := sdk.ParametersFromProjectVariables(proj)
	vars := make(map[string]string, len(tmpProj))
	for k, v := range tmpProj {
		vars[k] = v
	}

	tmpProj = sdk.ParametersFromProjectKeys(proj)
	for k, v := range tmpProj {
		vars[k] = v
	}

	// COMPUTE APPLICATION VARIABLE
	if runContext.Application.ID != 0 {
		vars["cds.application"] = runContext.Application.Name
		tmp := sdk.ParametersFromApplicationVariables(runContext.Application)
		for k, v := range tmp {
			vars[k] = v
		}

		tmp = sdk.ParametersFromApplicationKeys(runContext.Application)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// COMPUTE ENVIRONMENT VARIABLE
	if runContext.Environment.ID != 0 {
		vars["cds.environment"] = runContext.Environment.Name
		tmp := sdk.ParametersFromEnvironmentVariables(runContext.Environment)
		for k, v := range tmp {
			vars[k] = v
		}
		tmp = sdk.ParametersFromEnvironmentKeys(runContext.Environment)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// COMPUTE  INTEGRATION VARIABLE
	if runContext.ProjectIntegration.ID != 0 {
		vars["cds.integration"] = runContext.ProjectIntegration.Name
		tmp := sdk.ParametersFromIntegration(runContext.ProjectIntegration.Config)
		for k, v := range tmp {
			vars[k] = v
		}

		// COMPUTE DEPLOYMENT STRATEGIES VARIABLE
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

	// COMPUTE PIPELINE PARAMETER
	tmpPip := sdk.ParametersFromPipelineParameters(pipelineParameters)
	for k, v := range tmpPip {
		vars[k] = v
	}

	// COMPUTE PAYLOAD
	e := dump.NewDefaultEncoder()

	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	e.ExtraFields.DetailedMap = false
	e.ExtraFields.DetailedStruct = false
	e.ExtraFields.Len = false
	e.ExtraFields.Type = false
	tmpVars, errdump := e.ToStringMap(payload)
	if errdump != nil {
		return nil, sdk.WrapError(errdump, "do-dump error")
	}

	//Merge the dumped payload with vars
	vars = sdk.ParametersMapMerge(vars, tmpVars)

	vars["cds.project"] = w.ProjectKey
	vars["cds.workflow"] = w.Name
	vars["cds.pipeline"] = runContext.Pipeline.Name

	// COMPUTE VCS STRATEGY VARIABLE
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

func getParentParameters(w *sdk.WorkflowRun, nodeRuns []*sdk.WorkflowNodeRun) ([]sdk.Parameter, error) {
	params := make([]sdk.Parameter, 0, len(nodeRuns))
	for _, parentNodeRun := range nodeRuns {
		var nodeName string

		node := w.Workflow.WorkflowData.NodeByID(parentNodeRun.WorkflowNodeID)
		if node == nil {
			return nil, sdk.WithStack(fmt.Errorf("Unable to find node %d in workflow", parentNodeRun.WorkflowNodeID))
		}
		nodeName = node.Name

		parentParams := make([]sdk.Parameter, 0, len(parentNodeRun.BuildParameters))
		for _, param := range parentNodeRun.BuildParameters {
			prefix := "workflow." + nodeName + "."
			if param.Name == "" || param.Name == "cds.semver" || param.Name == "cds.release.version" ||
				strings.HasPrefix(param.Name, "cds.proj") ||
				strings.HasPrefix(param.Name, "cds.version") || strings.HasPrefix(param.Name, "cds.run.number") ||
				strings.HasPrefix(param.Name, "cds.workflow") || strings.HasPrefix(param.Name, "job.requirement") {
				continue
			}

			if strings.HasPrefix(param.Name, "workflow.") {
				parentParams = append(parentParams, param)
				continue
			}

			// We inherite git variables is there is more than one repositories in the whole workflow
			if strings.HasPrefix(param.Name, "git.") {
				parentParams = append(parentParams, param)

				// Create parent param
				param.Name = prefix + param.Name
				parentParams = append(parentParams, param)
				continue
			}
			if strings.HasPrefix(param.Name, "gerrit.") {
				parentParams = append(parentParams, param)
				continue
			}

			if param.Name == "payload" || strings.HasPrefix(param.Name, "cds.triggered") || strings.HasPrefix(param.Name, "cds.release") {
				// keep p.Name as is
			} else if strings.HasPrefix(param.Name, "cds.") {
				param.Name = strings.Replace(param.Name, "cds.", prefix, 1)
			}
			parentParams = append(parentParams, param)
		}
		params = append(params, parentParams...)
	}
	return params, nil
}

func getNodeRunBuildParameters(ctx context.Context, proj sdk.Project, wr *sdk.WorkflowRun, run *sdk.WorkflowNodeRun, runContext nodeRunContext) ([]sdk.Parameter, error) {
	ctx, end := telemetry.Span(ctx, "workflow.getNodeRunBuildParameters",
		telemetry.Tag(telemetry.TagWorkflow, wr.Workflow.Name),
		telemetry.Tag(telemetry.TagWorkflowRun, wr.Number),
		telemetry.Tag(telemetry.TagWorkflowNodeRun, run.ID),
	)
	defer end()

	// GET PARAMETER FROM NODE CONTEXT
	params, errparam := getBuildParameterFromNodeContext(proj, wr.Workflow, runContext, run.PipelineParameters, run.Payload, run.HookEvent)
	if errparam != nil {
		return nil, sdk.WrapError(errparam, "unable to compute node build parameters")
	}

	errm := &sdk.MultiError{}
	//override default parameters value
	tmp := sdk.ParametersToMap(params)
	if wr.Version != nil {
		tmp["cds.version"] = *wr.Version
	} else {
		tmp["cds.version"] = fmt.Sprintf("%d", run.Number)
	}
	tmp["cds.run"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run.number"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run.subnumber"] = fmt.Sprintf("%d", run.SubNumber)

	if wr.Workflow.TemplateInstance != nil {
		tmp["cds.template.version"] = fmt.Sprintf("%d", wr.Workflow.TemplateInstance.WorkflowTemplateVersion)
	}

	_, next := telemetry.Span(ctx, "workflow.interpolate")
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
