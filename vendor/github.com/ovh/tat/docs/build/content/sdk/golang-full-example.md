---
title: "Golang - Full Example"
weight: 2
toc: true
prev: "/sdk/golang-simple-main"

---

## Usage

This example uses [viper](https://github.com/spf13/viper), [cobra](https://github.com/spf13/cobra) and [tatcli](https://ovh.github.io/tat/tatcli/general/) config file.

```
Usage:
 go build && ./mycli-full demo /YouTopic/subTopic your message

with a config file:
 go build && ./mycli-full --configFile $HOME/.tatcli/config.local.json demo /YouTopic/subTopic your message

```

## File main.go

```go
package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:  "mycli-full",
	Long: `SDK Use Demo`,
}

// URL of tat engine, tat username and tat password
var (
	home     = os.Getenv("HOME")
	taturl   string
	username string
	password string

	// ConfigFile is $HOME/.tatcli/config.json per default
	// contains user, password and url of tat
	configFile string
)

func main() {
	addCommands()

	rootCmd.PersistentFlags().StringVarP(&taturl, "url", "", "", "URL Tat Engine, facultative if you have a "+home+"/.tatcli/config.json file")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "username, facultative if you have a "+home+"/.tatcli/config.json file")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "password, facultative if you have a "+home+"/.tatcli/config.json file")
	rootCmd.PersistentFlags().StringVarP(&configFile, "configFile", "c", home+"/.tatcli/config.json", "configuration file, default is "+home+"/.tatcli/config.json")

	viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	viper.BindPFlag("username", rootCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))

	log.SetLevel(log.DebugLevel)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

//AddCommands adds child commands to the root command rootCmd.
func addCommands() {
	rootCmd.AddCommand(cmdDemo)
}

var cmdDemo = &cobra.Command{
	Use:   "demo <topic> <msg>",
	Short: "Demo Post Msg",
	Run: func(cmd *cobra.Command, args []string) {
		create(args[0], args[1])
	},
}

// create creates a message in specified topic
func create(topic, message string) {
	readConfig()
	m := tat.MessageJSON{Text: message, Topic: topic}
	msgCreated, err := getClient().MessageAdd(m)
	if err != nil {
		log.Errorf("Error:%s", err)
		return
	}
	log.Debugf("ID Message Created: %d", msgCreated.Message.ID)
}

func getClient() *tat.Client {
	tc, err := tat.NewClient(tat.Options{
		URL:      viper.GetString("url"),
		Username: viper.GetString("username"),
		Password: viper.GetString("password"),
		Referer:  "mycli.v0",
	})

	if err != nil {
		log.Fatalf("Error while create new Tat Client: %s", err)
	}

	tat.DebugLogFunc = log.Debugf
	return tc
}

// readConfig reads config in .tatcli/config per default
func readConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
		viper.ReadInConfig() // Find and read the config file
	}
}

```

## Notice
You should split this file into many files.

See https://github.com/ovh/tat/tatcli for CLI with many subcommands.
