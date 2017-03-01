package main

import (
	"os"
	"path/filepath"

	"gopkg.in/urfave/cli.v1"
)

func initCli(mainFunc func()) *cli.App {
	return &cli.App{
		Name:         filepath.Base(os.Args[0]),
		HelpName:     filepath.Base(os.Args[0]),
		Usage:        "CDS Kafka Publish Plugin",
		UsageText:    "Publish & Receive your CDS Builds and Artifacts through Kafka",
		Author:       "François SAMIN <francois.samin@©orp.ovh.com>",
		Version:      version,
		BashComplete: cli.DefaultAppComplete,
		Writer:       os.Stdout,
		Commands: []cli.Command{
			cli.Command{
				Name:      "help",
				Aliases:   []string{"h"},
				Usage:     "Shows a list of commands or help for one command",
				ArgsUsage: "[command]",
				Action:    helpAction,
			},
			cli.Command{
				Name:    "info",
				Aliases: []string{"i"},
				Usage:   "Show CDS Plugin information",
				Action:  infoAction,
			},
			cli.Command{
				Name:      "listen",
				Aliases:   []string{"l"},
				Usage:     "Listen a Kafka topic and wait for chunks",
				ArgsUsage: "<kafka> <topic> <group> <key>",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "pgp-decrypt",
						Usage: " <file> Decrypt incomming with a private ARMOR GPG Key",
						Value: "",
					},
				},
				Action: listenAction,
			},
			cli.Command{
				Name:      "send",
				Aliases:   []string{"s"},
				Usage:     "[DEBUG] Encrypt and send a file through Kafka topic",
				ArgsUsage: "<kafka> <topic> <key> <file>",
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "as-file",
						Usage: "Send file as plain file",
					},
					cli.BoolFlag{
						Name:  "as-chunks",
						Usage: "Send file as chunked file",
					},
					cli.StringFlag{
						Name:  "pgp-encrypt",
						Usage: "Encrypt file with a public pgp file",
						Value: "",
					},
					cli.Int64Flag{
						Name:  "actionID",
						Usage: "CDS Action ID Context",
					},
				},
				Action: sendAction,
			},
			cli.Command{
				Name:      "send-context",
				Usage:     "[DEBUG] Send a context through Kafka topic",
				ArgsUsage: "<kafka> <topic> <key> <actionID> <file> <file> ...",
				Action:    sendContext,
			},
			cli.Command{
				Name:      "ack",
				Usage:     "Send Ack to CDS",
				ArgsUsage: "<kafka> <topic> <key> <cds-action json file> <OK|KO>",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "log",
						Usage: "<file> Attach log file",
					},
				},
				Action: ackAction,
			},
		},
		Action: func(c *cli.Context) {
			mainFunc()
		},
	}
}
