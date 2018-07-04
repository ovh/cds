package workflow

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/tracing"
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
func GetNodeBuildParameters(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, n *sdk.WorkflowNode, pipelineParameters []sdk.Parameter, payload interface{}) ([]sdk.Parameter, error) {
	tmpProj := sdk.ParametersFromProjectVariables(*proj)
	vars := make(map[string]string, len(tmpProj))
	for k, v := range tmpProj {
		vars[k] = v
	}

	// compute application variables
	app, has := n.Application()
	if has {
		vars["cds.application"] = app.Name
		tmp := sdk.ParametersFromApplicationVariables(app)
		for k, v := range tmp {
			vars[k] = v
		}

		// Get GitUrl
		projectVCSServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
		if projectVCSServer != nil {
			client, errclient := repositoriesmanager.AuthorizedClient(db, store, projectVCSServer)
			if errclient != nil {
				return nil, sdk.WrapError(errclient, "GetNodeBuildParameters> Cannot connect get repository manager client")
			}
			_, next := tracing.Span(ctx, "workflow.GetNodeBuildParameters.vcs.RepoByFullname")
			r, errR := client.RepoByFullname(app.RepositoryFullname)
			next()

			if errR != nil {
				return nil, sdk.WrapError(errR, "GetNodeBuildParameters> Cannot get git.url")
			}
			vars["git.url"] = r.SSHCloneURL
			vars["git.http_url"] = r.HTTPCloneURL

			_, next = tracing.Span(ctx, "workflow.GetNodeBuildParameters.vcs.Branches")
			branches, errB := client.Branches(r.Fullname)
			next()

			if errB != nil {
				return nil, sdk.WrapError(errB, "GetNodeBuildParameters> Cannot get branches on %s, app:%s", r.SSHCloneURL, n.Context.Application.Name)
			}
			for _, b := range branches {
				if b.Default {
					vars["git.default_branch"] = b.DisplayID
					break
				}
			}
		}
	}

	// compute environment variables
	env, has := n.Environment()
	if has {
		vars["cds.environment"] = env.Name
		tmp := sdk.ParametersFromEnvironmentVariables(env)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// compute parameters variables
	ppf, has := n.ProjectPlatform()
	if has {
		vars["cds.platform"] = ppf.Name
		tmp := sdk.ParametersFromPlatform(ppf.Config)
		for k, v := range tmp {
			vars[k] = v
		}

		// Process deployment strategy of the chosen platform
		if n.Context.Application != nil {
			for pfName, pfConfig := range n.Context.Application.DeploymentStrategies {
				if pfName == n.Context.ProjectPlatform.Name {
					tmp := sdk.ParametersFromPlatform(pfConfig)
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
	vars = sdk.ParametersMapMerge(vars, tmpVars)

	// TODO Update suggest.go  with new variable

	vars["cds.project"] = w.ProjectKey
	vars["cds.workflow"] = w.Name
	vars["cds.pipeline"] = n.Pipeline.Name

	if n.Context != nil && n.Context.Application != nil && n.Context.Application.RepositoryStrategy.ConnectionType != "" {
		vars["git.connection.type"] = n.Context.Application.RepositoryStrategy.ConnectionType
		if n.Context.Application.RepositoryStrategy.SSHKey != "" {
			vars["git.ssh.key"] = n.Context.Application.RepositoryStrategy.SSHKey
		}
		if n.Context.Application.RepositoryStrategy.PGPKey != "" {
			vars["git.pgp.key"] = n.Context.Application.RepositoryStrategy.PGPKey
		}
		if n.Context.Application.RepositoryStrategy.User != "" {
			vars["git.http.user"] = n.Context.Application.RepositoryStrategy.User
		}
	} else {
		// remove vcs strategy variable
		delete(vars, "git.ssh.key")
		delete(vars, "git.pgp.key")
		delete(vars, "git.http.user")
	}

	params := []sdk.Parameter{}
	for k, v := range vars {
		sdk.AddParameter(&params, k, sdk.StringParameter, v)
	}

	return params, nil
}

func getParentParameters(db gorp.SqlExecutor, w *sdk.WorkflowRun, run *sdk.WorkflowNodeRun, nodeRunIds []int64, payload map[string]string) ([]sdk.Parameter, error) {
	params := make([]sdk.Parameter, 0, len(nodeRunIds))
	for _, nodeRunID := range nodeRunIds {
		parentNodeRun, errNR := LoadNodeRunByID(db, nodeRunID, LoadRunOptions{})
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
				strings.HasPrefix(p.Name, "cds.workflow") || strings.HasPrefix(p.Name, "job.requirement") {
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

func getNodeRunBuildParameters(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.WorkflowRun, run *sdk.WorkflowNodeRun) ([]sdk.Parameter, error) {
	ctx, end := tracing.Span(ctx, "workflow.getNodeRunBuildParameters",
		tracing.Tag("workflow", w.Workflow.Name),
		tracing.Tag("workflow_run", w.Number),
		tracing.Tag("workflow_node_run", run.ID),
	)
	defer end()

	//Load node definition
	n := w.Workflow.GetNode(run.WorkflowNodeID)
	if n == nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", run.WorkflowNodeID), "getNodeRunParameters>")
	}

	//Get node build parameters
	params, errparam := GetNodeBuildParameters(ctx, db, store, proj, &w.Workflow, n, run.PipelineParameters, run.Payload)
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

	_, next := tracing.Span(ctx, "workflow.interpolate")
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
