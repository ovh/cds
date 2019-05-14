package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func insertNodeHookData(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node) error {
	if n.Hooks == nil || len(n.Hooks) == 0 {
		return nil
	}

	for i := range n.Hooks {
		h := &n.Hooks[i]
		h.NodeID = n.ID

		dbHook := dbNodeHookData(*h)
		if err := db.Insert(&dbHook); err != nil {
			return sdk.WrapError(err, "insertNodeHookData> Unable to insert workflow node hook")
		}
		h.ID = dbHook.ID
	}
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
