package workflow

import (
	"context"
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// ImportOptions is option to parse a workflow
type ImportOptions struct {
	Force              bool
	WorkflowName       string
	FromRepository     string
	IsDefaultBranch    bool
	FromBranch         string
	VCSServer          string
	RepositoryName     string
	RepositoryStrategy sdk.RepositoryStrategy
	HookUUID           string
}

// Parse parse an exportentities.workflow and return the parsed workflow
func Parse(proj *sdk.Project, ew *exportentities.Workflow, u *sdk.User) (*sdk.Workflow, error) {
	log.Info("Parse>> Parse workflow %s in project %s", ew.Name, proj.Key)
	log.Debug("Parse>> Workflow: %+v", ew)

	//Check valid application name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(ew.Name) {
		return nil, sdk.WrapError(sdk.ErrInvalidApplicationPattern, "Workflow name %s do not respect pattern %s", ew.Name, sdk.NamePattern)
	}

	//Inherit permissions from project
	if len(ew.Permissions) == 0 {
		ew.Permissions = make(map[string]int)
		for _, p := range proj.ProjectGroups {
			ew.Permissions[p.Group.Name] = p.Permission
		}
	}

	//Parse workflow
	w, errW := ew.GetWorkflow()
	if errW != nil {
		return nil, sdk.NewError(sdk.ErrWrongRequest, errW)
	}
	w.ProjectID = proj.ID
	w.ProjectKey = proj.Key

	return w, nil
}

// ParseAndImport parse an exportentities.workflow and insert or update the workflow in database
func ParseAndImport(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, oldW *sdk.Workflow, ew *exportentities.Workflow, u *sdk.User, opts ImportOptions) (*sdk.Workflow, []sdk.Message, error) {
	ctx, end := observability.Span(ctx, "workflow.ParseAndImport")
	defer end()

	log.Info("ParseAndImport>> Import workflow %s in project %s (force=%v)", ew.Name, proj.Key, opts.Force)
	log.Debug("ParseAndImport>> Workflow: %+v", ew)

	//Parse workflow
	w, errW := Parse(proj, ew, u)
	if errW != nil {
		return nil, nil, errW
	}

	// Load deep pipelines if we come from workflow run ( so we have hook uuid ).
	// We need deep pipelines to be able to run stages/jobs
	if err := IsValid(ctx, store, db, w, proj, u, LoadOptions{DeepPipeline: opts.HookUUID != ""}); err != nil {
		// Get spawn infos from error
		msg, ok := sdk.ErrorToMessage(err)
		if ok {
			return nil, []sdk.Message{msg}, sdk.WrapError(err, "Workflow is not valid")
		}
		return nil, nil, sdk.WrapError(err, "Workflow is not valid")
	}

	if err := RenameNode(db, w); err != nil {
		return nil, nil, sdk.WrapError(err, "Unable to rename node")
	}

	w.FromRepository = opts.FromRepository
	if !opts.IsDefaultBranch {
		w.DerivationBranch = opts.FromBranch
	}

	// do not override application data if no opts were given
	appID := w.WorkflowData.Node.Context.ApplicationID
	if opts.VCSServer != "" && appID != 0 {
		app := w.GetApplication(appID)
		app.VCSServer = opts.VCSServer
		app.RepositoryFullname = opts.RepositoryName
		app.RepositoryStrategy = opts.RepositoryStrategy
		w.Applications[appID] = app
	}

	if w.FromRepository != "" {
		if len(w.WorkflowData.Node.Hooks) == 0 {
			// When you came from run workflow you have uuid
			if opts.HookUUID != "" && oldW != nil {
				oldHooks := oldW.WorkflowData.GetHooks()
				if h, has := oldHooks[opts.HookUUID]; has {
					w.WorkflowData.Node.Hooks = append(w.WorkflowData.Node.Hooks, sdk.NodeHook{
						Ref:           h.Ref,
						UUID:          h.UUID,
						Config:        h.Config,
						HookModelName: h.HookModelName,
						HookModelID:   h.HookModelID,
					})
				}
			} else {
				// If we are coming from a workflow init command, the opts.HookUUID is empty, and we have to take the old value
				if opts.HookUUID == "" && oldW != nil &&
					len(oldW.WorkflowData.Node.Hooks) == 1 &&
					oldW.WorkflowData.Node.Hooks[0].HookModelName == sdk.RepositoryWebHookModel.Name {
					opts.HookUUID = oldW.WorkflowData.Node.Hooks[0].UUID
				}

				w.WorkflowData.Node.Hooks = append(w.WorkflowData.Node.Hooks, sdk.NodeHook{
					HookModelName: sdk.RepositoryWebHookModel.Name,
					HookModelID:   sdk.RepositoryWebHookModel.ID,
					Config:        sdk.RepositoryWebHookModel.DefaultConfig,
					UUID:          opts.HookUUID,
				})
			}

			var err error
			if w.WorkflowData.Node.Context.DefaultPayload, err = DefaultPayload(ctx, db, store, proj, w); err != nil {
				return nil, nil, sdk.WrapError(err, "Unable to get default payload")
			}
		}
	}

	if opts.WorkflowName != "" && w.Name != opts.WorkflowName {
		return nil, nil, sdk.WrapError(sdk.ErrWorkflowNameImport, "Wrong workflow name")
	}

	//Import
	done := new(sync.WaitGroup)
	done.Add(1)
	msgChan := make(chan sdk.Message)
	msgList := []sdk.Message{}
	go func(array *[]sdk.Message) {
		defer done.Done()
		for {
			m, more := <-msgChan
			if !more {
				return
			}
			*array = append(*array, m)
		}
	}(&msgList)

	globalError := Import(ctx, db, store, proj, oldW, w, u, opts.Force, msgChan)
	close(msgChan)
	done.Wait()

	return w, msgList, globalError
}
