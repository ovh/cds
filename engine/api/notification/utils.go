package notification

import (
	"bytes"

	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// SendMailNotif Send user notification by mail
func SendMailNotif(notif *sdk.Notif) {
	log.Debug("notification.SendMailNotif> Send notif '%s'", notif.Title)
	for _, recipient := range notif.Recipients {
		if err := mail.SendEmail(notif.Title, bytes.NewBufferString(notif.Message), recipient); err != nil {
			log.Critical("SendMailNotif> %s", err)
		}
	}
}
