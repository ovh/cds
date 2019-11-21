package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/fsamin/smtp"

	"github.com/ovh/cds/tools/smtpmock"
)

func main() {
	log.Fatal(smtpmock.StartServer(context.Background(), ":2023",
		smtpmock.Handle("*@*", func(envelope *smtp.Envelope) error {
			fmt.Println("Message Received", envelope.MessageTo)
			fmt.Println("From:", envelope.MessageFrom, envelope.RemoteAddr)
			fmt.Println("To:", envelope.MessageTo)
			io.Copy(os.Stdout, envelope.MessageData)
			return nil
		})))
}
