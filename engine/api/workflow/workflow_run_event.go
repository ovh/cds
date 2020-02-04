package workflow

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/fatih/structs"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ResyncCommitStatus resync commit status for a workflow run
func ResyncCommitStatus(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun) error {
	_, end := observability.Span(ctx, "workflow.resyncCommitStatus",
		observability.Tag(observability.TagWorkflow, wr.Workflow.Name),
		observability.Tag(observability.TagWorkflowRun, wr.Number),
	)
	defer end()

	for nodeID, nodeRuns := range wr.WorkflowNodeRuns {
		sort.Slice(nodeRuns, func(i, j int) bool {
			return nodeRuns[i].SubNumber >= nodeRuns[j].SubNumber
		})
		nodeRun := nodeRuns[0]

		if !sdk.StatusIsTerminated(nodeRun.Status) {
			continue
		}

		var vcsServerName string
		var repoFullName string

		node := wr.Workflow.WorkflowData.NodeByID(nodeID)
		if !node.IsLinkedToRepo(&wr.Workflow) {
			continue
		}
		vcsServerName = wr.Workflow.Applications[node.Context.ApplicationID].VCSServer
		repoFullName = wr.Workflow.Applications[node.Context.ApplicationID].RepositoryFullname

		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, vcsServerName)
		if vcsServer == nil {
			return nil
		}

		details := fmt.Sprintf("on project:%s workflow:%s node:%s num:%d sub:%d vcs:%s", proj.Name, wr.Workflow.Name, nodeRun.WorkflowNodeName, nodeRun.Number, nodeRun.SubNumber, vcsServer.Name)

		//Get the RepositoriesManager Client
		client, errClient := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
		if errClient != nil {
			return sdk.WrapError(errClient, "resyncCommitStatus> Cannot get client %s", details)
		}

		ref := nodeRun.VCSHash
		if nodeRun.VCSTag != "" {
			ref = nodeRun.VCSTag
		}

		statuses, errStatuses := client.ListStatuses(ctx, repoFullName, ref)
		if errStatuses != nil {
			return sdk.WrapError(errStatuses, "resyncCommitStatus> Cannot get statuses %s", details)
		}

		var statusFound *sdk.VCSCommitStatus
		expected := sdk.VCSCommitStatusDescription(proj.Key, wr.Workflow.Name, sdk.EventRunWorkflowNode{
			NodeName: nodeRun.WorkflowNodeName,
		})

		for i, status := range statuses {
			if status.Decription == expected {
				statusFound = &statuses[i]
				break
			}
		}

		if statusFound == nil || statusFound.State == "" {
			if err := sendVCSEventStatus(ctx, db, store, proj, wr, &nodeRun); err != nil {
				log.Error(ctx, "resyncCommitStatus> Error sending status %s err: %v", details, err)
			}

			if err := sendVCSPullRequestComment(ctx, db, store, proj, wr, &nodeRun); err != nil {
				log.Error(ctx, "resyncCommitStatus> Error sending pr comments %s %s err:%v", statusFound.State, details, err)
			}
			continue
		}

		skipStatus := false
		switch statusFound.State {
		case sdk.StatusSuccess:
			switch nodeRun.Status {
			case sdk.StatusSuccess:
				skipStatus = true
			}
		case sdk.StatusFail:
			switch nodeRun.Status {
			case sdk.StatusFail:
				skipStatus = true
			}

		case sdk.StatusSkipped:
			switch nodeRun.Status {
			case sdk.StatusDisabled, sdk.StatusNeverBuilt, sdk.StatusSkipped:
				skipStatus = true
			}
		}

		if !skipStatus {
			if err := sendVCSEventStatus(ctx, db, store, proj, wr, &nodeRun); err != nil {
				log.Error(ctx, "resyncCommitStatus> Error sending status %s %s err:%v", statusFound.State, details, err)
			}
		}

		if err := sendVCSPullRequestComment(ctx, db, store, proj, wr, &nodeRun); err != nil {
			log.Error(ctx, "resyncCommitStatus> Error sending pr comments %s %s err:%v", statusFound.State, details, err)
		}

	}
	return nil
}

// sendVCSEventStatus send status
func sendVCSEventStatus(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, nodeRun *sdk.WorkflowNodeRun) error {
	log.Debug("Send status for node run %d", nodeRun.ID)
	var app sdk.Application
	var pip sdk.Pipeline
	var env sdk.Environment
	node := wr.Workflow.WorkflowData.NodeByID(nodeRun.WorkflowNodeID)
	if !node.IsLinkedToRepo(&wr.Workflow) {
		return nil
	}

	app = wr.Workflow.Applications[node.Context.ApplicationID]
	if node.Context.PipelineID > 0 {
		pip = wr.Workflow.Pipelines[node.Context.PipelineID]
	}
	if node.Context.EnvironmentID > 0 {
		env = wr.Workflow.Environments[node.Context.EnvironmentID]
	}

	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
	if vcsServer == nil {
		return nil
	}

	//Get the RepositoriesManager Client
	client, errClient := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
	if errClient != nil {
		return sdk.WrapError(errClient, "sendVCSEventStatus> Cannot get client")
	}

	var eventWNR = sdk.EventRunWorkflowNode{
		ID:             nodeRun.ID,
		Number:         nodeRun.Number,
		SubNumber:      nodeRun.SubNumber,
		Status:         nodeRun.Status,
		Start:          nodeRun.Start.Unix(),
		Done:           nodeRun.Done.Unix(),
		Manual:         nodeRun.Manual,
		HookEvent:      nodeRun.HookEvent,
		Payload:        nodeRun.Payload,
		SourceNodeRuns: nodeRun.SourceNodeRuns,
		Hash:           nodeRun.VCSHash,
		Tag:            nodeRun.VCSTag,
		BranchName:     nodeRun.VCSBranch,
		NodeID:         nodeRun.WorkflowNodeID,
		RunID:          nodeRun.WorkflowRunID,
		StagesSummary:  make([]sdk.StageSummary, len(nodeRun.Stages)),
		NodeName:       nodeRun.WorkflowNodeName,
	}

	for i := range nodeRun.Stages {
		eventWNR.StagesSummary[i] = nodeRun.Stages[i].ToSummary()
	}

	var pipName, appName, envName string

	pipName = pip.Name
	appName = app.Name
	eventWNR.RepositoryManagerName = app.VCSServer
	eventWNR.RepositoryFullName = app.RepositoryFullname

	if env.Name != "" {
		envName = env.Name
	}

	report, err := nodeRun.Report()
	if err != nil {
		log.Error(ctx, "sendVCSEventStatus> unable to compute node run report%v", err)
		return nil
	}

	// Check if it's a gerrit or not
	vcsConf, err := repositoriesmanager.LoadByName(ctx, db, vcsServer.Name)
	if err != nil {
		return err
	}

	if vcsConf.Type == "gerrit" {
		// Get gerrit variable
		var project, changeID, branch, revision, url string
		projectParam := sdk.ParameterFind(nodeRun.BuildParameters, "git.repository")
		if projectParam != nil {
			project = projectParam.Value
		}
		changeIDParam := sdk.ParameterFind(nodeRun.BuildParameters, "gerrit.change.id")
		if changeIDParam != nil {
			changeID = changeIDParam.Value
		}
		branchParam := sdk.ParameterFind(nodeRun.BuildParameters, "gerrit.change.branch")
		if branchParam != nil {
			branch = branchParam.Value
		}
		revisionParams := sdk.ParameterFind(nodeRun.BuildParameters, "git.hash")
		if revisionParams != nil {
			revision = revisionParams.Value
		}
		urlParams := sdk.ParameterFind(nodeRun.BuildParameters, "cds.ui.pipeline.run")
		if urlParams != nil {
			url = urlParams.Value
		}
		eventWNR.GerritChange = &sdk.GerritChangeEvent{
			ID:         changeID,
			DestBranch: branch,
			Project:    project,
			Revision:   revision,
			Report:     report,
			URL:        url,
		}
	}

	evt := sdk.Event{
		EventType:       fmt.Sprintf("%T", eventWNR),
		Payload:         structs.Map(eventWNR),
		Timestamp:       time.Now(),
		ProjectKey:      proj.Key,
		WorkflowName:    wr.Workflow.Name,
		PipelineName:    pipName,
		ApplicationName: appName,
		EnvironmentName: envName,
	}

	if err := client.SetStatus(ctx, evt); err != nil {
		if err2 := repositoriesmanager.RetryEvent(&evt, err, store); err2 != nil {
			log.Error(ctx, "sendEvent>processEvent> err while retry event: %v", err2)
		}
		log.Error(ctx, "sendEvent> err:%v", err)
	}

	return nil
}

func sendVCSPullRequestComment(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, nodeRun *sdk.WorkflowNodeRun) error {
	log.Debug("Send pull-request comment for node run %d", nodeRun.ID)

	var app sdk.Application
	node := wr.Workflow.WorkflowData.NodeByID(nodeRun.WorkflowNodeID)
	if !node.IsLinkedToRepo(&wr.Workflow) {
		return nil
	}
	notif, errN := loadVCSNotificationWithNodeID(db, wr.WorkflowID, node.ID)
	if errN != nil {
		return sdk.WrapError(errN, "cannot load notification")
	}

	// vcs notification not enabled
	if notif.ID == 0 {
		return nil
	}

	if nodeRun.VCSReport == "" {
		nodeRun.VCSReport = notif.Settings.Template.Body
	}

	app = wr.Workflow.Applications[node.Context.ApplicationID]

	report, err := nodeRun.Report()
	if err != nil {
		log.Error(ctx, "sendVCSPullRequestComment> unable to compute node run report%v", err)
		return nil
	}

	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
	if vcsServer == nil {
		return nil
	}

	//Get the RepositoriesManager Client
	client, errClient := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
	if errClient != nil {
		return sdk.WrapError(errClient, "sendVCSPullRequestComment> Cannot get client")
	}

	// Check if it's a gerrit or not
	vcsConf, err := repositoriesmanager.LoadByName(ctx, db, vcsServer.Name)
	if err != nil {
		return err
	}

	if vcsConf.Type != "gerrit" && (notif.Settings.Template.DisableComment == nil || !*notif.Settings.Template.DisableComment) {
		//Check if this branch and this commit is a pullrequest
		prs, err := client.PullRequests(ctx, app.RepositoryFullname)
		if err != nil {
			log.Error(ctx, "sendVCSPullRequestComment> unable to get pull requests on repo %s: %v", app.RepositoryFullname, err)
			return nil
		}

		//Send comment on pull request
		if nodeRun.Status == sdk.StatusFail || nodeRun.Status == sdk.StatusStopped || notif.Settings.OnSuccess == sdk.UserNotificationAlways {
			for _, pr := range prs {
				if pr.Head.Branch.DisplayID == nodeRun.VCSBranch && pr.Head.Branch.LatestCommit == nodeRun.VCSHash && !pr.Merged && !pr.Closed {
					if err := client.PullRequestComment(ctx, app.RepositoryFullname, pr.ID, report); err != nil {
						log.Error(ctx, "sendVCSPullRequestComment> unable to send PR report:%v", err)
						return nil
					}
					// if we found the pull request for head branch we can break (only one PR for the branch should exist)
					break
				}
			}
		}
	}

	return nil
}
