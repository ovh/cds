package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pelletier/go-toml"
	"github.com/spf13/cobra"
	_ "github.com/spf13/viper/remote"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type Configuration struct {
	Log struct {
		Level string
	}
	API      api.Configuration
	Hatchery []struct{}
}

type ServiceServeOptions struct {
	SetHeaderFunc func() map[string]string
	Middlewares   []api.Middleware
}

type Service interface {
	Init(cfg interface{}) error
	Serve(ctx context.Context, opts *ServiceServeOptions) error
}

var (
	cfgFile      string
	remoteCfg    string
	remoteCfgKey string
	vaultAddr    string
	vaultToken   string
	vaultConfKey = "/secret/cds/conf"
	conf         = &Configuration{}
)

func init() {
	startCmd.Flags().StringVar(&cfgFile, "config", "", "config file")
	startCmd.Flags().StringVar(&remoteCfg, "remote-config", "", "(optional) consul configuration store")
	startCmd.Flags().StringVar(&remoteCfgKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")
	startCmd.Flags().StringVar(&vaultAddr, "vault-addr", "", "(optional) Vault address to fetch secrets from vault (example: https://vault.mydomain.net:8200)")
	startCmd.Flags().StringVar(&vaultToken, "vault-token", "", "(optional) Vault token to fetch secrets from vault")
	//Database command
	mainCmd.AddCommand(database.DBCmd)
	//Start command
	mainCmd.AddCommand(startCmd)
	//config command
	mainCmd.AddCommand(configCmd)
}

func main() {
	mainCmd.Execute()
}

var mainCmd = &cobra.Command{
	Use:   "engine",
	Short: "CDS Engine",
	Long: `
CDS
Continuous Delivery Service
Enterprise-Grade Continuous Delivery & DevOps Automation Open Source Platform

Copyright (c) 2013-2017, OVH SAS.
All rights reserved.`,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CDS Configuration",
	Long: `
CDS configuration file assistant.
Comming soon...`,
	Run: func(cmd *cobra.Command, args []string) {
		btes, err := toml.Marshal(*conf)
		if err != nil {
			sdk.Exit("%v", err)
		}
		fmt.Println(string(btes))
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start CDS",
	Long: `
Start CDS Engine Services:
 * API: 
 	This is the core component of CDS.
 * Hatcheries:
	They are the components responsible for spawning workers. Supported platforms/orchestrators are:
	 * Local machine
	 * Local Docker
	 * Openstack
	 * Docker Swarm
	 * Openstack

Start all of this with a single command: 
	$ engine start [api] [hatchery:local] [hatchery:docker] [hatchery:marathon] [hatchery:openstack] [hatchery:swarm] -f config.toml

All the services are using the same configuration file format. See $ engine config command for more details.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}

		//Initialize config
		config()

		//Initialize logs
		log.Initialize(&log.Conf{Level: conf.Log.Level})

		//Initialize context
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)

		// Gracefully shutdown all
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
		defer func() {
			signal.Stop(c)
			cancel()
		}()

		for _, a := range args {
			switch a {
			case "api":
				go startAPI(ctx)
			}
		}

		//Wait for the end
		select {
		case <-c:
			cancel()
			os.Exit(0)
		case <-ctx.Done():
		}
	},
}

func startAPI(c context.Context) {
	newAPI := api.New()
	if err := newAPI.Init(conf.API); err != nil {
		sdk.Exit("Unable to init API: %v", err)
	}
	if err := newAPI.Serve(c); err != nil {
		sdk.Exit("API has been stopped: %v", err)
	}
}
