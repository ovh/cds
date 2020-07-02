package ascode

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// EventType type for as code events.
type EventType string

// AsCodeEventType values.
const (
	PipelineEvent    EventType = "pipeline"
	WorkflowEvent    EventType = "workflow"
	ApplicationEvent EventType = "application"
	EnvironmentEvent EventType = "environment"
)

type EntityData struct {
	Type          EventType
	ID            int64
	Name          string
	FromRepo      string
	OperationUUID string
}

// UpdateAsCodeResult pulls repositories operation and the create pullrequest + update workflow
func UpdateAsCodeResult(ctx context.Context, db *gorp.DbMap, store cache.Store, proj sdk.Project, workflowHolder sdk.Workflow, rootApp sdk.Application, ed EntityData, u sdk.Identifiable) {
	var asCodeEvent *sdk.AsCodeEvent
	globalOperation := sdk.Operation{
		UUID: ed.OperationUUID,
	}
	var globalErr error

	ope, err := operation.Poll(ctx, db, ed.OperationUUID)
	if err != nil {
		globalErr = err
	} else {
		asCodeEvent, err = createPullRequest(ctx, db, store, proj, workflowHolder.ID, rootApp, ed, u, ope.Setup)
		if err != nil {
			globalErr = err
		}
		sdk.GoRoutine(context.Background(), fmt.Sprintf("UpdateAsCodeResult-pusblish-as-code-event-%s", asCodeEvent.ID), func(ctx context.Context) {
			event.PublishAsCodeEvent(ctx, proj.Key, workflowHolder.Name, *asCodeEvent, u)
		})
	}

	if globalErr != nil {
		isErrWithStack := sdk.IsErrorWithStack(globalErr)
		fields := logrus.Fields{}
		if isErrWithStack {
			fields["stack_trace"] = fmt.Sprintf("%+v", globalErr)
		}
		log.ErrorWithFields(ctx, fields, "%s", globalErr)

		globalOperation.Status = sdk.OperationStatusError
		globalOperation.Error = sdk.ToOperationError(globalErr)
	} else {
		globalOperation.Status = sdk.OperationStatusDone
		globalOperation.Setup.Push.PRLink = asCodeEvent.PullRequestURL
	}

	sdk.GoRoutine(context.Background(), fmt.Sprintf("UpdateAsCodeResult-pusblish-operation-%s", globalOperation.UUID), func(ctx context.Context) {
		event.PublishOperation(ctx, proj.Key, globalOperation, u)
	})
}

func createPullRequest(ctx context.Context, db *gorp.DbMap, store cache.Store, proj sdk.Project, workflowHolderID int64, rootApp sdk.Application, ed EntityData, u sdk.Identifiable, opeSetup sdk.OperationSetup) (*sdk.AsCodeEvent, error) {
	vcsServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, db, proj.Key, rootApp.VCSServer)
	if err != nil {
		return nil, err
	}
	client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
	if err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to create repositories manager client")
	}

	request := sdk.VCSPullRequest{
		Title: opeSetup.Push.Message,
		Head: sdk.VCSPushEvent{
			Branch: sdk.VCSBranch{
				DisplayID: opeSetup.Push.FromBranch,
			},
			Repo: rootApp.RepositoryFullname,
		},
		Base: sdk.VCSPushEvent{
			Branch: sdk.VCSBranch{
				DisplayID: opeSetup.Push.ToBranch,
			},
			Repo: rootApp.RepositoryFullname,
		},
	}

	// Try to reuse a PR for the branche if exists else create a new one
	var pr *sdk.VCSPullRequest
	prs, err := client.PullRequests(ctx, rootApp.RepositoryFullname, sdk.VCSRequestModifierWithState(sdk.VCSPullRequestStateOpen))
	if err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to list pull request")
	}
	for _, prItem := range prs {
		if prItem.Base.Branch.DisplayID == opeSetup.Push.ToBranch && prItem.Head.Branch.DisplayID == opeSetup.Push.FromBranch {
			pr = &prItem
			break
		}
	}
	if pr == nil {
		newPR, err := client.PullRequestCreate(ctx, rootApp.RepositoryFullname, request)
		if err != nil {
			return nil, sdk.NewErrorFrom(err, "unable to create pull request")
		}
		pr = &newPR
	}

	// Find existing ascode event with this pull request info
	asCodeEvent, err := LoadEventByWorkflowIDAndPullRequest(ctx, db, workflowHolderID, rootApp.RepositoryFullname, int64(pr.ID))
	if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil, sdk.NewErrorFrom(err, "unable to save pull request")
	}
	if asCodeEvent.ID == 0 {
		asCodeEvent = &sdk.AsCodeEvent{
			WorkflowID:     workflowHolderID,
			FromRepo:       ed.FromRepo,
			PullRequestID:  int64(pr.ID),
			PullRequestURL: pr.URL,
			Username:       u.GetUsername(),
			CreateDate:     time.Now(),
			Migrate:        !opeSetup.Push.Update,
		}
	}

	switch ed.Type {
	case WorkflowEvent:
		if asCodeEvent.Data.Workflows == nil {
			asCodeEvent.Data.Workflows = make(map[int64]string)
		}
		found := false
		for k := range asCodeEvent.Data.Workflows {
			if k == ed.ID {
				found = true
				break
			}
		}
		if !found {
			asCodeEvent.Data.Workflows[ed.ID] = ed.Name
		}
	case PipelineEvent:
		if asCodeEvent.Data.Pipelines == nil {
			asCodeEvent.Data.Pipelines = make(map[int64]string)
		}
		found := false
		for k := range asCodeEvent.Data.Pipelines {
			if k == ed.ID {
				found = true
				break
			}
		}
		if !found {
			asCodeEvent.Data.Pipelines[ed.ID] = ed.Name
		}
	case ApplicationEvent:
		if asCodeEvent.Data.Applications == nil {
			asCodeEvent.Data.Applications = make(map[int64]string)
		}
		found := false
		for k := range asCodeEvent.Data.Applications {
			if k == ed.ID {
				found = true
				break
			}
		}
		if !found {
			asCodeEvent.Data.Applications[ed.ID] = ed.Name
		}
	case EnvironmentEvent:
		if asCodeEvent.Data.Environments == nil {
			asCodeEvent.Data.Environments = make(map[int64]string)
		}
		found := false
		for k := range asCodeEvent.Data.Environments {
			if k == ed.ID {
				found = true
				break
			}
		}
		if !found {
			asCodeEvent.Data.Environments[ed.ID] = ed.Name
		}
	}

	if err := UpsertEvent(db, asCodeEvent); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to insert as code event")
	}

	return asCodeEvent, nil
}
