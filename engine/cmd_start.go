package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/services"
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
	"github.com/ovh/cds/engine/ui"
	"github.com/ovh/cds/engine/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/spf13/cobra"
)

func init() {
	startCmd.Flags().StringVar(&flagStartConfigFile, "config", "", "config file")
	startCmd.Flags().StringVar(&flagStartRemoteConfig, "remote-config", "", "(optional) consul configuration store")
	startCmd.Flags().StringVar(&flagStartRemoteConfigKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")
	startCmd.Flags().StringVar(&flagStartVaultAddr, "vault-addr", "", "(optional) Vault address to fetch secrets from vault (example: https://vault.mydomain.net:8200)")
	startCmd.Flags().StringVar(&flagStartVaultToken, "vault-token", "", "(optional) Vault token to fetch secrets from vault")
}

var (
	flagStartConfigFile      string
	flagStartRemoteConfig    string
	flagStartRemoteConfigKey string
	flagStartVaultAddr       string
	flagStartVaultToken      string
)

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

	$ engine start [api] [hatchery:local] [hatchery:marathon] [hatchery:openstack] [hatchery:swarm] [hatchery:vsphere] [elasticsearch] [hooks] [vcs] [repositories] [migrate] [ui]

All the services are using the same configuration file format.

You have to specify where the toml configuration is. It can be a local file, provided by consul or vault.

You can also use or override toml file with environment variable.

See $ engine config command for more details.

`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help() // nolint
			return
		}

		// Initialize config
		conf := configImport(args, flagStartConfigFile, flagStartRemoteConfig, flagStartRemoteConfigKey, flagStartVaultAddr, flagStartVaultToken, false)
		ctx, cancel := context.WithCancel(context.Background())

		// initialize context
		defer cancel()

		// gracefully shutdown all
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
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

		var (
			serviceConfs []serviceConf
			names        []string
			types        []string
		)

		for _, a := range args {
			fmt.Printf("Starting service %s\n", a)
			switch a {
			case "api":
				if conf.API == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: api.New(), cfg: *conf.API})
				names = append(names, conf.API.Name)
				types = append(types, services.TypeAPI)

			case "ui":
				if conf.UI == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: ui.New(), cfg: *conf.UI})
				names = append(names, conf.UI.Name)
				types = append(types, services.TypeUI)

			case "migrate":
				if conf.DatabaseMigrate == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: migrateservice.New(), cfg: *conf.DatabaseMigrate})
				names = append(names, conf.DatabaseMigrate.Name)
				types = append(types, services.TypeDBMigrate)

			case "hatchery:local":
				if conf.Hatchery.Local == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: local.New(), cfg: *conf.Hatchery.Local})
				names = append(names, conf.Hatchery.Local.Name)
				types = append(types, services.TypeHatchery)

			case "hatchery:kubernetes":
				if conf.Hatchery.Kubernetes == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: kubernetes.New(), cfg: *conf.Hatchery.Kubernetes})
				names = append(names, conf.Hatchery.Kubernetes.Name)
				types = append(types, services.TypeHatchery)

			case "hatchery:marathon":
				if conf.Hatchery.Marathon == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: marathon.New(), cfg: *conf.Hatchery.Marathon})
				names = append(names, conf.Hatchery.Marathon.Name)
				types = append(types, services.TypeHatchery)

			case "hatchery:openstack":
				if conf.Hatchery.Openstack == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: openstack.New(), cfg: *conf.Hatchery.Openstack})
				names = append(names, conf.Hatchery.Openstack.Name)
				types = append(types, services.TypeAPI)

			case "hatchery:swarm":
				if conf.Hatchery.Swarm == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: swarm.New(), cfg: *conf.Hatchery.Swarm})
				names = append(names, conf.Hatchery.Swarm.Name)
				types = append(types, services.TypeHatchery)

			case "hatchery:vsphere":
				if conf.Hatchery.VSphere == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: vsphere.New(), cfg: *conf.Hatchery.VSphere})
				names = append(names, conf.Hatchery.VSphere.Name)
				types = append(types, services.TypeHatchery)

			case "hooks":
				if conf.Hooks == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: hooks.New(), cfg: *conf.Hooks})
				names = append(names, conf.Hooks.Name)
				types = append(types, services.TypeHooks)

			case "vcs":
				if conf.VCS == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: vcs.New(), cfg: *conf.VCS})
				names = append(names, conf.VCS.Name)
				types = append(types, services.TypeVCS)

			case "repositories":
				if conf.Repositories == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: repositories.New(), cfg: *conf.Repositories})
				names = append(names, conf.Repositories.Name)
				types = append(types, services.TypeRepositories)

			case "elasticsearch":
				if conf.ElasticSearch == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: elasticsearch.New(), cfg: *conf.ElasticSearch})
				names = append(names, conf.ElasticSearch.Name)
				types = append(types, services.TypeElasticsearch)

			default:
				fmt.Printf("Error: service '%s' unknown\n", a)
				os.Exit(1)
			}
		}

		//Initialize logs
		logConf := log.Conf{
			Level:                      conf.Log.Level,
			GraylogProtocol:            conf.Log.Graylog.Protocol,
			GraylogHost:                conf.Log.Graylog.Host,
			GraylogPort:                fmt.Sprintf("%d", conf.Log.Graylog.Port),
			GraylogExtraKey:            conf.Log.Graylog.ExtraKey,
			GraylogExtraValue:          conf.Log.Graylog.ExtraValue,
			GraylogFieldCDSVersion:     sdk.VERSION,
			GraylogFieldCDSOS:          sdk.GOOS,
			GraylogFieldCDSArch:        sdk.GOARCH,
			GraylogFieldCDSServiceName: strings.Join(names, "_"),
			GraylogFieldCDSServiceType: strings.Join(types, "_"),
			Ctx:                        ctx,
		}
		// TODO Logger: each service should have it own logger
		log.Initialize(&logConf)

		// Sort the slice of services we have to start to be sure to start the API au first
		sort.Slice(serviceConfs, func(i, j int) bool {
			return serviceConfs[i].arg < serviceConfs[j].arg
		})

		//Configure the services
		for _, s := range serviceConfs {
			if err := s.service.ApplyConfiguration(s.cfg); err != nil {
				sdk.Exit("Unable to init service %s: %v", s.arg, err)
			}

			if srv, ok := s.service.(service.BeforeStart); ok {
				if err := srv.BeforeStart(); err != nil {
					sdk.Exit("Unable to start service %s: %v", s.arg, err)
				}
			}

			c, err := observability.Init(ctx, conf.Telemetry, s.service)
			if err != nil {
				sdk.Exit("Unable to start tracing exporter: %v", err)
			}

			go start(c, s.service, s.cfg, s.arg)

			// Stupid trick: when API is starting wait a bit before start the other
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
		log.Error(ctx, "%s> Unable to start service: %v", serviceName, err)
		return err
	}

	var srvConfig sdk.ServiceConfig
	b, _ := json.Marshal(cfg)
	json.Unmarshal(b, &srvConfig) // nolint

	// then register
	if err := s.Register(srvConfig); err != nil {
		log.Error(ctx, "%s> Unable to register: %v", serviceName, err)
		return err
	}
	log.Info("%s> Service registered", serviceName)

	// finally start the heartbeat goroutine
	go func() {
		if err := s.Heartbeat(ctx, s.Status); err != nil {
			log.Error(ctx, "%v", err)
			cancel()
		}
	}()

	go func() {
		if err := s.Serve(ctx); err != nil {
			log.Error(ctx, "%s> Serve: %+v", serviceName, err)
			cancel()
		}
	}()

	<-ctx.Done()

	if ctx.Err() != nil {
		log.Error(ctx, "%s> Service exiting with err: %+v", serviceName, ctx.Err())
	} else {
		log.Info("%s> Service exiting", serviceName)
	}
	return ctx.Err()
}
