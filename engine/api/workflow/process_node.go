package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func processNodeTriggers(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, parentNodeRun []*sdk.WorkflowNodeRun, node *sdk.Node, parentSubNumber int) (*ProcessorReport, error) {
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
				log.Error(ctx, "processWorkflowRun> Unable to process node ID=%d: %s", t.ChildNode.ID, errPwnr)
				AddWorkflowRunInfo(wr, sdk.SpawnMsgNew(*sdk.MsgWorkflowError, sdk.ExtractHTTPError(errPwnr)))
			}
			report.Merge(ctx, r1)
			continue
		}
	}
	return report, nil
}

func processNodeRun(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun,
	mapNodes map[int64]*sdk.Node, n *sdk.Node, subNumber int, parentNodeRuns []*sdk.WorkflowNodeRun,
	hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	report := new(ProcessorReport)
	exist, errN := nodeRunExist(db, wr.ID, n.ID, wr.Number, subNumber)
	if errN != nil {
		return nil, false, sdk.WrapError(errN, "processNodeRun> unable to check if node run exist")
	}
	if exist {
		return nil, false, nil
	}

	var end func()
	ctx, end = telemetry.Span(ctx, "workflow.processNodeRun",
		telemetry.Tag(telemetry.TagWorkflow, wr.Workflow.Name),
		telemetry.Tag(telemetry.TagWorkflowRun, wr.Number),
		telemetry.Tag(telemetry.TagWorkflowNode, n.Name),
	)
	defer end()

	// Keep old model behaviour on fork and join
	// Send manual event to join and fork children when it was a manual run and when fork and join don't have run condition
	if manual == nil && len(parentNodeRuns) == 1 && parentNodeRuns[0].Manual != nil {
		n := wr.Workflow.WorkflowData.NodeByID(parentNodeRuns[0].WorkflowNodeID)
		// If fork or JOIN and No run conditions
		if (n.Type == sdk.NodeTypeJoin || n.Type == sdk.NodeTypeFork) &&
			(n.Context == nil || (n.Context.Conditions.LuaScript == "" && len(n.Context.Conditions.PlainConditions) == 0)) {
			manual = parentNodeRuns[0].Manual
		}
	}

	switch n.Type {
	case sdk.NodeTypeFork, sdk.NodeTypePipeline, sdk.NodeTypeJoin:
		r1, conditionOK, errT := processNode(ctx, db, store, proj, wr, n, subNumber, parentNodeRuns, hookEvent, manual)
		if errT != nil {
			return nil, false, sdk.WrapError(errT, "Unable to processNode")
		}
		report.Merge(ctx, r1)
		return report, conditionOK, nil
	case sdk.NodeTypeOutGoingHook:
		r1, conditionOK, errO := processNodeOutGoingHook(ctx, db, store, proj, wr, mapNodes, parentNodeRuns, n, subNumber, manual)
		if errO != nil {
			return nil, false, sdk.WrapError(errO, "Unable to processNodeOutGoingHook")
		}
		report.Merge(ctx, r1)
		return report, conditionOK, nil
	}
	return nil, false, nil
}

func processNode(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun,
	n *sdk.Node, subNumber int, parents []*sdk.WorkflowNodeRun,
	hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	report := new(ProcessorReport)

	//TODO: Check user for manual done but check permission also for automatic trigger and hooks (with system to authenticate a webhook)
	if n.Context == nil {
		n.Context = &sdk.NodeContext{}
	}

	// CHECK NODE
	if n.Context.PipelineID == 0 && n.Type == sdk.NodeTypePipeline {
		return nil, false, sdk.ErrPipelineNotFound
	}

	nr := createWorkflowNodeRun(wr, n, parents, subNumber, hookEvent, manual)

	// PIPELINE PARAMETER
	if n.Type == sdk.NodeTypePipeline {
		nr.PipelineParameters = computePipelineParameters(wr, n, manual)
	}

	// PAYLOAD
	var errorPayload error
	nr.Payload, errorPayload = computePayload(n, hookEvent, manual)
	if errorPayload != nil {
		return nil, false, errorPayload
	}

	// WORKFLOW RUN BUILD PARAMETER
	var errBP error
	nr.BuildParameters, errBP = computeBuildParameters(wr, nr, parents, manual)
	if errBP != nil {
		return nil, false, errBP
	}

	// BUILD RUN CONTEXT
	// Process parameters for the jobs
	runContext := nodeRunContext{
		WorkflowProjectIntegrations: wr.Workflow.Integrations,
		ProjectIntegrations:         make([]sdk.ProjectIntegration, 0),
	}
	if n.Context.PipelineID != 0 {
		runContext.Pipeline = wr.Workflow.Pipelines[n.Context.PipelineID]
	}
	if n.Context.ApplicationID != 0 {
		runContext.Application = wr.Workflow.Applications[n.Context.ApplicationID]
	}
	if n.Context.EnvironmentID != 0 {
		runContext.Environment = wr.Workflow.Environments[n.Context.EnvironmentID]
	}
	if n.Context.ProjectIntegrationID != 0 {
		runContext.ProjectIntegrations = append(runContext.ProjectIntegrations, wr.Workflow.ProjectIntegrations[n.Context.ProjectIntegrationID])
	}
	runContext.WorkflowProjectIntegrations = append(runContext.WorkflowProjectIntegrations, wr.Workflow.Integrations...)

	// NODE CONTEXT BUILD PARAMETER
	computeNodeContextBuildParameters(ctx, proj, wr, nr, n, runContext)

	// PARENT BUILD PARAMETER WITH git.*
	if len(parents) > 0 {
		_, next := telemetry.Span(ctx, "workflow.getParentParameters")
		parentsParams, errPP := getParentParameters(wr, parents)
		next()
		if errPP != nil {
			return nil, false, sdk.WrapError(errPP, "processNode> getParentParameters failed")
		}
		mapBuildParams := sdk.ParametersToMap(nr.BuildParameters)
		mapParentParams := sdk.ParametersToMap(parentsParams)

		nr.BuildParameters = sdk.ParametersFromMap(sdk.ParametersMapMerge(mapBuildParams, mapParentParams))
	}

	// If we rerun only failed job, retrieve cds.build variables from previous jobs
	if nr.Manual != nil && nr.Manual.OnlyFailedJobs {
		previousNodeRun, err := checkRunOnlyFailedJobs(wr, nr)
		if err != nil {
			return nil, false, err
		}
		for _, bp := range previousNodeRun.BuildParameters {
			if strings.HasPrefix(bp.Name, "cds.build.") {
				nr.BuildParameters = append(nr.BuildParameters, bp)
			}
		}
	}

	isRoot := n.ID == wr.Workflow.WorkflowData.Node.ID

	// GIT PARAMS
	// Here, nr.BuildParameters contains parent git params, get from getParentParameters

	var app sdk.Application
	var currentRepo string
	currentJobGitValues := map[string]string{}
	needVCSInfo := false

	// Get current application and repository
	if n.Context.ApplicationID != 0 {
		app = wr.Workflow.Applications[n.Context.ApplicationID]
		currentRepo = app.RepositoryFullname
	}

	parentRepo := sdk.ParameterFind(nr.BuildParameters, tagGitRepository)

	// Compute git params for current job
	// Get from parent when
	// * is root because they come from payload
	// * no repo on current job
	// * parent was on same repo
	if isRoot || currentRepo == "" || (parentRepo != nil && parentRepo.Value == currentRepo) {
		for _, param := range nr.BuildParameters {
			switch param.Name {
			case tagGitHash, tagGitBranch, tagGitTag, tagGitAuthor, tagGitMessage, tagGitRepository, tagGitURL, tagGitHTTPURL, tagGitServer:
				currentJobGitValues[param.Name] = param.Value
			}
		}
		if isRoot {
			needVCSInfo = true
		}
	}
	// Find an ancestor on the same repo
	if currentRepo != "" && parentRepo == nil {
		needVCSInfo = true
	} else if currentRepo != "" && parentRepo != nil && parentRepo.Value != currentRepo {
		// Try to found a parent on the same repo
		found := false
		for _, parent := range wr.WorkflowNodeRuns {
			repo := sdk.ParameterFind(parent[0].BuildParameters, tagGitRepository)
			// if same repo OR not same repo, and same application, but different repo, it's a fork
			if repo != nil && (repo.Value == currentRepo || parent[0].ApplicationID == n.Context.ApplicationID) {
				found = true
				// copy git info from ancestors
				for _, param := range parent[0].BuildParameters {
					switch param.Name {
					case tagGitHash, tagGitBranch, tagGitTag, tagGitAuthor, tagGitMessage, tagGitRepository, tagGitURL, tagGitHTTPURL, tagGitServer:
						currentJobGitValues[param.Name] = param.Value
					}
				}
				break
			}
		}
		// If we change repo and we dont find ancestor on the same repo, just keep the branch
		if !found {
			b := sdk.ParameterFind(nr.BuildParameters, tagGitBranch)
			if b != nil {
				currentJobGitValues[tagGitBranch] = b.Value
			}
			needVCSInfo = true
		}
	}

	// GET VCS Infos IF NEEDED
	// * root Node
	// * different repo
	var vcsInf *vcsInfos
	var errVcs error
	if needVCSInfo && app.VCSServer != "" {
		// We can't have both git.branch and git.tag values
		if currentJobGitValues["git.branch"] != "" && currentJobGitValues["git.tag"] != "" {
			return nil, false, sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("invalid git variables"))
		}

		vcsInf, errVcs = getVCSInfos(ctx, db, store, proj.Key, currentJobGitValues, app.Name, app.VCSServer, app.RepositoryFullname)
		if errVcs != nil {
			AddWorkflowRunInfo(wr, sdk.SpawnMsgNew(*sdk.MsgWorkflowError, sdk.ExtractHTTPError(errVcs)))
			return nil, false, sdk.WrapError(errVcs, "unable to get git informations")
		}
	}

	// Replace ("{{ }}" in vcsInfo that should be badly interpreted by interpolation engine)
	var repl = func(s *string) {
		*s = strings.ReplaceAll(*s, "{{", "((")
		*s = strings.ReplaceAll(*s, "}}", "))")
	}

	// Update git params / git columns
	if isRoot && vcsInf != nil {
		repl(&vcsInf.Author)
		repl(&vcsInf.Branch)
		repl(&vcsInf.Hash)
		repl(&vcsInf.Message)
		repl(&vcsInf.Tag)

		setValuesGitInBuildParameters(nr, runContext, *vcsInf)
	}

	// CONDITION
	if !checkCondition(ctx, wr, n.Context.Conditions, nr.BuildParameters) {
		log.Debug(ctx, "Conditions failed on processNode %d/%d", wr.ID, n.ID)
		log.Debug(ctx, "Conditions was: %+v", n.Context.Conditions)
		log.Debug(ctx, "BuildParameters was: %+v", nr.BuildParameters)
		return nil, false, nil
	}

	// Resync vcsInfos if we dont call func getVCSInfos
	if !needVCSInfo {
		vcsInf = &vcsInfos{}
		vcsInf.Repository = currentJobGitValues[tagGitRepository]
		vcsInf.Branch = currentJobGitValues[tagGitBranch]
		vcsInf.Tag = currentJobGitValues[tagGitTag]
		vcsInf.Hash = currentJobGitValues[tagGitHash]
		vcsInf.Author = currentJobGitValues[tagGitAuthor]
		vcsInf.Message = currentJobGitValues[tagGitMessage]
		vcsInf.URL = currentJobGitValues[tagGitURL]
		vcsInf.HTTPUrl = currentJobGitValues[tagGitHTTPURL]
		vcsInf.Server = currentJobGitValues[tagGitServer]
	}

	// Update datas if repo change
	if !isRoot && vcsInf != nil {
		repl(&vcsInf.Author)
		repl(&vcsInf.Branch)
		repl(&vcsInf.Hash)
		repl(&vcsInf.Message)
		repl(&vcsInf.Tag)

		setValuesGitInBuildParameters(nr, runContext, *vcsInf)
	}

	// ADD TAG
	// Tag VCS infos : add in tag only if it does not exist
	if !wr.TagExists(tagGitRepository) {
		wr.Tag(tagGitRepository, nr.VCSRepository)
		if nr.VCSBranch != "" && nr.VCSTag == "" {
			wr.Tag(tagGitBranch, nr.VCSBranch)
		}
		if nr.VCSTag != "" {
			wr.Tag(tagGitTag, nr.VCSTag)
		}
		wr.Tag(tagGitHash, nr.VCSHash)
		if vcsInf != nil {
			wr.Tag(tagGitAuthor, vcsInf.Author)
		}
	}

	// Add env tag
	if n.Context.EnvironmentID != 0 {
		wr.Tag(tagEnvironment, wr.Workflow.Environments[n.Context.EnvironmentID].Name)
	}

	for _, info := range wr.Infos {
		if info.Type == sdk.RunInfoTypeError && info.SubNumber == wr.LastSubNumber {
			nr.Status = sdk.StatusFail
			nr.Done = time.Now()
			break
		}
	}

	if err := insertWorkflowNodeRun(db, nr); err != nil {
		return nil, false, sdk.WrapError(err, "unable to insert run (node id : %d, node name : %s, subnumber : %d)", nr.WorkflowNodeID, nr.WorkflowNodeName, nr.SubNumber)
	}
	wr.LastExecution = time.Now()

	buildParameters := sdk.ParametersToMap(nr.BuildParameters)
	_, okUI := buildParameters["cds.ui.pipeline.run"]
	_, okID := buildParameters["cds.node.id"]
	if !okUI || !okID {
		if !okUI {
			uiRunURL := fmt.Sprintf("%s/project/%s/workflow/%s/run/%s/node/%d?name=%s", baseUIURL, buildParameters["cds.project"], buildParameters["cds.workflow"], buildParameters["cds.run.number"], nr.ID, n.Name)
			sdk.AddParameter(&nr.BuildParameters, "cds.ui.pipeline.run", sdk.StringParameter, uiRunURL)
		}
		if !okID {
			sdk.AddParameter(&nr.BuildParameters, "cds.node.id", sdk.StringParameter, fmt.Sprintf("%d", nr.ID))
		}

		if err := UpdateNodeRunBuildParameters(db, nr.ID, nr.BuildParameters); err != nil {
			return nil, false, sdk.WrapError(err, "unable to update workflow node run build parameters")
		}
	}

	report.Add(ctx, *nr)

	//Update workflow run
	if wr.WorkflowNodeRuns == nil {
		wr.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
	}
	wr.WorkflowNodeRuns[nr.WorkflowNodeID] = append(wr.WorkflowNodeRuns[nr.WorkflowNodeID], *nr)
	wr.LastSubNumber = MaxSubNumber(wr.WorkflowNodeRuns)

	if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
		return nil, false, sdk.WrapError(err, "unable to update workflow run")
	}

	//Check the context.mutex to know if we are allowed to run it
	if n.Context.Mutex {
		//Check if there are previous waiting or builing workflownoderun
		// with the same workflow_node_name for the same workflow

		// in this sql, we use 'and workflow_node_run.id < $2' and not and workflow_node_run.id <> $2
		// we check if there is a previous build in waiting status
		// and or if there is another build (never or not) with building status
		mutexQuery := `select count(1)
		from workflow_node_run
		join workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
		join workflow on workflow.id = workflow_run.workflow_id
		where workflow.id = $1
		and workflow_node_run.workflow_node_name = $3
		and (
			(workflow_node_run.id < $2 and workflow_node_run.status = $4)
			or
			(workflow_node_run.id <> $2 and workflow_node_run.status = $5)
		)`
		nbMutex, err := db.SelectInt(mutexQuery, n.WorkflowID, nr.ID, n.Name, sdk.StatusWaiting, sdk.StatusBuilding)
		if err != nil {
			return nil, false, sdk.WrapError(err, "unable to check mutexes")
		}
		if nbMutex > 0 {
			log.Debug(ctx, "Noderun %s processed but not executed because of mutex", n.Name)
			AddWorkflowRunInfo(wr, sdk.SpawnMsgNew(*sdk.MsgWorkflowNodeMutex, n.Name))
			if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
				return nil, false, sdk.WrapError(err, "unable to update workflow run")
			}

			// Mutex is locked, but it is as the workflow is ok to be run (conditions ok).
			// it's ok exit without error
			return report, true, nil
		}
		//Mutex is free, continue
	}

	//Execute the node run !
	r1, err := executeNodeRun(ctx, db, store, proj, nr)
	if err != nil {
		return nil, false, sdk.WrapError(err, "unable to execute workflow run")
	}
	report.Merge(ctx, r1)
	return report, true, nil
}

func getParentsStatus(wr *sdk.WorkflowRun, parents []*sdk.WorkflowNodeRun) string {
	for _, p := range parents {
		for _, v := range wr.WorkflowNodeRuns {
			for _, run := range v {
				if p.ID == run.ID {
					if run.Status == sdk.StatusFail || run.Status == sdk.StatusStopped {
						return run.Status
					}
				}
			}
		}
	}
	return sdk.StatusSuccess
}

func createWorkflowNodeRun(wr *sdk.WorkflowRun, n *sdk.Node, parents []*sdk.WorkflowNodeRun, subNumber int, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) *sdk.WorkflowNodeRun {
	/// GET PIPELINE STAGE + PIP PARAMETER IF NEED
	var stages []sdk.Stage
	var pip sdk.Pipeline
	if n.Context.PipelineID > 0 {
		pip = wr.Workflow.Pipelines[n.Context.PipelineID]
		stages = make([]sdk.Stage, len(pip.Stages))
		copy(stages, pip.Stages)
	}

	// CREATE RUN
	nodeRun := sdk.WorkflowNodeRun{
		WorkflowID:       wr.WorkflowID,
		LastModified:     time.Now(),
		Start:            time.Now(),
		Number:           wr.Number,
		SubNumber:        int64(subNumber),
		WorkflowRunID:    wr.ID,
		WorkflowNodeID:   n.ID,
		WorkflowNodeName: n.Name,
		Status:           sdk.StatusWaiting,
		Stages:           stages,
		Header:           wr.Header,
	}

	if nodeRun.SubNumber >= wr.LastSubNumber {
		wr.LastSubNumber = nodeRun.SubNumber
	}
	if n.Context.ApplicationID != 0 {
		nodeRun.ApplicationID = n.Context.ApplicationID
	}

	parentsIDs := make([]int64, len(parents))
	for i := range parents {
		parentsIDs[i] = parents[i].ID
	}
	nodeRun.SourceNodeRuns = parentsIDs
	nodeRun.HookEvent = hookEvent
	nodeRun.Manual = manual

	return &nodeRun
}

func computePipelineParameters(wr *sdk.WorkflowRun, n *sdk.Node, manual *sdk.WorkflowNodeRunManual) []sdk.Parameter {
	pip := wr.Workflow.Pipelines[n.Context.PipelineID]

	pipParams := sdk.ParametersMerge(pip.Parameter, n.Context.DefaultPipelineParameters)

	if manual != nil && len(manual.PipelineParameters) > 0 {
		pipParams = sdk.ParametersMerge(pipParams, manual.PipelineParameters)
	}

	// Take first value in pipeline parameter list if no default value is set
	for i := range pipParams {
		if pipParams[i].Type == sdk.ListParameter && strings.Contains(pipParams[i].Value, ";") {
			pipParams[i].Value = strings.Split(pipParams[i].Value, ";")[0]
		}
	}
	return pipParams
}

func computePayload(n *sdk.Node, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (interface{}, error) {
	switch {
	case hookEvent != nil && hookEvent.Payload != nil:
		return hookEvent.Payload, nil
	case manual != nil && manual.Payload != nil:
		return manual.Payload, nil
	default:
		return n.Context.DefaultPayloadToMap()
	}
}

func computeNodeContextBuildParameters(ctx context.Context, proj sdk.Project, wr *sdk.WorkflowRun, run *sdk.WorkflowNodeRun, n *sdk.Node, runContext nodeRunContext) {
	allContexts := sdk.NodeRunContext{}

	nodeRunParams, varsContext, errParam := getNodeRunBuildParameters(ctx, proj, wr, run, runContext)
	if errParam != nil {
		AddWorkflowRunInfo(wr, sdk.SpawnMsgNew(*sdk.MsgWorkflowError, sdk.ExtractHTTPError(errParam)))
		// if there an error -> display it in workflowRunInfo and not stop the launch
		log.Error(ctx, "processNode> getNodeRunBuildParameters failed. Project:%s [#%d.%d]%s.%d with payload %v err:%v", proj.Name, wr.Number, run.SubNumber, wr.Workflow.Name, n.ID, run.Payload, errParam)
	}
	run.BuildParameters = append(run.BuildParameters, nodeRunParams...)

	allContexts.Vars = varsContext
	allContexts.CDS = computeCDSContext(ctx, *wr, *run)
	run.Contexts = allContexts
}

func computeBuildParameters(wr *sdk.WorkflowRun, run *sdk.WorkflowNodeRun, parents []*sdk.WorkflowNodeRun, manual *sdk.WorkflowNodeRunManual) ([]sdk.Parameter, error) {
	params := make([]sdk.Parameter, 0, 1)

	params = append(params, sdk.Parameter{
		Name:  "cds.manual",
		Type:  sdk.StringParameter,
		Value: fmt.Sprintf("%v", manual != nil),
	})

	// ADD NODE NAME
	params = append(params, sdk.Parameter{
		Name:  "cds.node",
		Type:  sdk.StringParameter,
		Value: run.WorkflowNodeName,
	})

	// ADD PAYLOAD as STRING only for manual run
	if run.Payload != nil && run.Manual != nil {
		payloadStr, err := json.Marshal(run.Payload)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to marshal payload")
		}
		params = append(params, sdk.Parameter{
			Name:  "payload",
			Type:  sdk.TextParameter,
			Value: string(payloadStr),
		})

	}

	// ADD PARENT STATUS
	cdsStatusParam := sdk.Parameter{
		Name:  "cds.status",
		Type:  sdk.StringParameter,
		Value: getParentsStatus(wr, parents),
	}
	params = sdk.ParametersFromMap(
		sdk.ParametersMapMerge(
			sdk.ParametersToMap(params),
			sdk.ParametersToMap([]sdk.Parameter{cdsStatusParam}),
		),
	)

	// MANUAL BUILD PARAMETER
	if manual != nil {
		params = append(params, sdk.Parameter{
			Name:  "cds.triggered_by.email",
			Type:  sdk.StringParameter,
			Value: manual.Email,
		}, sdk.Parameter{
			Name:  "cds.triggered_by.fullname",
			Type:  sdk.StringParameter,
			Value: manual.Fullname,
		}, sdk.Parameter{
			Name:  "cds.triggered_by.username",
			Type:  sdk.StringParameter,
			Value: manual.Username,
		}, sdk.Parameter{
			Name:  "cds.manual",
			Type:  sdk.StringParameter,
			Value: "true",
		})
	}

	return params, nil
}
