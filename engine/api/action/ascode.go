package action

import (
	"context"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func CreateAsCodeAction(ctx context.Context, db gorpmapper.SqlExecutorWithTx) error {
	nb, err := db.SelectInt(`
        SELECT count(id)
        FROM action
        WHERE action.type = $1`, sdk.AsCodeAction)
	if err != nil {
		return sdk.WrapError(err, "unable to get ascode action")
	}

	// If the action doesn't exist, let's create
	if nb == 0 {
		log.Debug(ctx, "CreateAsCodeAction> create ascode action")
		act := sdk.Action{
			Type:    sdk.AsCodeAction,
			Name:    "AsCodeAction",
			Enabled: true,
		}
		if err := Insert(db, &act); err != nil {
			return err
		}
		return nil
	}
	return nil
}
