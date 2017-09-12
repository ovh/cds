package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	defaults "github.com/mcuadros/go-defaults"
	"github.com/spf13/cobra"
	_ "github.com/spf13/viper/remote"
	"github.com/yesnault/go-toml"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/hatchery/docker"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/marathon"
	"github.com/ovh/cds/engine/hatchery/openstack"
	"github.com/ovh/cds/engine/hatchery/vsphere"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

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
	//Version  command
	mainCmd.AddCommand(versionCmd)
	//Database command
	mainCmd.AddCommand(database.DBCmd)
	//Start command
	mainCmd.AddCommand(startCmd)
	//Config command
	mainCmd.AddCommand(configCmd)
	configNewCmd.Flags().BoolVar(&configNewAsEnvFlag, "env", false, "Print configuration as environment variable")

	configCmd.AddCommand(configNewCmd)
	configCmd.AddCommand(configCheckCmd)
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
https://ovh.github.io/cds/

Copyright (c) 2013-2017, OVH SAS.
All rights reserved.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display CDS version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(sdk.VERSION)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CDS Configuration",
}

var configNewAsEnvFlag bool

var configNewCmd = &cobra.Command{
	Use:   "new",
	Short: "CDS configuration file assistant",
	Long: `
Comming soon...`,
	Run: func(cmd *cobra.Command, args []string) {
		defaults.SetDefaults(conf)

		conf.API.Auth.SharedInfraToken = sdk.RandomString(128)
		conf.API.Secrets.Key = sdk.RandomString(32)
		conf.Hatchery.Local.API.Token = conf.API.Auth.SharedInfraToken

		if !configNewAsEnvFlag {
			btes, err := toml.Marshal(*conf)
			if err != nil {
				sdk.Exit("%v", err)
			}
			fmt.Println(string(btes))
		} else {
			m := AsEnvVariables(conf, "cds")
			keys := []string{}

			for k := range m {
				keys = append(keys, k)
			}

			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("export %s=\"%s\"\n", k, m[k])
			}
		}
	},
}

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check CDS configuration file",
	Long:  `$ engine config check <path>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			cmd.Help()
			sdk.Exit("Wrong usage")
		}

		cfgFile = args[0]
		//Initialize config
		config()

		var hasError bool
		if conf.API.URL.API != "" {
			if err := api.New().CheckConfiguration(conf.API); err != nil {
				fmt.Println(err)
				hasError = true
			}
		}

		if conf.Hatchery.Local.API.HTTP.URL != "" {
			if err := local.New().CheckConfiguration(conf.Hatchery.Local); err != nil {
				fmt.Println(err)
				hasError = true
			}
		}

		if conf.Hatchery.Docker.API.HTTP.URL != "" {
			if err := docker.New().CheckConfiguration(conf.Hatchery.Docker); err != nil {
				fmt.Println(err)
				hasError = true
			}
		}

		if conf.Hatchery.Marathon.API.HTTP.URL != "" {
			if err := marathon.New().CheckConfiguration(conf.Hatchery.Marathon); err != nil {
				fmt.Println(err)
				hasError = true
			}
		}

		if conf.Hatchery.Openstack.API.HTTP.URL != "" {
			if err := openstack.New().CheckConfiguration(conf.Hatchery.Openstack); err != nil {
				fmt.Println(err)
				hasError = true
			}
		}

		if conf.Hatchery.Swarm.API.HTTP.URL != "" {
			if err := openstack.New().CheckConfiguration(conf.Hatchery.Swarm); err != nil {
				fmt.Println(err)
				hasError = true
			}
		}

		if !hasError {
			fmt.Println("Configuration file OK")
		}
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
	 * Vsphere

Start all of this with a single command:
	$ engine start [api] [hatchery:local] [hatchery:docker] [hatchery:marathon] [hatchery:openstack] [hatchery:swarm] [hatchery:vsphere]
All the services are using the same configuration file format.
You have to specify where the toml configuration is. It can be a local file, provided by consul or vault.
You can also use or override toml file with environment variable.

See $ engine config command for more details.

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
			var s Service
			var cfg interface{}

			fmt.Printf("Starting service %s\n", a)

			switch a {
			case "api":
				s = api.New()
				cfg = conf.API
			case "hatchery:docker":
				s = local.New()
				cfg = conf.Hatchery.Docker
			case "hatchery:local":
				s = local.New()
				cfg = conf.Hatchery.Local
			case "hatchery:marathon":
				s = local.New()
				cfg = conf.Hatchery.Marathon
			case "hatchery:openstack":
				s = local.New()
				cfg = conf.Hatchery.Openstack
			case "hatchery:swarm":
				s = local.New()
				cfg = conf.Hatchery.Swarm
			case "hatchery:vsphere":
				s = vsphere.New()
				cfg = conf.Hatchery.VSphere
			}

			go start(ctx, s, cfg)

			//Stupid trick: when API is starting wait a bit before start the other
			if a == "API" {
				time.Sleep(2 * time.Second)
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

func start(c context.Context, s Service, cfg interface{}) {
	if err := s.ApplyConfiguration(cfg); err != nil {
		sdk.Exit("Unable to init service: %v", err)
	}
	if err := s.Serve(c); err != nil {
		sdk.Exit("Service has been stopped: %v", err)
	}
}
