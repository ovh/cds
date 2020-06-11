package main

import (
	"os"
	"path/filepath"

	"gopkg.in/urfave/cli.v1"
)

var (
	version = "0.4"
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
			{
				Name:      "help",
				Aliases:   []string{"h"},
				Usage:     "Shows a list of commands or help for one command",
				ArgsUsage: "[command]",
				Action:    helpAction,
			},
			{
				Name:      "listen",
				Aliases:   []string{"l"},
				Usage:     "Listen a Kafka topic and wait for chunks",
				ArgsUsage: "<kafka> <topic> <group> <user>",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "kafka-password",
						Usage:  " <password> Specify kafka password",
						EnvVar: "CDS_KAFKA_PASSWORD",
					},
					cli.StringFlag{
						Name:   "key",
						Usage:  " <key> Specify key used for aes encryption",
						EnvVar: "CDS_PLUGIN_KEY",
					},
					cli.StringFlag{
						Name:  "pgp-decrypt",
						Usage: " <file> Decrypt incoming with a private ARMOR GPG Key",
						Value: "",
					},
					cli.StringFlag{
						Name:  "exec",
						Usage: " <script> Exec script on complete receive",
						Value: "",
					},
				},
				Action: listenAction,
			},
			{
				Name:      "ack",
				Usage:     "Send Ack to CDS",
				ArgsUsage: "<kafka> <topic> <user> <cds-action json file> <OK|KO>",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "kafka-password",
						Usage:  " <password> Specify kafka password",
						EnvVar: "CDS_KAFKA_PASSWORD",
					},
					cli.StringFlag{
						Name:   "key",
						Usage:  " <key> Specify key used for aes encryption",
						EnvVar: "CDS_PLUGIN_KEY",
					},
					cli.StringFlag{
						Name:  "log",
						Usage: "--log <file>: Attach a log file",
					},
					cli.StringSliceFlag{
						Name:  "artifact",
						Usage: "--artifact <file> [--artifact <file>]: Upload artifact files ",
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
