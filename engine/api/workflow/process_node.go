package workflow

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func processNodeTriggers(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, parentNodeRun []*sdk.WorkflowNodeRun, node *sdk.Node, parentSubNumber int) (*ProcessorReport, error) {
	report := new(ProcessorReport)

	for j := range node.Triggers {
		t := &node.Triggers[j]

		var abortTrigger bool
		if previousRunArray, ok := wr.WorkflowNodeRuns[t.ChildNode.ID]; ok {
			for _, previousRun := range previousRunArray {
				if int(previousRun.SubNumber) == parentSubNumber {
					abortTrigger = true
					break
				}
			}
		}

		if !abortTrigger {
			//Keep the subnumber of the previous node in the graph
			r1, _, errPwnr := processNodeRun(ctx, db, store, proj, wr, mapNodes, &t.ChildNode, int(parentSubNumber), parentNodeRun, nil, nil)
			if errPwnr != nil {
				log.Error("processWorkflowRun> Unable to process node ID=%d: %s", t.ChildNode.ID, errPwnr)
				AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
					ID:   sdk.MsgWorkflowError.ID,
					Args: []interface{}{errPwnr.Error()},
				})
			}
			_, _ = report.Merge(r1, nil)
			continue
		}
	}
	return report, nil
}

func processNodeRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, n *sdk.Node, subNumber int, parentNodeRuns []*sdk.WorkflowNodeRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	report := new(ProcessorReport)
	exist, errN := nodeRunExist(db, n.ID, wr.Number, subNumber)
	if errN != nil {
		return nil, false, sdk.WrapError(errN, "processNodeRun> unable to check if node run exist")
	}
	if exist {
		return nil, false, nil
	}

	var end func()
	ctx, end = observability.Span(ctx, "workflow.processWorkflowNodeRun",
		observability.Tag(observability.TagWorkflow, wr.Workflow.Name),
		observability.Tag(observability.TagWorkflowRun, wr.Number),
		observability.Tag(observability.TagWorkflowNode, n.Name),
	)
	defer end()

	switch n.Type {
	case sdk.NodeTypeFork, sdk.NodeTypePipeline, sdk.NodeTypeJoin:
		// r1, errT := processNodeTriggers(ctx, db, store, proj, wr, mapNodes, parentNodeRuns, n, subNumber)
		r1, conditionOK, errT := processNode(ctx, db, store, proj, wr, mapNodes, n, subNumber, parentNodeRuns, hookEvent, manual)
		_, _ = report.Merge(r1, nil)
		return report, conditionOK, sdk.WrapError(errT, "processNode> Unable to processNode")
	case sdk.NodeTypeOutGoingHook:
		r1, errO := processNodeOutGoingHook(ctx, db, store, proj, wr, mapNodes, parentNodeRuns, n, subNumber)
		_, _ = report.Merge(r1, nil)
		return report, true, sdk.WrapError(errO, "processNodeRun> Unable to processNodeOutGoingHook")
	}
	return nil, false, nil
}

func processNode(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, n *sdk.Node, subNumber int, parents []*sdk.WorkflowNodeRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	report := new(ProcessorReport)

	//TODO: Check user for manual done but check permission also for automatic trigger and hooks (with system to authenticate a webhook)
	if n.Context == nil {
		n.Context = &sdk.NodeContext{}
	}

	if n.Context.PipelineID == 0 && n.Type == sdk.NodeTypePipeline {
		return report, false, sdk.ErrPipelineNotFound
	}

	// For node with pipeline
	var stages []sdk.Stage
	if n.Context.PipelineID > 0 {
		pip, has := wr.Workflow.Pipelines[n.Context.PipelineID]
		if !has {
			return nil, false, fmt.Errorf("pipeline %d not found in workflow", n.Context.PipelineID)
		}
		stages = make([]sdk.Stage, len(pip.Stages))
		copy(stages, pip.Stages)

		//If the pipeline has parameter but none are defined on context, use the defaults
		if len(pip.Parameter) > 0 && len(n.Context.DefaultPipelineParameters) == 0 {
			n.Context.DefaultPipelineParameters = pip.Parameter
		}

	}

	// Create run
	run := &sdk.WorkflowNodeRun{
		WorkflowID:       wr.WorkflowID,
		LastModified:     time.Now(),
		Start:            time.Now(),
		Number:           wr.Number,
		SubNumber:        int64(subNumber),
		WorkflowRunID:    wr.ID,
		WorkflowNodeID:   n.ID,
		WorkflowNodeName: n.Name,
		Status:           string(sdk.StatusWaiting),
		Stages:           stages,
		Header:           wr.Header,
	}

	if run.SubNumber >= wr.LastSubNumber {
		wr.LastSubNumber = run.SubNumber
	}
	if n.Context.ApplicationID != 0 {
		run.ApplicationID = n.Context.ApplicationID
	}

	runPayload := map[string]string{}

	parentsIDs := make([]int64, len(parents))
	for i := range parents {
		parentsIDs[i] = parents[i].ID
	}

	parentStatus := sdk.StatusSuccess.String()
	run.SourceNodeRuns = parentsIDs
	if parents != nil {
		for _, p := range parents {
			for _, v := range wr.WorkflowNodeRuns {
				for _, run := range v {
					if p.ID == run.ID {
						if run.Status == sdk.StatusFail.String() || run.Status == sdk.StatusStopped.String() {
							parentStatus = run.Status
						}
					}
				}
			}
		}

		//Merge the payloads from all the sources
		_, next := observability.Span(ctx, "workflow.processNode.mergePayload")
		for _, r := range parents {
			e := dump.NewDefaultEncoder(new(bytes.Buffer))
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false
			m1, errm1 := e.ToStringMap(r.Payload)
			if errm1 != nil {
				AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
					ID:   sdk.MsgWorkflowError.ID,
					Args: []interface{}{errm1.Error()},
				})
				log.Error("processNode> Unable to compute hook payload: %v", errm1)
			}
			runPayload = sdk.ParametersMapMerge(runPayload, m1)
		}
		run.Payload = runPayload
		run.PipelineParameters = n.Context.DefaultPipelineParameters
		next()
	}

	run.HookEvent = hookEvent
	if hookEvent != nil {
		runPayload = sdk.ParametersMapMerge(runPayload, hookEvent.Payload)
		run.Payload = runPayload
		run.PipelineParameters = n.Context.DefaultPipelineParameters
	}

	run.BuildParameters = append(run.BuildParameters, sdk.Parameter{
		Name:  "cds.node",
		Type:  sdk.StringParameter,
		Value: run.WorkflowNodeName,
	})

	run.Manual = manual
	if manual != nil {
		e := dump.NewDefaultEncoder(new(bytes.Buffer))
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
		e.ExtraFields.DetailedMap = false
		e.ExtraFields.DetailedStruct = false
		e.ExtraFields.Len = false
		e.ExtraFields.Type = false
		m1, errm1 := e.ToStringMap(manual.Payload)
		if errm1 != nil {
			return report, false, sdk.WrapError(errm1, "r> Unable to compute payload")
		}
		runPayload = sdk.ParametersMapMerge(runPayload, m1)
		run.Payload = runPayload
		run.PipelineParameters = manual.PipelineParameters
		run.BuildParameters = append(run.BuildParameters, sdk.Parameter{
			Name:  "cds.triggered_by.email",
			Type:  sdk.StringParameter,
			Value: manual.User.Email,
		}, sdk.Parameter{
			Name:  "cds.triggered_by.fullname",
			Type:  sdk.StringParameter,
			Value: manual.User.Fullname,
		}, sdk.Parameter{
			Name:  "cds.triggered_by.username",
			Type:  sdk.StringParameter,
			Value: manual.User.Username,
		}, sdk.Parameter{
			Name:  "cds.manual",
			Type:  sdk.StringParameter,
			Value: "true",
		})
	} else {
		run.BuildParameters = append(run.BuildParameters, sdk.Parameter{
			Name:  "cds.manual",
			Type:  sdk.StringParameter,
			Value: "false",
		})
	}

	cdsStatusParam := sdk.Parameter{
		Name:  "cds.status",
		Type:  sdk.StringParameter,
		Value: parentStatus,
	}
	run.BuildParameters = sdk.ParametersFromMap(
		sdk.ParametersMapMerge(
			sdk.ParametersToMap(run.BuildParameters),
			sdk.ParametersToMap([]sdk.Parameter{cdsStatusParam}),
		),
	)

	// Process parameters for the jobs
	runContext := nodeRunContext{}

	if n.Context.PipelineID != 0 {
		runContext.Pipeline = wr.Workflow.Pipelines[n.Context.PipelineID]
	}
	if n.Context.ApplicationID != 0 {
		runContext.Application = wr.Workflow.Applications[n.Context.ApplicationID]
	}
	if n.Context.EnvironmentID != 0 {
		runContext.Environment = wr.Workflow.Environments[n.Context.EnvironmentID]
	}
	if n.Context.ProjectPlatformID != 0 {
		runContext.ProjectPlatform = wr.Workflow.ProjectPlatforms[n.Context.ProjectPlatformID]
	}
	jobParams, errParam := getNodeRunBuildParameters(ctx, proj, wr, run, runContext)
	if errParam != nil {
		AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{errParam.Error()},
		})
		// if there an error -> display it in workflowRunInfo and not stop the launch
		log.Error("processNode> getNodeRunBuildParameters failed. Project:%s [#%d.%d]%s.%d with payload %v err:%s", proj.Name, wr.Number, subNumber, wr.Workflow.Name, n.ID, run.Payload, errParam)
	}
	run.BuildParameters = append(run.BuildParameters, jobParams...)

	log.Debug("processNode> %+v", jobParams)
	log.Debug("processNode> %+v", run.BuildParameters)

	// Inherit parameter from parent job
	if len(parentsIDs) > 0 {
		_, next := observability.Span(ctx, "workflow.getParentParameters")
		parentsParams, errPP := getParentParameters(wr, parents, runPayload)
		next()
		if errPP != nil {
			return report, false, sdk.WrapError(errPP, "processNode> getParentParameters failed")
		}
		mapBuildParams := sdk.ParametersToMap(run.BuildParameters)
		mapParentParams := sdk.ParametersToMap(parentsParams)

		run.BuildParameters = sdk.ParametersFromMap(sdk.ParametersMapMerge(mapBuildParams, mapParentParams))
	}

	//Parse job params to get the VCS infos
	currentGitValues := map[string]string{}
	for _, param := range jobParams {
		switch param.Name {
		case tagGitHash, tagGitBranch, tagGitTag, tagGitAuthor, tagGitMessage, tagGitRepository, tagGitURL, tagGitHTTPURL:
			currentGitValues[param.Name] = param.Value
		}
	}

	//Parse job params to get the VCS infos
	previousGitValues := map[string]string{}
	for _, param := range run.BuildParameters {
		switch param.Name {
		case tagGitHash, tagGitBranch, tagGitTag, tagGitAuthor, tagGitMessage, tagGitRepository, tagGitURL, tagGitHTTPURL:
			previousGitValues[param.Name] = param.Value
		}
	}

	var isRoot bool
	if n.ID == wr.Workflow.Root.ID {
		isRoot = true
	}

	gitValues := currentGitValues
	if previousGitValues[tagGitURL] == currentGitValues[tagGitURL] || previousGitValues[tagGitHTTPURL] == currentGitValues[tagGitHTTPURL] {
		gitValues = previousGitValues
	}

	var vcsInfos vcsInfos

	if n.Context.ApplicationID != 0 {
		app := wr.Workflow.Applications[n.Context.ApplicationID]
		var errVcs error
		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
		vcsInfos, errVcs = getVCSInfos(ctx, db, store, vcsServer, gitValues, app.Name, app.VCSServer, app.RepositoryFullname, !isRoot, previousGitValues[tagGitRepository])
		if errVcs != nil {
			if strings.Contains(errVcs.Error(), "branch has been deleted") {
				AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
					ID:   sdk.MsgWorkflowRunBranchDeleted.ID,
					Args: []interface{}{vcsInfos.Branch},
				})
			} else {
				AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
					ID:   sdk.MsgWorkflowError.ID,
					Args: []interface{}{errVcs.Error()},
				})
			}
			if isRoot {
				return report, false, sdk.WrapError(errVcs, "processNode> Cannot get VCSInfos")
			}

			return nil, true, nil
		}
	}

	// only if it's the root pipeline, we put the git... in the build parameters
	// this allow user to write some run conditions with .git.var on the root pipeline
	if isRoot {
		log.Debug("before setValuesGitInBuildParameters> %+v", run.BuildParameters)
		log.Debug("before setValuesGitInBuildParameters> %+v", vcsInfos)
		setValuesGitInBuildParameters(run, vcsInfos)
		log.Debug("setValuesGitInBuildParameters> %+v", run.BuildParameters)
	}

	// Check Run Conditions
	if hookEvent != nil {
		hooks := wr.Workflow.WorkflowData.GetHooks()
		hook, ok := hooks[hookEvent.WorkflowNodeHookUUID]
		if !ok {
			return report, false, sdk.WrapError(sdk.ErrNoHook, "processNode> Unable to find hook %s", hookEvent.WorkflowNodeHookUUID)
		}

		// Check conditions
		var params = run.BuildParameters
		// Define specific destination parameters
		dest := mapNodes[hook.NodeID]
		if dest == nil {
			return report, false, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "processNode> Unable to find node %d", hook.NodeID)
		}

		if !checkNodeRunCondition(wr, dest.Context.Conditions, params) {
			log.Debug("processNode> Avoid trigger workflow from hook %s", hook.UUID)
			return report, false, nil
		}
	} else {
		if !checkNodeRunCondition(wr, n.Context.Conditions, run.BuildParameters) {
			log.Debug("processNode> Condition failed %d/%d %+v", wr.ID, n.ID, run.BuildParameters)
			return report, false, nil
		}
	}

	if !isRoot {
		setValuesGitInBuildParameters(run, vcsInfos)
	}

	// Tag VCS infos : add in tag only if it does not exist
	if !wr.TagExists(tagGitRepository) {
		wr.Tag(tagGitRepository, run.VCSRepository)
		if run.VCSBranch != "" && run.VCSTag == "" {
			wr.Tag(tagGitBranch, run.VCSBranch)
		}
		if run.VCSTag != "" {
			wr.Tag(tagGitTag, run.VCSTag)
		}
		if len(run.VCSHash) >= 7 {
			wr.Tag(tagGitHash, run.VCSHash[:7])
		} else {
			wr.Tag(tagGitHash, run.VCSHash)
		}
		wr.Tag(tagGitAuthor, vcsInfos.Author)
	}

	// Add env tag
	if n.Context.EnvironmentID != 0 {
		wr.Tag(tagEnvironment, wr.Workflow.Environments[n.Context.EnvironmentID].Name)
	}

	for _, info := range wr.Infos {
		if info.IsError && info.SubNumber == wr.LastSubNumber {
			run.Status = string(sdk.StatusFail)
			run.Done = time.Now()
			break
		}
	}

	if err := insertWorkflowNodeRun(db, run); err != nil {
		return report, false, sdk.WrapError(err, "processNode> unable to insert run (node id : %d, node name : %s, subnumber : %d)", run.WorkflowNodeID, run.WorkflowNodeName, run.SubNumber)
	}
	wr.LastExecution = time.Now()

	buildParameters := sdk.ParametersToMap(run.BuildParameters)
	_, okUI := buildParameters["cds.ui.pipeline.run"]
	_, okID := buildParameters["cds.node.id"]
	if !okUI || !okID {
		if !okUI {
			uiRunURL := fmt.Sprintf("%s/project/%s/workflow/%s/run/%s/node/%d?name=%s", baseUIURL, buildParameters["cds.project"], buildParameters["cds.workflow"], buildParameters["cds.run.number"], run.ID, buildParameters["cds.workflow"])
			sdk.AddParameter(&run.BuildParameters, "cds.ui.pipeline.run", sdk.StringParameter, uiRunURL)
		}
		if !okID {
			sdk.AddParameter(&run.BuildParameters, "cds.node.id", sdk.StringParameter, fmt.Sprintf("%d", run.ID))
		}

		if err := UpdateNodeRunBuildParameters(db, run.ID, run.BuildParameters); err != nil {
			return report, false, sdk.WrapError(err, "processNode> unable to update workflow node run build parameters")
		}
	}

	report.Add(*run)

	//Update workflow run
	if wr.WorkflowNodeRuns == nil {
		wr.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
	}
	wr.WorkflowNodeRuns[run.WorkflowNodeID] = append(wr.WorkflowNodeRuns[run.WorkflowNodeID], *run)
	wr.LastSubNumber = MaxSubNumber(wr.WorkflowNodeRuns)

	if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
		return report, false, sdk.WrapError(err, "processNode> unable to update workflow run")
	}

	//Check the context.mutex to know if we are allowed to run it
	if n.Context.Mutex {
		//Check if there are builing workflownoderun with the same workflow_node_name for the same workflow
		mutexQuery := `select count(1)
		from workflow_node_run
		join workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
		join workflow on workflow.id = workflow_run.workflow_id
		where workflow.id = $1
		and workflow_node_run.id <> $2
		and workflow_node_run.workflow_node_name = $3
		and workflow_node_run.status = $4`
		nbMutex, err := db.SelectInt(mutexQuery, n.WorkflowID, run.ID, n.Name, string(sdk.StatusBuilding))
		if err != nil {
			return report, false, sdk.WrapError(err, "processNode> unable to check mutexes")
		}
		if nbMutex > 0 {
			log.Debug("processNode> Noderun %s processed but not executed because of mutex", n.Name)
			AddWorkflowRunInfo(wr, false, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowNodeMutex.ID,
				Args: []interface{}{n.Name},
			})

			if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
				return report, false, sdk.WrapError(err, "processNode> unable to update workflow run")
			}

			//Mutex is locked. exit without error
			return report, false, nil
		}
		//Mutex is free, continue
	}

	//Execute the node run !
	r1, err := execute(ctx, db, store, proj, run, runContext)
	if err != nil {
		return report, false, sdk.WrapError(err, "processNode> unable to execute workflow run")
	}
	_, _ = report.Merge(r1, nil)
	return report, true, nil
}
