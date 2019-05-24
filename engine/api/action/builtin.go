package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/action"
	"github.com/ovh/cds/sdk/log"
)

// CreateBuiltinActions add builtin actions in database if needed
func CreateBuiltinActions(db *gorp.DbMap) error {
	for i := range action.List {
		if err := checkBuiltinAction(db, &action.List[i].Action); err != nil {
			return err
		}
	}

	return nil
}

// checkBuiltinAction add builtin actions in database if needed
func checkBuiltinAction(db *gorp.DbMap, a *sdk.Action) error {
	a.Type = sdk.BuiltinAction

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()

	nb, err := tx.SelectInt("SELECT COUNT(1) FROM action WHERE action.name = $1 and action.type = $2", a.Name, sdk.BuiltinAction)
	if err != nil {
		return sdk.WrapError(err, "unable to count action %s", a.Name)
	}

	// If the action doesn't exist, let's create
	if nb == 0 {
		log.Debug("createBuiltinAction> create builtin action %s", a.Name)
		if err := Insert(tx, a); err != nil {
			return err
		}
		return sdk.WithStack(tx.Commit())
	}

	id, err := tx.SelectInt("SELECT id FROM action WHERE action.name = $1 and action.type = $2", a.Name, sdk.BuiltinAction)
	if err != nil {
		return sdk.WrapError(err, "unable to get action %s ID", a.Name)
	}

	a.ID = id
	log.Debug("createBuiltinAction> update builtin action %s", a.Name)
	if err := Update(tx, a); err != nil {
		return err
	}
	return sdk.WithStack(tx.Commit())
}
