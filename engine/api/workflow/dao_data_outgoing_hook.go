package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func insertNodeOutGoingHookData(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node) error {
	if n.OutGoingHookContext == nil {
		return nil
	}
	n.OutGoingHookContext.ID = 0

	n.OutGoingHookContext.NodeID = n.ID

	icon := w.HookModels[n.OutGoingHookContext.HookModelID].Icon

	n.OutGoingHookContext.Config["hookIcon"] = sdk.WorkflowNodeHookConfigValue{
		Value:        icon,
		Configurable: false,
	}
	n.OutGoingHookContext.Config[sdk.HookConfigProject] = sdk.WorkflowNodeHookConfigValue{Value: w.ProjectKey}
	n.OutGoingHookContext.Config[sdk.HookConfigWorkflow] = sdk.WorkflowNodeHookConfigValue{Value: w.Name}

	dbhook := dbNodeOutGoingHookData(*n.OutGoingHookContext)
	if err := db.Insert(&dbhook); err != nil {
		return sdk.WrapError(err, "insertNodeOutGoingHookData> Unable to insert outgoing hook")
	}
	n.OutGoingHookContext.ID = dbhook.ID

	return nil
}
