package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/gops/agent"
	"github.com/spf13/cobra"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api"
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
	"github.com/ovh/cds/sdk/log"
)

var (
	flagStartConfigFile      string
	flagStartRemoteConfig    string
	flagStartRemoteConfigKey string
	flagStartVaultAddr       string
	flagStartVaultToken      string
)

func init() {
	startCmd.Flags().StringVar(&flagStartConfigFile, "config", "", "config file")
	startCmd.Flags().StringVar(&flagStartRemoteConfig, "remote-config", "", "(optional) consul configuration store")
	startCmd.Flags().StringVar(&flagStartRemoteConfigKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")
	startCmd.Flags().StringVar(&flagStartVaultAddr, "vault-addr", "", "(optional) Vault address to fetch secrets from vault (example: https://vault.mydomain.net:8200)")
	startCmd.Flags().StringVar(&flagStartVaultToken, "vault-token", "", "(optional) Vault token to fetch secrets from vault")
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start CDS",
	Long: `
Start CDS Engine Services

#### API

This is the core component of CDS.


#### Hatcheries

They are the components responsible for spawning workers. Supported integrations/orchestrators are:

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

		// Initialize config
		conf := configImport(args, flagStartConfigFile, flagStartRemoteConfig, flagStartRemoteConfigKey, flagStartVaultAddr, flagStartVaultToken)

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

		ctx, cancel := context.WithCancel(context.Background())

		// initialize context
		instance := "cdsinstance"
		if conf.Tracing != nil && conf.Tracing.Name != "" {
			instance = conf.Tracing.Name
		}
		tagCDSInstance, _ := tag.NewKey("cds")
		ctx, _ = tag.New(ctx, tag.Upsert(tagCDSInstance, instance))

		defer cancel()

		// gracefully shutdown all
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
				names = append(names, instance)
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
			GraylogFieldCDSOS:      sdk.GOOS,
			GraylogFieldCDSArch:    sdk.GOARCH,
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
	if err := serve(c, s, serviceName, cfg); err != nil {
		sdk.Exit("Service has been stopped: %s %+v", serviceName, err)
	}
}

func serve(c context.Context, s service.Service, serviceName string, cfg interface{}) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	x, err := s.Init(cfg)
	if err != nil {
		return err
	}

	// first signin
	if err := s.Start(ctx, x); err != nil {
		log.Error("%s> Unable to start service: %v", serviceName, err)
		return err
	}

	var srvConfig sdk.ServiceConfig
	b, _ := json.Marshal(cfg)
	json.Unmarshal(b, &srvConfig) // nolint

	// then register
	if err := s.Register(c, srvConfig); err != nil {
		log.Error("%s> Unable to register: %v", serviceName, err)
		return err
	}
	log.Info("%s> Service registered", serviceName)

	// finally start the heartbeat goroutine
	go func() {
		if err := s.Heartbeat(ctx, s.Status); err != nil {
			log.Error("%v", err)
			cancel()
		}
	}()

	go func() {
		if err := s.Serve(c); err != nil {
			log.Error("%s> Serve: %+v", serviceName, err)
			cancel()
		}
	}()

	<-ctx.Done()

	if ctx.Err() != nil {
		log.Error("%s> Service exiting with err: %+v", serviceName, ctx.Err())
	} else {
		log.Info("%s> Service exiting", serviceName)
	}
	return ctx.Err()
}
