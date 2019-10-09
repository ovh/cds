package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// CountHooksByApplication count hooks by application id
func CountHooksByApplication(db gorp.SqlExecutor, appID int64) (int64, error) {
	query := `
    SELECT count(w_node_hook.*)
    FROM w_node_hook
    JOIN w_node_context ON w_node_context.node_id = w_node_hook.node_id
    WHERE w_node_context.application_id = $1;
  `
	count, err := db.SelectInt(query, appID)
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	return count, nil
}

// LoadHookByUUID load a hook by his uuid
func LoadHookByUUID(db gorp.SqlExecutor, uuid string) (sdk.NodeHook, error) {
	var hook sdk.NodeHook
	var res dbNodeHookData
	if err := db.SelectOne(&res, "select * from w_node_hook where uuid = $1", uuid); err != nil {
		if err == sql.ErrNoRows {
			return hook, sdk.WithStack(sdk.ErrNotFound)
		}
		return hook, sdk.WithStack(err)
	}

	model, err := LoadHookModelByID(db, res.HookModelID)
	if err != nil {
		return hook, sdk.WithStack(err)
	}
	res.HookModelName = model.Name
	return sdk.NodeHook(res), nil
}

// LoadAllHooks returns all hooks
func LoadAllHooks(db gorp.SqlExecutor) ([]sdk.NodeHook, error) {
	models, err := LoadHookModels(db)
	if err != nil {
		return nil, err
	}

	var res []dbNodeHookData
	if _, err := db.Select(&res, "select * from w_node_hook"); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}

	nodes := make([]sdk.NodeHook, 0, len(res))
	for i := range res {
		for _, m := range models {
			if res[i].HookModelID == m.ID {
				res[i].HookModelName = m.Name
				break
			}
		}
		nodes = append(nodes, sdk.NodeHook(res[i]))
	}

	return nodes, nil
}

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
