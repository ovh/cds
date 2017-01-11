package notification

import (
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

// SendBuiltinNotif sends a builtin notification
func SendBuiltinNotif(db database.QueryExecuter, ab *sdk.ActionBuild, notif sdk.Notif) {
	if !notifON {
		return
	}

	notif.NotifType = sdk.BuiltinNotif
	go post(&notif)
}
