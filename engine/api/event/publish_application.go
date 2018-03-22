package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// PublishApplicationEvent publish application event
func PublishApplicationEvent(payload interface{}, key, appName string, u *sdk.User) {
	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         structs.Map(payload),
		ProjectKey:      key,
		ApplicationName: appName,
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
	}
	publishEvent(event)
}

// PublishAddApplication publishes an event for the creation of the given application
func PublishAddApplication(projKey string, app sdk.Application, u *sdk.User) {
	e := sdk.EventAddApplication{
		app,
	}
	PublishApplicationEvent(e, projKey, app.Name, u)
}

// PublishUpdateApplication publishes an event for the update of the given application
func PublishUpdateApplication(projKey string, app sdk.Application, oldApp sdk.Application, u *sdk.User) {
	e := sdk.EventUpdateApplication{
		NewMetadata:           app.Metadata,
		NewRepositoryStrategy: app.RepositoryStrategy,
		NewName:               app.Name,
		OldMetadata:           oldApp.Metadata,
		OldRepositoryStrategy: oldApp.RepositoryStrategy,
		OldName:               oldApp.Name,
	}
	PublishApplicationEvent(e, projKey, app.Name, u)
}
