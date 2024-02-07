package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

type VCSEventMessenger struct {
	commitsStatuses map[string][]sdk.VCSCommitStatus
	vcsClient       sdk.VCSAuthorizedClientService
}

// ResyncCommitStatus resync commit status for a workflow run
func ResyncCommitStatus(ctx context.Context, db *gorp.DbMap, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, cdsUIURL string) error {
	_, end := telemetry.Span(ctx, "workflow.resyncCommitStatus",
		telemetry.Tag(telemetry.TagWorkflow, wr.Workflow.Name),
		telemetry.Tag(telemetry.TagWorkflowRun, wr.Number),
	)
	defer end()

	eventMessenger := &VCSEventMessenger{commitsStatuses: make(map[string][]sdk.VCSCommitStatus)}
	for _, nodeRuns := range wr.WorkflowNodeRuns {
		sort.Slice(nodeRuns, func(i, j int) bool {
			return nodeRuns[i].SubNumber >= nodeRuns[j].SubNumber
		})
		nodeRun := nodeRuns[0]

		if err := eventMessenger.SendVCSEvent(ctx, db, store, proj, *wr, nodeRun, cdsUIURL); err != nil {
			log.Error(ctx, "resyncCommitStatus > unable to send vcs event: %v", err)
		}
	}

	return nil
}

func (e *VCSEventMessenger) SendVCSEvent(ctx context.Context, db *gorp.DbMap, store cache.Store, proj sdk.Project, wr sdk.WorkflowRun, nodeRun sdk.WorkflowNodeRun, cdsUIURL string) error {
	ctx = context.WithValue(ctx, cdslog.NodeRunID, nodeRun.ID)

	if nodeRun.Status == sdk.StatusWaiting {
		return nil
	}

	log.Info(ctx, "sending VCS event for status = %q", nodeRun.Status)

	if e.commitsStatuses == nil {
		e.commitsStatuses = make(map[string][]sdk.VCSCommitStatus)
	}

	node := wr.Workflow.WorkflowData.NodeByID(nodeRun.WorkflowNodeID)
	if !node.IsLinkedToRepo(&wr.Workflow) {
		log.Info(ctx, "node is not linked to the repo, skipping")
		return nil
	}

	var notifs []sdk.WorkflowNotification
	// browse notification to find vcs one
	for _, n := range wr.Workflow.Notifications {
		if n.Type != sdk.VCSUserNotification {
			continue
		}
		// If list of node is nill, send notification to all of them
		if len(n.NodeIDs) == 0 {
			notifs = append(notifs, n)
			continue
		}
		// browser source node id
		for _, src := range n.NodeIDs {
			if src == node.ID {
				notifs = append(notifs, n)
				break
			}
		}
	}

	if len(notifs) == 0 {
		log.Info(ctx, "no vcs notification set in the node, skipping")
		return nil
	}

	vcsServerName := wr.Workflow.Applications[node.Context.ApplicationID].VCSServer
	repoFullName := wr.Workflow.Applications[node.Context.ApplicationID].RepositoryFullname

	//Get the RepositoriesManager Client
	if e.vcsClient == nil {
		var err error
		e.vcsClient, err = repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServerName)
		if err != nil {
			return sdk.WrapError(err, "can't get AuthorizedClient for %v/%v", proj.Key, vcsServerName)
		}
	}

	ref := nodeRun.VCSHash
	if nodeRun.VCSTag != "" {
		ref = nodeRun.VCSTag
	}

	statuses, ok := e.commitsStatuses[ref]
	if !ok {
		var err error
		log.Info(ctx, "getting status for %s %s", repoFullName, ref)
		statuses, err = e.vcsClient.ListStatuses(ctx, repoFullName, ref)
		if err != nil {
			return sdk.WrapError(err, "can't ListStatuses for %v with vcs %v/%v", repoFullName, proj.Key, vcsServerName)
		}
		e.commitsStatuses[ref] = statuses
	}
	expected := sdk.VCSCommitStatusDescription(proj.Key, wr.Workflow.Name, sdk.EventRunWorkflowNode{
		NodeName: nodeRun.WorkflowNodeName,
	})
	log.Info(ctx, "expected status description is %q", expected)

	if e.vcsClient.IsBitbucketCloud() {
		if len(expected) > 36 { // 40 maxlength on bitbucket cloud
			expected = expected[:36]
		}
	}

	var statusFound *sdk.VCSCommitStatus
	for i, status := range statuses {
		if status.Decription == expected {
			statusFound = &statuses[i]
			break
		}
	}

	if statusFound == nil || statusFound.State == "" {
		for i := range notifs {
			log.Info(ctx, "status %q %s not found, sending a new one %+v", expected, nodeRun.Status, notifs[i])
			if err := e.sendVCSEventStatus(ctx, db, store, proj.Key, wr, &nodeRun, notifs[i], vcsServerName, cdsUIURL); err != nil {
				return sdk.WrapError(err, "can't sendVCSEventStatus vcs %v/%v", proj.Key, vcsServerName)
			}
		}
	} else {
		skipStatus := false
		switch statusFound.State {
		case sdk.StatusSuccess:
			switch nodeRun.Status {
			case sdk.StatusSuccess:
				log.Info(ctx, "status %q %s found, skipping", expected, statusFound.State)
				skipStatus = true
			}
		case sdk.StatusFail:
			switch nodeRun.Status {
			case sdk.StatusFail:
				log.Info(ctx, "status %q %s found, skipping", expected, statusFound.State)
				skipStatus = true
			}

		case sdk.StatusSkipped:
			switch nodeRun.Status {
			case sdk.StatusDisabled, sdk.StatusNeverBuilt, sdk.StatusSkipped:
				log.Info(ctx, "status %q %s found, skipping", expected, statusFound.State)
				skipStatus = true
			}
		}

		if !skipStatus {
			for i := range notifs {
				log.Info(ctx, "status %q %s not found, sending a new one %+v", expected, nodeRun.Status, notifs[i])
				if err := e.sendVCSEventStatus(ctx, db, store, proj.Key, wr, &nodeRun, notifs[i], vcsServerName, cdsUIURL); err != nil {
					return sdk.WrapError(err, "can't sendVCSEventStatus vcs %v/%v", proj.Key, vcsServerName)
				}
			}
		}
	}

	if !sdk.StatusIsTerminated(nodeRun.Status) {
		return nil
	}
	for i := range notifs {
		if err := e.sendVCSPullRequestComment(ctx, db, wr, &nodeRun, notifs[i], vcsServerName); err != nil {
			return sdk.WrapError(err, "can't sendVCSPullRequestComment vcs %v/%v", proj.Key, vcsServerName)
		}
	}

	return nil
}

// sendVCSEventStatus send status
func (e *VCSEventMessenger) sendVCSEventStatus(ctx context.Context, db gorp.SqlExecutor, store cache.Store, projectKey string, wr sdk.WorkflowRun, nodeRun *sdk.WorkflowNodeRun, notif sdk.WorkflowNotification, vcsServerName string, cdsUIURL string) error {
	if notif.Settings.Template == nil || (notif.Settings.Template.DisableStatus != nil && *notif.Settings.Template.DisableStatus) {
		return nil
	}

	log.Info(ctx, "Send status %q for node run %d", nodeRun.Status, nodeRun.ID)
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
		return err
	}

	// Check if it's a gerrit or not
	isGerrit, err := e.vcsClient.IsGerrit(ctx, db)
	if err != nil {
		return err
	}
	if isGerrit {
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
		if changeID != "" {
			eventWNR.GerritChange = &sdk.GerritChangeEvent{
				ID:         changeID,
				DestBranch: branch,
				Project:    project,
				Revision:   revision,
				Report:     report,
				URL:        url,
			}
		}
	}

	payload, _ := json.Marshal(eventWNR)

	evt := sdk.Event{
		EventType:       fmt.Sprintf("%T", eventWNR),
		Payload:         payload,
		Timestamp:       time.Now(),
		ProjectKey:      projectKey,
		WorkflowName:    wr.Workflow.Name,
		PipelineName:    pipName,
		ApplicationName: appName,
		EnvironmentName: envName,
	}

	buildStatus := sdk.VCSBuildStatus{
		Description:        eventWNR.NodeName + ": " + eventWNR.Status,
		URLCDS:             fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", cdsUIURL, evt.ProjectKey, evt.WorkflowName, eventWNR.Number),
		Context:            fmt.Sprintf("%s-%s-%s", evt.ProjectKey, evt.WorkflowName, eventWNR.NodeName),
		Status:             eventWNR.Status,
		RepositoryFullname: eventWNR.RepositoryFullName,
		GitHash:            eventWNR.Hash,
		GerritChange:       eventWNR.GerritChange,
	}

	if err := e.vcsClient.SetStatus(ctx, buildStatus); err != nil {
		if err2 := repositoriesmanager.RetryEvent(&evt, err, store); err2 != nil {
			return err2
		}
		return err
	}

	return nil
}

func (e *VCSEventMessenger) sendVCSPullRequestComment(ctx context.Context, db gorp.SqlExecutor, wr sdk.WorkflowRun, nodeRun *sdk.WorkflowNodeRun, notif sdk.WorkflowNotification, vcsServerName string) error {
	log.Info(ctx, "Send pull-request comment for node run %d", nodeRun.ID)
	if notif.Settings.Template == nil {
		log.Info(ctx, "nothing to do: template is empty", nodeRun.ID)
		return nil
	}
	if notif.Settings.Template.DisableComment != nil && *notif.Settings.Template.DisableComment {
		log.Info(ctx, "nothing to do: comment are disabled")
		return nil
	}

	if nodeRun.Status != sdk.StatusFail && nodeRun.Status != sdk.StatusStopped && notif.Settings.OnSuccess != sdk.UserNotificationAlways {
		log.Info(ctx, "nothing to do: status is %v", nodeRun.Status)
		return nil
	}

	var app sdk.Application
	node := wr.Workflow.WorkflowData.NodeByID(nodeRun.WorkflowNodeID)
	if !node.IsLinkedToRepo(&wr.Workflow) {
		log.Info(ctx, "nothing to do: node is not linked to repo")
		return nil
	}

	if nodeRun.VCSReport == "" {
		nodeRun.VCSReport = notif.Settings.Template.Body
	}

	app = wr.Workflow.Applications[node.Context.ApplicationID]

	report, err := nodeRun.Report()
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return err
	}

	var changeID string
	changeIDParam := sdk.ParameterFind(nodeRun.BuildParameters, "gerrit.change.id")
	if changeIDParam != nil {
		changeID = changeIDParam.Value
	}

	var revision string
	revisionParams := sdk.ParameterFind(nodeRun.BuildParameters, "git.hash")
	if revisionParams != nil {
		revision = revisionParams.Value
	}

	reqComment := sdk.VCSPullRequestCommentRequest{Message: report}
	reqComment.Revision = revision

	isGerrit, err := e.vcsClient.IsGerrit(ctx, db)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return err
	}

	if changeID != "" && isGerrit {
		reqComment.ChangeID = changeID
		if err := e.vcsClient.PullRequestComment(ctx, app.RepositoryFullname, reqComment); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return err
		}
	} else if !isGerrit {
		//Check if this branch and this commit is a pullrequest
		prs, err := e.vcsClient.PullRequests(ctx, app.RepositoryFullname)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return err
		}

		//Send comment on pull request
		for _, pr := range prs {
			if pr.Head.Branch.DisplayID == nodeRun.VCSBranch && sdk.VCSIsSameCommit(pr.Head.Branch.LatestCommit, nodeRun.VCSHash) && !pr.Merged && !pr.Closed {
				reqComment.ID = pr.ID
				log.Info(ctx, "send comment (revision: %v pr: %v) on repo %s", reqComment.Revision, reqComment.ID, app.RepositoryFullname)
				if err := e.vcsClient.PullRequestComment(ctx, app.RepositoryFullname, reqComment); err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return err
				}
				break
			} else {
				log.Info(ctx, "nothing to do on pr %+v for branch %s", pr, nodeRun.VCSBranch)
			}
		}
	}
	return nil
}
