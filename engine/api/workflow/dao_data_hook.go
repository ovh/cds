package workflow

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func insertNodeHookData(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node) error {
	if n.Hooks == nil || len(n.Hooks) == 0 {
		return nil
	}

	hookToKeep := make([]sdk.NodeHook, 0)
	for i := range n.Hooks {
		h := &n.Hooks[i]
		h.NodeID = n.ID

		model := w.HookModels[h.HookModelID]
		if model.Name == sdk.RepositoryWebHookModelName && n.Context.ApplicationID == 0 {
			continue
		}

		//Configure the hook
		h.Config[sdk.HookConfigProject] = sdk.WorkflowNodeHookConfigValue{
			Value:        w.ProjectKey,
			Configurable: false,
		}

		h.Config[sdk.HookConfigWorkflow] = sdk.WorkflowNodeHookConfigValue{
			Value:        w.Name,
			Configurable: false,
		}

		h.Config[sdk.HookConfigWorkflowID] = sdk.WorkflowNodeHookConfigValue{
			Value:        fmt.Sprint(w.ID),
			Configurable: false,
		}

		if model.Name == sdk.RepositoryWebHookModelName || model.Name == sdk.GitPollerModelName {
			if n.Context.ApplicationID == 0 || w.Applications[n.Context.ApplicationID].RepositoryFullname == "" || w.Applications[n.Context.ApplicationID].VCSServer == "" {
				return sdk.WrapError(sdk.ErrForbidden, "insertNodeHookData> Cannot create a git poller or repository webhook on an application without a repository")
			}
			h.Config["vcsServer"] = sdk.WorkflowNodeHookConfigValue{
				Value:        w.Applications[n.Context.ApplicationID].VCSServer,
				Configurable: false,
			}
			h.Config["repoFullName"] = sdk.WorkflowNodeHookConfigValue{
				Value:        w.Applications[n.Context.ApplicationID].RepositoryFullname,
				Configurable: false,
			}
		}

		dbHook := dbNodeHookData(*h)
		if err := db.Insert(&dbHook); err != nil {
			return sdk.WrapError(err, "insertNodeHookData> Unable to insert workflow node hook")
		}
		h.ID = dbHook.ID

		hookToKeep = append(hookToKeep, *h)
	}

	n.Hooks = hookToKeep
	return nil
}

// PostInsert is a db hook
func (h *dbNodeHookData) PostInsert(db gorp.SqlExecutor) error {
	return h.PostUpdate(db)
}

// PostUpdate is a db hook
func (h *dbNodeHookData) PostUpdate(db gorp.SqlExecutor) error {
	config, errC := gorpmapping.JSONToNullString(h.Config)
	if errC != nil {
		return sdk.WrapError(errC, "dbNodeHookData.PostUpdate> Unable to marshall config")
	}

	if _, err := db.Exec("UPDATE w_node_hook SET config = $1 WHERE id = $2", config, h.ID); err != nil {
		return sdk.WrapError(err, "dbNodeHookData.PostUpdate> Unable to update config")
	}
	return nil
}
