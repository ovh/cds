package notification

import (
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// SendBuiltinNotif sends a builtin notification
func SendBuiltinNotif(db database.QueryExecuter, ab *sdk.ActionBuild, notif sdk.Notif) {
	if !notifON {
		return
	}

	log.Debug("notification.SendBuiltinNotif> pb:%d ab:%d event:%s status:%s notifType:%s destination:%s title:%s message:%s",
		ab.PipelineBuildID, ab.ID, notif.Event, notif.Status, sdk.BuiltinNotif,
		notif.Destination, notif.Title, notif.Message)

	notif.NotifType = sdk.BuiltinNotif
	go post(&notif)
}
