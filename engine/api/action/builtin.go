package action

import (
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/action"
	"github.com/ovh/cds/sdk/log"
)

// CreateBuiltinActions add builtin actions in database if needed
func CreateBuiltinActions(db gorpmapper.SqlExecutorWithTx) error {
	for i := range action.List {
		if err := checkBuiltinAction(db, &action.List[i].Action); err != nil {
			return err
		}
	}

	return nil
}

// checkBuiltinAction add builtin actions in database if needed
func checkBuiltinAction(db gorpmapper.SqlExecutorWithTx, a *sdk.Action) error {
	a.Type = sdk.BuiltinAction

	nb, err := db.SelectInt("SELECT COUNT(1) FROM action WHERE action.name = $1 and action.type = $2", a.Name, sdk.BuiltinAction)
	if err != nil {
		return sdk.WrapError(err, "unable to count action %s", a.Name)
	}

	// If the action doesn't exist, let's create
	if nb == 0 {
		log.Debug("createBuiltinAction> create builtin action %s", a.Name)
		if err := Insert(db, a); err != nil {
			return err
		}
		return nil
	}

	id, err := db.SelectInt("SELECT id FROM action WHERE action.name = $1 and action.type = $2", a.Name, sdk.BuiltinAction)
	if err != nil {
		return sdk.WrapError(err, "unable to get action %s ID", a.Name)
	}

	a.ID = id
	if err := Update(db, a); err != nil {
		return err
	}
	return nil
}
