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
				ArgsUsage: "<kafka> <topic> <group> <user> <password>",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "pgp-decrypt",
						Usage: " <file> Decrypt incomming with a private ARMOR GPG Key",
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
			cli.Command{
				Name:      "ack",
				Usage:     "Send Ack to CDS",
				ArgsUsage: "<kafka> <topic> <user> <password> <cds-action json file> <OK|KO>",
				Flags: []cli.Flag{
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
