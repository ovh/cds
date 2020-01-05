package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ovh/cds/tools/smtpmock/sdk"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var client sdk.Client

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "api-url",
			Value:   "http://localhost:2024",
			EnvVars: []string{"SMTPMOCK_API_URL"},
		},
		&cli.StringFlag{
			Name:    "signin-token",
			EnvVars: []string{"SMTPMOCK_SIGNIN_TOKEN"},
		},
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "messages",
				Usage:  "Get messages",
				Action: messages,
			},
			{
				Name:   "recipient-messages",
				Usage:  "Get messages for a recipient",
				Action: recipientMessages,
			},
			{
				Name:   "recipient-latest-message",
				Usage:  "Get latest message for a recipient",
				Action: recipientLatestMessage,
			},
		},
		Flags: flags,
		Before: func(ctx *cli.Context) error {
			client = sdk.NewClient(ctx.String("api-url"))

			token := ctx.String("signin-token")
			if token != "" {
				if _, err := client.Signin(token); err != nil {
					return err
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("%v\n", err)
	}
}

func messages(ctx *cli.Context) error {
	ms, err := client.GetMessages()
	if err != nil {
		return err
	}
	return printJSON(ms)
}

func recipientMessages(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return errors.New("missing recipient email")
	}
	ms, err := client.GetRecipientMessages(ctx.Args().First())
	if err != nil {
		return err
	}
	return printJSON(ms)
}

func recipientLatestMessage(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return errors.New("missing recipient email")
	}
	m, err := client.GetRecipientLatestMessage(ctx.Args().First())
	if err != nil {
		return err
	}
	return printJSON(m)
}

func printJSON(i interface{}) error {
	buf, err := json.Marshal(i)
	if err != nil {
		return errors.WithStack(err)
	}
	fmt.Println(string(buf))
	return nil
}
