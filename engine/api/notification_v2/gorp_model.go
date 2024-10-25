package notification_v2

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type dbProjectNotification struct {
	sdk.ProjectNotification
	gorpmapper.SignedEntity
}

func (n dbProjectNotification) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{n.ID, n.ProjectKey, n.WebHookURL, n.Filters}
	return gorpmapper.CanonicalForms{
		"{{.ID}}{{.ProjectKey}}{{md5sum .WebHookURL}}{{md5sum .Filters}}",
		"{{.ID}}{{.ProjectKey}}{{hash .WebHookURL}}{{hash .Filters}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbProjectNotification{}, "project_notification", false, "id"))
}
