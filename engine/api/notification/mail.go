package notification

import (
	"bytes"
	"regexp"

	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var regexpIsHTML = regexp.MustCompile(`^\w*\n*<[a-z][\s\S]*>`)

// SendMailNotif Send user notification by mail
func SendMailNotif(notif sdk.EventNotif) {
	log.Info("notification.SendMailNotif> Send notif '%s'", notif.Subject)
	errors := []string{}
	for _, recipient := range notif.Recipients {
		isHTML := regexpIsHTML.MatchString(notif.Body)
		if err := mail.SendEmail(notif.Subject, bytes.NewBufferString(notif.Body), recipient, isHTML); err != nil {
			errors = append(errors, err.Error())
		}
	}
}
