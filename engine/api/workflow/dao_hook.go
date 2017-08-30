package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
)

// insertHook inserts a hook
func insertHook(db gorp.SqlExecutor, node *sdk.WorkflowNode, hook *sdk.WorkflowNodeHook) error {
	hook.WorkflowNodeID = node.ID
	if hook.WorkflowHookModelID == 0 {
		hook.WorkflowHookModelID = hook.WorkflowHookModel.ID
	}

	if hook.WorkflowHookModelID != 0 {
		model, errm := LoadHookModelByID(db, hook.WorkflowHookModelID)
		if errm != nil {
			return sdk.WrapError(errm, "insertHook> Unable to load model %d", hook.WorkflowHookModelID)
		}
		hook.WorkflowHookModel = *model
	} else {
		model, errm := LoadHookModelByName(db, hook.WorkflowHookModel.Name)
		if errm != nil {
			return sdk.WrapError(errm, "insertHook> Unable to load model %s", hook.WorkflowHookModel.Name)
		}
		hook.WorkflowHookModel = *model
	}
	hook.WorkflowHookModelID = hook.WorkflowHookModel.ID

	//TODO Check configuration of the hook vs the model

	uuid, erruuid := sessionstore.NewSessionKey()
	if erruuid != nil {
		return sdk.WrapError(erruuid, "insertHook> Unable to load model %d", hook.WorkflowHookModelID)
	}

	hook.UUID = string(uuid)

	dbhook := NodeHook(*hook)
	if err := db.Insert(&dbhook); err != nil {
		return sdk.WrapError(err, "insertHook> Unable to insert hook")
	}
	*hook = sdk.WorkflowNodeHook(dbhook)
	return nil
}

//PostInsert is a db hook
func (r *NodeHook) PostInsert(db gorp.SqlExecutor) error {
	if r.Conditions == nil {
		r.Conditions = []sdk.WorkflowTriggerCondition{}
	}

	sConditions, err := gorpmapping.JSONToNullString(r.Conditions)
	if err != nil {
		return err
	}

	sConfig, err := gorpmapping.JSONToNullString(r.Config)
	if err != nil {
		return err
	}

	if _, err := db.Exec("update workflow_node_hook set conditions = $2, config = $3 where id = $1", r.ID, sConditions, sConfig); err != nil {
		return err
	}
	return err
}

//PostGet is a db hook
func (r *NodeHook) PostGet(db gorp.SqlExecutor) error {
	var res = struct {
		Conditions sql.NullString `db:"conditions"`
		Config     sql.NullString `db:"config"`
	}{}
	if err := db.SelectOne(&res, "select conditions, config from workflow_node_hook where id = $1", r.ID); err != nil {
		return err
	}

	conf := sdk.WorkflowNodeHookConfig{}
	conditions := []sdk.WorkflowTriggerCondition{}

	if err := gorpmapping.JSONNullString(res.Conditions, &conditions); err != nil {
		return err
	}

	if err := gorpmapping.JSONNullString(res.Config, &conf); err != nil {
		return err
	}

	r.Conditions = conditions
	r.Config = conf
	return nil
}

func loadHooks(db gorp.SqlExecutor, node *sdk.WorkflowNode) ([]sdk.WorkflowNodeHook, error) {
	res := []NodeHook{}
	if _, err := db.Select(&res, "select id, uuid, workflow_hook_model_id from workflow_node_hook where workflow_node_id = $1", node.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "loadHooks")
	}

	nodes := []sdk.WorkflowNodeHook{}
	for i := range res {
		if err := res[i].PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "loadHooks")
		}
		nodes = append(nodes, sdk.WorkflowNodeHook(res[i]))
	}
	return nodes, nil
}
