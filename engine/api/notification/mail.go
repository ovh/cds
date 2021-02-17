package notification

import (
	"bytes"
	"context"
	"regexp"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/sdk"
)

var regexpIsHTML = regexp.MustCompile(`^\w*\n*<[a-z][\s\S]*>`)

// sendMailNotif Send user notification by mail
func sendMailNotif(ctx context.Context, notif sdk.EventNotif) {
	log.Info(ctx, "notification.sendMailNotif> Send notif '%s' nb.Recipients:%d", notif.Subject, len(notif.Recipients))
	for _, recipient := range notif.Recipients {
		isHTML := regexpIsHTML.MatchString(notif.Body)
		if err := mail.SendEmail(ctx, notif.Subject, bytes.NewBufferString(notif.Body), recipient, isHTML); err != nil {
			log.Error(ctx, "sendMailNotif>error while sending mail: %v to recipients:%+v and subject:%v", err, notif.Recipients, notif.Subject)
		}
	}
}
