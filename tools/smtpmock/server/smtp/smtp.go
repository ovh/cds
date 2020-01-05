package smtp

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/fsamin/smtp"

	"github.com/ovh/cds/tools/smtpmock/sdk"
	"github.com/ovh/cds/tools/smtpmock/server/store"
)

func Start(ctx context.Context, port int) error {
	srv := smtp.NewServeMux()
	srv.HandleFunc("*@*", smtpHandler)
	fmt.Printf("smtp server started on :%d\n", port)
	return smtp.ListenAndServeWithContext(ctx, fmt.Sprintf(":%d", port), srv)
}

func smtpHandler(envelope *smtp.Envelope) error {
	m := sdk.Message{
		RemoteAddress: envelope.RemoteAddr,
		FromAgent:     envelope.FromAgent,
		To:            envelope.MessageTo,
		From:          envelope.MessageFrom,
		User:          envelope.User,
	}

	btes, err := ioutil.ReadAll(envelope.MessageData)
	if err != nil {
		return err
	}

	m.Content = string(btes)

	store.AddMessage(m)

	return nil
}
