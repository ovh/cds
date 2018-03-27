package workflow

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/log"
)

func getNodeJobRunParameters(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, stage *sdk.Stage) ([]sdk.Parameter, *sdk.MultiError) {
	params := run.BuildParameters
	tmp := map[string]string{}

	tmp["cds.stage"] = stage.Name
	tmp["cds.job"] = j.Action.Name
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
func GetNodeBuildParameters(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, n *sdk.WorkflowNode, pipelineParameters []sdk.Parameter, payload interface{}) ([]sdk.Parameter, error) {
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

		// Get GitUrl
		projectVCSServer := repositoriesmanager.GetProjectVCSServer(proj, n.Context.Application.VCSServer)
		if projectVCSServer != nil {
			client, errclient := repositoriesmanager.AuthorizedClient(db, store, projectVCSServer)
			if errclient != nil {
				return nil, sdk.WrapError(errclient, "GetNodeBuildParameters> Cannot connect get repository manager client")
			}
			r, errR := client.RepoByFullname(n.Context.Application.RepositoryFullname)
			if errR != nil {
				return nil, sdk.WrapError(errR, "GetNodeBuildParameters> Cannot get git.url")
			}
			vars["git.url"] = r.SSHCloneURL
			vars["git.http_url"] = r.HTTPCloneURL

			branches, errB := client.Branches(r.Fullname)
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

	log.Debug("GetNodeBuildParameters> compute payload :%#v", payload)

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

		if _, ok := vars["git.branch"]; !ok && n.Context.Application.RepositoryStrategy.Branch != "" {
			vars["git.branch"] = n.Context.Application.RepositoryStrategy.Branch
		}
		if _, ok := vars["git.default_branch"]; !ok && n.Context.Application.RepositoryStrategy.DefaultBranch != "" {
			vars["git.default_branch"] = n.Context.Application.RepositoryStrategy.Branch
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

func getNodeRunBuildParameters(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, run *sdk.WorkflowNodeRun) ([]sdk.Parameter, error) {
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
	params, errparam := GetNodeBuildParameters(db, store, proj, &w.Workflow, n, run.PipelineParameters, run.Payload)
	if errparam != nil {
		return nil, sdk.WrapError(err, "getNodeRunParameters> Unable to compute node build parameters")
	}

	errm := &sdk.MultiError{}
	//override default parameters value
	tmp := sdk.ParametersToMap(params)
	tmp["cds.version"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run.number"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run.subnumber"] = fmt.Sprintf("%d", run.SubNumber)

	params = []sdk.Parameter{}
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
