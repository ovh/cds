package notification

import (
	"bytes"

	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SendMailNotif Send user notification by mail
func SendMailNotif(notif sdk.EventNotif) {
	log.Info("notification.SendMailNotif> Send notif '%s'", notif.Subject)
	errors := []string{}
	for _, recipient := range notif.Recipients {
		if err := mail.SendEmail(notif.Subject, bytes.NewBufferString(notif.Body), recipient); err != nil {
			errors = append(errors, err.Error())
		}
	}
}
