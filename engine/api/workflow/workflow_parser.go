package workflow

import (
	"context"
	"sync"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
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
func Parse(ctx context.Context, proj sdk.Project, ew exportentities.Workflow) (*sdk.Workflow, error) {
	log.Info(ctx, "Parse>> Parse workflow %s in project %s", ew.GetName(), proj.Key)
	log.Debug("Parse>> Workflow: %+v", ew)

	//Parse workflow
	w, errW := exportentities.ParseWorkflow(ew)
	if errW != nil {
		return nil, sdk.NewError(sdk.ErrWrongRequest, errW)
	}
	w.ProjectID = proj.ID
	w.ProjectKey = proj.Key

	// Get permission from project if needed
	if len(w.Groups) == 0 {
		w.Groups = make([]sdk.GroupPermission, 0, len(proj.ProjectGroups))
		for _, gp := range proj.ProjectGroups {
			perm := sdk.GroupPermission{Group: sdk.Group{Name: gp.Group.Name}, Permission: gp.Permission}
			w.Groups = append(w.Groups, perm)
		}
	}
	return w, nil
}

// ParseAndImport parse an exportentities.workflow and insert or update the workflow in database
func ParseAndImport(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, oldW *sdk.Workflow, ew exportentities.Workflow, u sdk.Identifiable, opts ImportOptions) (*sdk.Workflow, []sdk.Message, error) {
	ctx, end := telemetry.Span(ctx, "workflow.ParseAndImport")
	defer end()

	log.Info(ctx, "ParseAndImport>> Import workflow %s in project %s (force=%v)", ew.GetName(), proj.Key, opts.Force)

	//Parse workflow
	w, err := Parse(ctx, proj, ew)
	if err != nil {
		return nil, nil, err
	}

	// Load deep pipelines if we come from workflow run ( so we have hook uuid ).
	// We need deep pipelines to be able to run stages/jobs
	if err := CompleteWorkflow(ctx, db, w, proj, LoadOptions{DeepPipeline: opts.HookUUID != ""}); err != nil {
		// Get spawn infos from error
		msg, ok := sdk.ErrorToMessage(err)
		if ok {
			return nil, []sdk.Message{msg}, sdk.WrapError(err, "workflow is not valid")
		}
		return nil, nil, sdk.WrapError(err, "workflow is not valid")
	}
	if err := RenameNode(ctx, db, w); err != nil {
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
		// Get repowebhook from previous version of workflow
		var oldRepoWebHook *sdk.NodeHook
		if oldW != nil {
			for i := range oldW.WorkflowData.Node.Hooks {
				h := &oldW.WorkflowData.Node.Hooks[i]
				if h.IsRepositoryWebHook() {
					oldRepoWebHook = h
					break
				}
			}

			if oldRepoWebHook != nil {
				// Update current repo web hook if found
				var currentRepoWebHook *sdk.NodeHook
				// Get current webhook
				for i := range w.WorkflowData.Node.Hooks {
					h := &w.WorkflowData.Node.Hooks[i]
					if h.IsRepositoryWebHook() {
						h.UUID = oldRepoWebHook.UUID
						h.Config.MergeWith(
							oldRepoWebHook.Config.Filter(
								func(k string, v sdk.WorkflowNodeHookConfigValue) bool {
									return !v.Configurable
								},
							),
						)
						// get only non cofigurable stuff
						currentRepoWebHook = h
						log.Debug("workflow.ParseAndImport> keeping the old repository web hook: %+v (%+v)", h, oldRepoWebHook)
						break
					}
				}

				// If not found, take the default config
				if currentRepoWebHook == nil {
					h := sdk.NodeHook{
						UUID:          oldRepoWebHook.UUID,
						HookModelName: oldRepoWebHook.HookModelName,
						Config:        sdk.RepositoryWebHookModel.DefaultConfig.Clone(),
						HookModelID:   sdk.RepositoryWebHookModel.ID,
					}
					oldNonConfigurableConfig := oldRepoWebHook.Config.Filter(func(k string, v sdk.WorkflowNodeHookConfigValue) bool {
						return !v.Configurable
					})
					for k, v := range oldNonConfigurableConfig {
						h.Config[k] = v
					}
					w.WorkflowData.Node.Hooks = append(w.WorkflowData.Node.Hooks, h)
				}
			}
		}

		// If there is no old workflow OR workflow existing on CDS does not have a repoWebhook,
		// we have to create a new repo webhook
		if oldW == nil || oldRepoWebHook == nil {
			// Init new repo webhook
			var newRepoWebHook = sdk.NodeHook{
				HookModelName: sdk.RepositoryWebHookModel.Name,
				HookModelID:   sdk.RepositoryWebHookModel.ID,
				Config:        sdk.RepositoryWebHookModel.DefaultConfig.Clone(),
			}

			// If the new workflow already contains a repowebhook, we dont have to add a new one
			var hasARepoWebHook bool
			for _, h := range w.WorkflowData.Node.Hooks {
				if h.Ref() == newRepoWebHook.Ref() {
					hasARepoWebHook = true
					break
				}
				if h.HookModelName == newRepoWebHook.HookModelName &&
					h.ConfigValueContainsEventsDefault() {
					hasARepoWebHook = true
					break
				}
			}
			if !hasARepoWebHook {
				w.WorkflowData.Node.Hooks = append(w.WorkflowData.Node.Hooks, newRepoWebHook)
			}

			var err error
			if w.WorkflowData.Node.Context.DefaultPayload, err = DefaultPayload(ctx, db, store, proj, w); err != nil {
				return nil, nil, err
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

	if ew.GetVersion() == exportentities.WorkflowVersion1 {
		msgList = append(msgList, sdk.NewMessage(sdk.MsgWorkflowDeprecatedVersion, proj.Key, ew.GetName()))
	}

	return w, msgList, globalError
}
