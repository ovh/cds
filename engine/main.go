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
)

type Configuration struct {
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
	mainCmd.Flags().StringVar(&cfgFile, "config", "", "config file")
	mainCmd.Flags().StringVar(&remoteCfg, "remote-config", "", "(optional) consul configuration store")
	mainCmd.Flags().StringVar(&remoteCfgKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")
	mainCmd.Flags().StringVar(&vaultAddr, "vault-addr", "", "(optional) Vault address to fetch secrets from vault (example: https://vault.mydomain.net:8200)")
	mainCmd.Flags().StringVar(&vaultToken, "vault-token", "", "(optional) Vault token to fetch secrets from vault")
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
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "engine config",
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
	Short: "engine start [api] [hatchery:local]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}

		//Init config
		config()

		//Initliaze context
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
