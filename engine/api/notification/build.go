package notification

/* // TODO EVENT yesnault FILE TO DELETE

import (
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// SendActionBuild sends a build notification


func SendActionBuild(db database.QueryExecuter, ab *sdk.ActionBuild, event sdk.NotifEventType, status sdk.Status) {
	if !notifON {
		return
	}

	log.Debug("notification.SendActionBuild> pb:%d ab:%d event:%s status:%s notifType:%s",
		ab.PipelineBuildID, ab.ID, event, status, sdk.ActionBuildNotif)

	n := &sdk.Notif{
		DateNotif:   time.Now().Unix(),
		ActionBuild: ab,
		Event:       event,
		Status:      status,
		NotifType:   sdk.ActionBuildNotif,
	}

	go post(n)
}

*/
