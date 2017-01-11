package notification

import (
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

// SendActionBuild sends a build notification
func SendActionBuild(db database.QueryExecuter, ab *sdk.ActionBuild, event sdk.NotifEventType, status sdk.Status) {
	if !notifON {
		return
	}

	n := &sdk.Notif{
		DateNotif:   time.Now().Unix(),
		ActionBuild: ab,
		Event:       event,
		Status:      status,
		NotifType:   sdk.ActionBuildNotif,
	}

	go post(n)
}
