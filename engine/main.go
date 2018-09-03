package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/google/gops/agent"
	"github.com/spf13/cobra"
	_ "github.com/spf13/viper/remote"
	"github.com/yesnault/go-toml"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/elasticsearch"
	"github.com/ovh/cds/engine/hatchery/kubernetes"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/marathon"
	"github.com/ovh/cds/engine/hatchery/openstack"
	"github.com/ovh/cds/engine/hatchery/swarm"
	"github.com/ovh/cds/engine/hatchery/vsphere"
	"github.com/ovh/cds/engine/hooks"
	"github.com/ovh/cds/engine/migrateservice"
	"github.com/ovh/cds/engine/repositories"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/doc"
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
	//Update  command
	mainCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateFromGithub, "from-github", false, "Update binary from latest github release")
	updateCmd.Flags().StringVar(&updateURLAPI, "api", "", "Update binary from a CDS Engine API")

	mainCmd.AddCommand(uptodateCmd)
	uptodateCmd.Flags().BoolVar(&updateFromGithub, "from-github", false, "Update binary from latest github release")
	uptodateCmd.Flags().StringVar(&updateURLAPI, "api", "", "Update binary from a CDS Engine API")

	//Database command
	mainCmd.AddCommand(database.DBCmd)
	//Start command
	mainCmd.AddCommand(startCmd)
	//Config command
	mainCmd.AddCommand(configCmd)
	configNewCmd.Flags().BoolVar(&configNewAsEnvFlag, "env", false, "Print configuration as environment variable")

	configCmd.AddCommand(configNewCmd)
	configCmd.AddCommand(configCheckCmd)

	// doc command (hidden command)
	mainCmd.AddCommand(docCmd)
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

## Download

You'll find last release of CDS ` + "`engine`" + ` on [Github Releases](https://github.com/ovh/cds/releases/latest).
`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display CDS version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CDS Engine version:%s os:%s architecture:%s\n", sdk.VERSION, runtime.GOOS, runtime.GOARCH)
	},
}

var docCmd = &cobra.Command{
	Use:    "doc <generation-path> <git-directory>",
	Short:  "generate hugo doc for building http://ovh.github.com/cds",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			cmd.Usage()
			os.Exit(1)
		}
		if err := doc.GenerateDocumentation(mainCmd, args[0], args[1]); err != nil {
			sdk.Exit(err.Error())
		}
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
# generate the whole configuration file
	$ engine config new > conf.toml

# you can compose your file configuration
# this will generate a file configuration containing
# api and hatchery:local µService
	$ engine config new api hatchery:local

# For advanced usage, Debug and Tracing section can be generated as:
	$ engine config new debug tracing [µService(s)...]

# All options
	$ engine config new [debug] [tracing] [api] [hatchery:local] [hatchery:marathon] [hatchery:openstack] [hatchery:swarm] [hatchery:vsphere] [elasticsearch] [hooks] [vcs] [repositories] [migrate]

`,
	Run: func(cmd *cobra.Command, args []string) {
		configBootstrap(args)
		configSetDefaults()

		var sharedInfraToken = sdk.RandomString(128)

		if conf.API != nil {
			conf.API.Auth.SharedInfraToken = sharedInfraToken
			conf.API.Secrets.Key = sdk.RandomString(32)
			conf.API.Providers = append(conf.API.Providers, api.ProviderConfiguration{
				Name:  "sample-provider",
				Token: sdk.RandomString(32),
			})
			conf.API.Services = append(conf.API.Services, api.ServiceConfiguration{
				Name:       "sample-service",
				URL:        "https://ovh.github.io",
				Port:       "443",
				Path:       "/cds",
				HealthPath: "/cds",
				HealthPort: "443",
				HealthURL:  "https://ovh.github.io",
				Type:       "doc",
			})
		} else {
			sharedInfraToken = "enter sharedInfraToken from section [api.auth] here"
		}

		if conf.Hatchery != nil {
			conf.Hatchery.Local.API.Token = sharedInfraToken
			conf.Hatchery.Openstack.API.Token = sharedInfraToken
			conf.Hatchery.VSphere.API.Token = sharedInfraToken
			conf.Hatchery.Swarm.API.Token = sharedInfraToken
			conf.Hatchery.Swarm.DockerEngines = map[string]swarm.DockerEngineConfiguration{
				"sample-docker-engine": {
					Host: "///var/run/docker.sock",
				},
			}
			conf.Hatchery.Marathon.API.Token = sharedInfraToken
		}

		if conf.Hooks != nil {
			conf.Hooks.API.Token = sharedInfraToken
		}

		if conf.Repositories != nil {
			conf.Repositories.API.Token = sharedInfraToken
		}

		if conf.DatabaseMigrate != nil {
			conf.DatabaseMigrate.API.Token = sharedInfraToken
		}

		if conf.VCS != nil {
			conf.VCS.API.Token = sharedInfraToken
			conf.VCS.Servers = map[string]vcs.ServerConfiguration{}
			conf.VCS.Servers["Github"] = vcs.ServerConfiguration{
				URL: "https://github.com",
				Github: &vcs.GithubServerConfiguration{
					ClientID:     "xxxx",
					ClientSecret: "xxxx",
				},
			}
			conf.VCS.Servers["Bitbucket"] = vcs.ServerConfiguration{
				URL: "https://mybitbucket.com",
				Bitbucket: &vcs.BitbucketServerConfiguration{
					ConsumerKey: "xxx",
					PrivateKey:  "xxx",
				},
			}
			conf.VCS.Servers["Gitlab"] = vcs.ServerConfiguration{
				URL: "https://gitlab.com",
				Gitlab: &vcs.GitlabServerConfiguration{
					AppID:  "xxxx",
					Secret: "xxxx",
				},
			}
		}

		if !configNewAsEnvFlag {
			btes, err := toml.Marshal(*conf)
			if err != nil {
				sdk.Exit("%v", err)
			}
			fmt.Println(string(btes))
		} else {
			m := AsEnvVariables(conf, "cds", true)
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
		configBootstrap(args)
		config([]string{})

		var hasError bool
		if conf.API != nil && conf.API.URL.API != "" {
			fmt.Printf("checking api configuration...\n")
			if err := api.New().CheckConfiguration(*conf.API); err != nil {
				fmt.Printf("api Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.DatabaseMigrate != nil && conf.DatabaseMigrate.API.HTTP.URL != "" {
			fmt.Printf("checking migrate configuration...\n")
			if err := api.New().CheckConfiguration(*conf.DatabaseMigrate); err != nil {
				fmt.Printf("migrate Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Local != nil && conf.Hatchery.Local.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:local configuration...\n")
			if err := local.New().CheckConfiguration(*conf.Hatchery.Local); err != nil {
				fmt.Printf("hatchery:local Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Marathon != nil && conf.Hatchery.Marathon.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:marathon configuration...\n")
			if err := marathon.New().CheckConfiguration(*conf.Hatchery.Marathon); err != nil {
				fmt.Printf("hatchery:marathon Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Openstack != nil && conf.Hatchery.Openstack.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:openstack configuration...\n")
			if err := openstack.New().CheckConfiguration(*conf.Hatchery.Openstack); err != nil {
				fmt.Printf("hatchery:openstack Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Kubernetes != nil && conf.Hatchery.Kubernetes.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:kubernetes configuration...\n")
			if err := kubernetes.New().CheckConfiguration(*conf.Hatchery.Kubernetes); err != nil {
				fmt.Printf("hatchery:kubernetes Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Swarm != nil && conf.Hatchery.Swarm.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:swarm configuration...\n")
			if err := swarm.New().CheckConfiguration(*conf.Hatchery.Swarm); err != nil {
				fmt.Printf("hatchery:swarm Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.VSphere != nil && conf.Hatchery.VSphere.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:vsphere configuration...\n")
			if err := vsphere.New().CheckConfiguration(*conf.Hatchery.VSphere); err != nil {
				fmt.Printf("hatchery:vsphere Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.VCS != nil && conf.VCS.API.HTTP.URL != "" {
			fmt.Printf("checking vcs configuration...\n")
			if err := vcs.New().CheckConfiguration(*conf.VCS); err != nil {
				fmt.Printf("vcs Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hooks != nil && conf.Hooks.API.HTTP.URL != "" {
			fmt.Printf("checking hooks configuration...\n")
			if err := hooks.New().CheckConfiguration(*conf.Hooks); err != nil {
				fmt.Printf("hooks Configuration: %v\n", err)
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
Start CDS Engine Services

#### API

This is the core component of CDS.


#### Hatcheries

They are the components responsible for spawning workers. Supported platforms/orchestrators are:

* Local machine
* Openstack
* Docker Swarm
* Openstack
* Vsphere

#### Hooks
This component operates CDS workflow hooks

#### Repositories
This component operates CDS workflow repositories

#### VCS
This component operates CDS VCS connectivity

Start all of this with a single command:

	$ engine start [api] [hatchery:local] [hatchery:marathon] [hatchery:openstack] [hatchery:swarm] [hatchery:vsphere] [elasticsearch] [hooks] [vcs] [repositories] [migrate]

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
		configBootstrap(args)
		configSetDefaults()
		config(args)

		// gops debug
		if conf.Debug.Enable {
			if conf.Debug.RemoteDebugURL != "" {
				log.Info("Starting gops agent on %s", conf.Debug.RemoteDebugURL)
				if err := agent.Listen(&agent.Options{Addr: conf.Debug.RemoteDebugURL}); err != nil {
					log.Error("Error on starting gops agent: %v", err)
				}
			} else {
				log.Info("Starting gops agent locally")
				if err := agent.Listen(nil); err != nil {
					log.Error("Error on starting gops agent locally: %v", err)
				}
			}
		}

		//Initialize context
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Gracefully shutdown all
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
		go func() {
			<-c
			signal.Stop(c)
			cancel()
		}()

		type serviceConf struct {
			arg     string
			service service.Service
			cfg     interface{}
		}
		services := []serviceConf{}

		names := []string{}
		for _, a := range args {
			fmt.Printf("Starting service %s\n", a)
			switch a {
			case "api":
				services = append(services, serviceConf{arg: a, service: api.New(), cfg: *conf.API})
				names = append(names, conf.API.Name)
			case "migrate":
				services = append(services, serviceConf{arg: a, service: migrateservice.New(), cfg: *conf.DatabaseMigrate})
				names = append(names, conf.DatabaseMigrate.Name)
			case "hatchery:local":
				services = append(services, serviceConf{arg: a, service: local.New(), cfg: *conf.Hatchery.Local})
				names = append(names, conf.Hatchery.Local.Name)
			case "hatchery:kubernetes":
				services = append(services, serviceConf{arg: a, service: kubernetes.New(), cfg: *conf.Hatchery.Kubernetes})
				names = append(names, conf.Hatchery.Kubernetes.Name)
			case "hatchery:marathon":
				services = append(services, serviceConf{arg: a, service: marathon.New(), cfg: *conf.Hatchery.Marathon})
				names = append(names, conf.Hatchery.Marathon.Name)
			case "hatchery:openstack":
				services = append(services, serviceConf{arg: a, service: openstack.New(), cfg: *conf.Hatchery.Openstack})
				names = append(names, conf.Hatchery.Openstack.Name)
			case "hatchery:swarm":
				services = append(services, serviceConf{arg: a, service: swarm.New(), cfg: *conf.Hatchery.Swarm})
				names = append(names, conf.Hatchery.Swarm.Name)
			case "hatchery:vsphere":
				services = append(services, serviceConf{arg: a, service: vsphere.New(), cfg: *conf.Hatchery.VSphere})
				names = append(names, conf.Hatchery.VSphere.Name)
			case "hooks":
				services = append(services, serviceConf{arg: a, service: hooks.New(), cfg: *conf.Hooks})
				names = append(names, conf.Hooks.Name)
			case "vcs":
				services = append(services, serviceConf{arg: a, service: vcs.New(), cfg: *conf.VCS})
				names = append(names, conf.VCS.Name)
			case "repositories":
				services = append(services, serviceConf{arg: a, service: repositories.New(), cfg: *conf.Repositories})
				names = append(names, conf.Repositories.Name)
			case "elasticsearch":
				services = append(services, serviceConf{arg: a, service: elasticsearch.New(), cfg: *conf.ElasticSearch})
				names = append(names, conf.ElasticSearch.Name)
			default:
				fmt.Printf("Error: service '%s' unknown\n", a)
				os.Exit(1)
			}
		}

		//Initialize logs
		log.Initialize(&log.Conf{
			Level:                  conf.Log.Level,
			GraylogProtocol:        conf.Log.Graylog.Protocol,
			GraylogHost:            conf.Log.Graylog.Host,
			GraylogPort:            fmt.Sprintf("%d", conf.Log.Graylog.Port),
			GraylogExtraKey:        conf.Log.Graylog.ExtraKey,
			GraylogExtraValue:      conf.Log.Graylog.ExtraValue,
			GraylogFieldCDSVersion: sdk.VERSION,
			GraylogFieldCDSName:    strings.Join(names, "_"),
			Ctx:                    ctx,
		})

		//Configure the services
		for _, s := range services {
			if err := s.service.ApplyConfiguration(s.cfg); err != nil {
				sdk.Exit("Unable to init service %s: %v", s.arg, err)
			}

			if srv, ok := s.service.(service.BeforeStart); ok {
				if err := srv.BeforeStart(); err != nil {
					sdk.Exit("Unable to start service %s: %v", s.arg, err)
				}
			}

			// Initialiaze tracing
			if err := observability.Init(*conf.Tracing, "cds-"+s.arg); err != nil {
				sdk.Exit("Unable to start tracing exporter: %v", err)
			}
		}

		//Start the services
		for _, s := range services {
			go start(ctx, s.service, s.cfg, s.arg)
			//Stupid trick: when API is starting wait a bit before start the other
			if s.arg == "API" || s.arg == "api" {
				time.Sleep(2 * time.Second)
			}
		}

		//Wait for the end
		<-ctx.Done()
		if ctx.Err() != nil {
			fmt.Printf("Exiting (%v)\n", ctx.Err())
		}
	},
}

func start(c context.Context, s service.Service, cfg interface{}, serviceName string) {
	if err := serve(c, s, serviceName); err != nil {
		sdk.Exit("Service has been stopped: %s %v", serviceName, err)
	}
}

func serve(c context.Context, s service.Service, serviceName string) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	//First register(heartbeat)
	if _, err := s.DoHeartbeat(s.Status); err != nil {
		log.Error("%s> Unable to register: %v", serviceName, err)
		return err
	}
	log.Info("%s> Service registered", serviceName)

	//Start the heartbeat goroutine
	go func() {
		if err := s.Heartbeat(ctx, s.Status); err != nil {
			log.Error("%v", err)
			cancel()
		}
	}()

	go func() {
		if err := s.Serve(c); err != nil {
			log.Error("%s> Serve: %v", serviceName, err)
			cancel()
		}
	}()

	<-ctx.Done()

	if ctx.Err() != nil {
		log.Error("%s> Service exiting with err: %v", serviceName, ctx.Err())
	} else {
		log.Info("%s> Service exiting", serviceName)
	}
	return ctx.Err()
}
