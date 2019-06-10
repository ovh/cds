package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	exist, errN := nodeRunExist(db, wr.ID, n.ID, wr.Number, subNumber)
	if errN != nil {
		return nil, false, sdk.WrapError(errN, "processNodeRun> unable to check if node run exist")
	}
	if exist {
		return nil, false, nil
	}

	var end func()
	ctx, end = observability.Span(ctx, "workflow.processNodeRun",
		observability.Tag(observability.TagWorkflow, wr.Workflow.Name),
		observability.Tag(observability.TagWorkflowRun, wr.Number),
		observability.Tag(observability.TagWorkflowNode, n.Name),
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
		r1, conditionOK, errT := processNode(ctx, db, store, proj, wr, mapNodes, n, subNumber, parentNodeRuns, hookEvent, manual)
		if errT != nil {
			return nil, false, sdk.WrapError(errT, "Unable to processNode")
		}
		report.Merge(r1, nil) // nolint
		return report, conditionOK, nil
	case sdk.NodeTypeOutGoingHook:
		r1, conditionOK, errO := processNodeOutGoingHook(ctx, db, store, proj, wr, mapNodes, parentNodeRuns, n, subNumber)
		if errO != nil {
			return nil, false, sdk.WrapError(errO, "Unable to processNodeOutGoingHook")
		}
		report.Merge(r1, nil) // nolint
		return report, conditionOK, nil
	}
	return nil, false, nil
}

func processNode(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, n *sdk.Node, subNumber int, parents []*sdk.WorkflowNodeRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	report := new(ProcessorReport)

	//TODO: Check user for manual done but check permission also for automatic trigger and hooks (with system to authenticate a webhook)
	if n.Context == nil {
		n.Context = &sdk.NodeContext{}
	}

	// CHECK NODE
	if n.Context.PipelineID == 0 && n.Type == sdk.NodeTypePipeline {
		return nil, false, sdk.ErrPipelineNotFound
	}

	run := createWorkflowNodeRun(wr, n, parents, subNumber, hookEvent, manual)

	// PIPELINE PARAMETER
	if n.Type == sdk.NodeTypePipeline {
		run.PipelineParameters = computePipelineParameters(wr, n, manual)
	}

	// PAYLOAD
	var errorPayload error
	run.Payload, errorPayload = computePayload(n, hookEvent, manual)
	if errorPayload != nil {
		return nil, false, errorPayload
	}

	// WORKFLOW RUN BUILD PARAMETER
	var errBP error
	run.BuildParameters, errBP = computeBuildParameters(wr, run, parents, manual)
	if errBP != nil {
		return nil, false, errBP
	}

	// BUILD RUN CONTEXT
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
	if n.Context.ProjectIntegrationID != 0 {
		runContext.ProjectIntegration = wr.Workflow.ProjectIntegrations[n.Context.ProjectIntegrationID]
	}

	// NODE CONTEXT BUILD PARAMETER
	computeNodeContextBuildParameters(ctx, proj, wr, run, n, runContext)

	// PARENT BUILD PARAMETER WITH git.*
	if len(parents) > 0 {
		_, next := observability.Span(ctx, "workflow.getParentParameters")
		parentsParams, errPP := getParentParameters(wr, parents)
		next()
		if errPP != nil {
			return nil, false, sdk.WrapError(errPP, "processNode> getParentParameters failed")
		}
		mapBuildParams := sdk.ParametersToMap(run.BuildParameters)
		mapParentParams := sdk.ParametersToMap(parentsParams)

		run.BuildParameters = sdk.ParametersFromMap(sdk.ParametersMapMerge(mapBuildParams, mapParentParams))
	}

	isRoot := n.ID == wr.Workflow.WorkflowData.Node.ID

	// GIT PARAMS
	// Here, run.BuildParameters contains parent git params, get from getParentParameters

	var app sdk.Application
	var currentRepo string
	currentJobGitValues := map[string]string{}
	needVCSInfo := false

	// Get current application and repository
	if n.Context.ApplicationID != 0 {
		app = wr.Workflow.Applications[n.Context.ApplicationID]
		currentRepo = app.RepositoryFullname
	}

	parentRepo := sdk.ParameterFind(&run.BuildParameters, tagGitRepository)

	// Compute git params for current job
	// Get from parent when
	// * is root because they come from payload
	// * no repo on current job
	// * parent was on same repo
	if isRoot || currentRepo == "" || (parentRepo != nil && parentRepo.Value == currentRepo) {
		for _, param := range run.BuildParameters {
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
	if currentRepo != "" && parentRepo != nil && parentRepo.Value != currentRepo {
		// Try to found a parent on the same repo
		found := false
		for _, parent := range wr.WorkflowNodeRuns {
			repo := sdk.ParameterFind(&parent[0].BuildParameters, tagGitRepository)
			if repo != nil && repo.Value == currentRepo {
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
			b := sdk.ParameterFind(&run.BuildParameters, tagGitBranch)
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
	if needVCSInfo {
		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
		vcsInf, errVcs = getVCSInfos(ctx, db, store, vcsServer, currentJobGitValues, app.Name, app.VCSServer, app.RepositoryFullname)
		if errVcs != nil {
			AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowError.ID,
				Args: []interface{}{errVcs.Error()},
			})
			return nil, false, sdk.WrapError(errVcs, "unable to get git informations")
		}
	}

	// Update git params / git columns
	if isRoot && vcsInf != nil {
		setValuesGitInBuildParameters(run, *vcsInf)
	}

	// CONDITION
	if hookEvent != nil {
		hooks := wr.Workflow.WorkflowData.GetHooks()
		hook, ok := hooks[hookEvent.WorkflowNodeHookUUID]
		if !ok {
			return nil, false, sdk.WrapError(sdk.ErrNoHook, "Unable to find hook %s", hookEvent.WorkflowNodeHookUUID)
		}

		// Check conditions
		var params = run.BuildParameters
		// Define specific destination parameters
		dest := mapNodes[hook.NodeID]
		if dest == nil {
			return nil, false, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "Unable to find node %d", hook.NodeID)
		}

		if !checkCondition(wr, dest.Context.Conditions, params) {
			log.Debug("Avoid trigger workflow from hook %s", hook.UUID)
			return nil, false, nil
		}
	} else {
		if !checkCondition(wr, n.Context.Conditions, run.BuildParameters) {
			log.Debug("Condition failed %d/%d %+v", wr.ID, n.ID, run.BuildParameters)
			return nil, false, nil
		}
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
		setValuesGitInBuildParameters(run, *vcsInf)
	}

	// ADD TAG
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
		wr.Tag(tagGitAuthor, vcsInf.Author)
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
		return nil, false, sdk.WrapError(err, "unable to insert run (node id : %d, node name : %s, subnumber : %d)", run.WorkflowNodeID, run.WorkflowNodeName, run.SubNumber)
	}
	wr.LastExecution = time.Now()

	buildParameters := sdk.ParametersToMap(run.BuildParameters)
	_, okUI := buildParameters["cds.ui.pipeline.run"]
	_, okID := buildParameters["cds.node.id"]
	if !okUI || !okID {
		if !okUI {
			uiRunURL := fmt.Sprintf("%s/project/%s/workflow/%s/run/%s/node/%d?name=%s", baseUIURL, buildParameters["cds.project"], buildParameters["cds.workflow"], buildParameters["cds.run.number"], run.ID, n.Name)
			sdk.AddParameter(&run.BuildParameters, "cds.ui.pipeline.run", sdk.StringParameter, uiRunURL)
		}
		if !okID {
			sdk.AddParameter(&run.BuildParameters, "cds.node.id", sdk.StringParameter, fmt.Sprintf("%d", run.ID))
		}

		if err := UpdateNodeRunBuildParameters(db, run.ID, run.BuildParameters); err != nil {
			return nil, false, sdk.WrapError(err, "unable to update workflow node run build parameters")
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
		return nil, false, sdk.WrapError(err, "unable to update workflow run")
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
			return nil, false, sdk.WrapError(err, "unable to check mutexes")
		}
		if nbMutex > 0 {
			log.Debug("Noderun %s processed but not executed because of mutex", n.Name)
			AddWorkflowRunInfo(wr, false, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowNodeMutex.ID,
				Args: []interface{}{n.Name},
			})

			if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
				return nil, false, sdk.WrapError(err, "unable to update workflow run")
			}

			//Mutex is locked. exit without error
			return report, false, nil
		}
		//Mutex is free, continue
	}

	//Execute the node run !
	r1, err := execute(ctx, db, store, proj, run, runContext)
	if err != nil {
		return nil, false, sdk.WrapError(err, "unable to execute workflow run")
	}
	_, _ = report.Merge(r1, nil)
	return report, true, nil
}

func getParentsStatus(wr *sdk.WorkflowRun, parents []*sdk.WorkflowNodeRun) string {
	for _, p := range parents {
		for _, v := range wr.WorkflowNodeRuns {
			for _, run := range v {
				if p.ID == run.ID {
					if run.Status == sdk.StatusFail.String() || run.Status == sdk.StatusStopped.String() {
						return run.Status
					}
				}
			}
		}
	}
	return sdk.StatusSuccess.String()
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
	run := sdk.WorkflowNodeRun{
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

	parentsIDs := make([]int64, len(parents))
	for i := range parents {
		parentsIDs[i] = parents[i].ID
	}
	run.SourceNodeRuns = parentsIDs
	run.HookEvent = hookEvent
	run.Manual = manual
	return &run
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

func computeNodeContextBuildParameters(ctx context.Context, proj *sdk.Project, wr *sdk.WorkflowRun, run *sdk.WorkflowNodeRun, n *sdk.Node, runContext nodeRunContext) {
	nodeRunParams, errParam := getNodeRunBuildParameters(ctx, proj, wr, run, runContext)
	if errParam != nil {
		AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{errParam.Error()},
		})
		// if there an error -> display it in workflowRunInfo and not stop the launch
		log.Error("processNode> getNodeRunBuildParameters failed. Project:%s [#%d.%d]%s.%d with payload %v err:%v", proj.Name, wr.Number, run.SubNumber, wr.Workflow.Name, n.ID, run.Payload, errParam)
	}
	run.BuildParameters = append(run.BuildParameters, nodeRunParams...)
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

	// ADD PAYLOAD as STRING
	if run.Payload != nil {
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
	}

	return params, nil
}
