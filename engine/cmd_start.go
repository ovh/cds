package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/cdn"
	"github.com/ovh/cds/engine/elasticsearch"
	"github.com/ovh/cds/engine/hatchery/kubernetes"
	"github.com/ovh/cds/engine/hatchery/local"
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
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"

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

type serviceConf struct {
	arg     string
	service service.Service
	cfg     interface{}
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

#### CDN
This component operates CDS CDN to handle storage

Start all of this with a single command:

	$ engine start [api] [cdn] [hatchery:local] [hatchery:openstack] [hatchery:swarm] [hatchery:vsphere] [elasticsearch] [hooks] [vcs] [repositories] [migrate] [ui]

All the services are using the same configuration file format.

You have to specify where the toml configuration is. It can be a local file, provided by consul or vault.

You can also use or override toml file with environment variable.

See $ engine config command for more details.

`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			args = strings.Split(os.Getenv("CDS_SERVICE"), " ")
		}

		if len(args) == 0 {
			cmd.Help() // nolint
			return
		}

		// Initialize config
		conf := configImport(args, flagStartConfigFile, flagStartRemoteConfig, flagStartRemoteConfigKey, flagStartVaultAddr, flagStartVaultToken, false)
		ctx, cancel := context.WithCancel(context.Background())

		// initialize context
		defer cancel()

		var (
			serviceConfs []serviceConf
			names        []string
			types        []string
		)

		for _, a := range args {
			fmt.Printf("Starting service %s\n", a)
			switch a {
			case sdk.TypeAPI:
				if conf.API == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: api.New(), cfg: *conf.API})
				names = append(names, conf.API.Name)
				types = append(types, sdk.TypeAPI)

			case sdk.TypeUI:
				if conf.UI == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: ui.New(), cfg: *conf.UI})
				names = append(names, conf.UI.Name)
				types = append(types, sdk.TypeUI)

			case "migrate":
				if conf.DatabaseMigrate == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: migrateservice.New(), cfg: *conf.DatabaseMigrate})
				names = append(names, conf.DatabaseMigrate.Name)
				types = append(types, sdk.TypeDBMigrate)

			case sdk.TypeHatchery + ":local":
				if conf.Hatchery.Local == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: local.New(), cfg: *conf.Hatchery.Local})
				names = append(names, conf.Hatchery.Local.Name)
				types = append(types, sdk.TypeHatchery)

			case sdk.TypeHatchery + ":kubernetes":
				if conf.Hatchery.Kubernetes == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: kubernetes.New(), cfg: *conf.Hatchery.Kubernetes})
				names = append(names, conf.Hatchery.Kubernetes.Name)
				types = append(types, sdk.TypeHatchery)

			case sdk.TypeHatchery + ":openstack":
				if conf.Hatchery.Openstack == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: openstack.New(), cfg: *conf.Hatchery.Openstack})
				names = append(names, conf.Hatchery.Openstack.Name)
				types = append(types, sdk.TypeAPI)

			case sdk.TypeHatchery + ":swarm":
				if conf.Hatchery.Swarm == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: swarm.New(), cfg: *conf.Hatchery.Swarm})
				names = append(names, conf.Hatchery.Swarm.Name)
				types = append(types, sdk.TypeHatchery)

			case sdk.TypeHatchery + ":vsphere":
				if conf.Hatchery.VSphere == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: vsphere.New(), cfg: *conf.Hatchery.VSphere})
				names = append(names, conf.Hatchery.VSphere.Name)
				types = append(types, sdk.TypeHatchery)

			case sdk.TypeHooks:
				if conf.Hooks == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: hooks.New(), cfg: *conf.Hooks})
				names = append(names, conf.Hooks.Name)
				types = append(types, sdk.TypeHooks)

			case sdk.TypeCDN:
				if conf.CDN == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: cdn.New(), cfg: *conf.CDN})
				names = append(names, conf.CDN.Name)
				types = append(types, sdk.TypeCDN)

			case sdk.TypeVCS:
				if conf.VCS == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: vcs.New(), cfg: *conf.VCS})
				names = append(names, conf.VCS.Name)
				types = append(types, sdk.TypeVCS)

			case sdk.TypeRepositories:
				if conf.Repositories == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: repositories.New(), cfg: *conf.Repositories})
				names = append(names, conf.Repositories.Name)
				types = append(types, sdk.TypeRepositories)

			case sdk.TypeElasticsearch:
				if conf.ElasticSearch == nil {
					sdk.Exit("Unable to start: missing service %s configuration", a)
				}
				serviceConfs = append(serviceConfs, serviceConf{arg: a, service: elasticsearch.New(), cfg: *conf.ElasticSearch})
				names = append(names, conf.ElasticSearch.Name)
				types = append(types, sdk.TypeElasticsearch)

			default:
				fmt.Printf("Error: service '%s' unknown\n", a)
				os.Exit(1)
			}
		}

		// gracefully shutdown all
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func(ctx context.Context) {
			<-c
			unregisterServices(ctx, serviceConfs)
			signal.Stop(c)
			cancel()
		}(ctx)

		//Initialize logs
		logConf := cdslog.Conf{
			Level:                      conf.Log.Level,
			Format:                     conf.Log.Format,
			TextFields:                 conf.Log.TextFields,
			SkipTextFields:             conf.Log.SkipTextFields,
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
		}
		cdslog.Initialize(ctx, &logConf)

		// Sort the slice of services we have to start to be sure to start the API au first
		sort.Slice(serviceConfs, func(i, j int) bool {
			return serviceConfs[i].arg < serviceConfs[j].arg
		})

		// Detect if the API is co-located with other services
		var localAPI service.LocalAPIProvider
		hasNonAPIService := false
		for _, s := range serviceConfs {
			if apiProvider, ok := s.service.(service.LocalAPIProvider); ok {
				localAPI = apiProvider
			} else {
				hasNonAPIService = true
			}
		}

		// Enable gateway mode if the API is co-located with other services
		gatewayMode := localAPI != nil && hasNonAPIService && len(serviceConfs) > 1
		var gw *gateway
		if gatewayMode {
			// Determine gateway port: use the API's HTTP port
			apiPort := conf.API.HTTP.Port
			apiAddr := conf.API.HTTP.Addr
			gatewayBaseURL := fmt.Sprintf("http://%s:%d", apiAddr, apiPort)
			if apiAddr == "" || apiAddr == "0.0.0.0" {
				gatewayBaseURL = fmt.Sprintf("http://localhost:%d", apiPort)
			}

			gw = newGateway(ctx, apiAddr, apiPort)
			setGatewayMode(serviceConfs, gatewayBaseURL)
			log.Info(ctx, "Gateway mode enabled: all services on port %d", apiPort)
		}

		var wg sync.WaitGroup
		//Configure the services
		for i := range serviceConfs {
			s := serviceConfs[i]

			if err := s.service.ApplyConfiguration(s.cfg); err != nil {
				sdk.Exit("Unable to init service %s: %v", s.arg, err)
			}

			serviceCtx := context.WithValue(ctx, cdslog.Service, s.service.Name())
			log.Info(ctx, "%s> %s configuration applied", s.arg, s.service.Name())

			if srv, ok := s.service.(service.BeforeStart); ok {
				if err := srv.BeforeStart(serviceCtx); err != nil {
					sdk.Exit("Unable to start service %s: %v", s.arg, err)
				}
			}

			var err error
			serviceCtx, err = telemetry.Init(serviceCtx, conf.Telemetry, s.service)
			if err != nil {
				sdk.Exit("Unable to start tracing exporter: %v", err)
			}

			// For the API, use standard start; for other co-located services, pass localAPI
			sLocalAPI := localAPI
			if s.service.Type() == sdk.TypeAPI {
				sLocalAPI = nil // API does not need local signin
			}

			wg.Add(1)
			go func(srv serviceConf, localProvider service.LocalAPIProvider) {
				if err := startWithOpts(serviceCtx, srv.service, srv.arg, srv.cfg, localProvider); err != nil {
					log.Error(ctx, "%s> service has been stopped: %+v", srv.arg, err)
				}
				wg.Done()
			}(s, sLocalAPI)

			// When API is starting, wait for it to be ready before starting other services
			if s.arg == "API" || s.arg == "api" {
				if localAPI != nil {
					log.Info(ctx, "Waiting for API router to be ready...")
					waitCtx, waitCancel := context.WithTimeout(ctx, 120*time.Second)
					if err := localAPI.WaitForReady(waitCtx); err != nil {
						waitCancel()
						sdk.Exit("API did not become ready: %v", err)
					}
					waitCancel()
					log.Info(ctx, "API router is ready")
				} else {
					time.Sleep(2 * time.Second)
				}
			}

			// Register the service in the gateway
			if gw != nil {
				gw.register(s.service)
			}
		}

		// Start the gateway HTTP server if enabled
		if gw != nil {
			// Wait for services to initialize their routers (Serve runs in goroutine)
			time.Sleep(3 * time.Second)
			gw.build()

			// Register local handlers for API→service in-process communication
			for _, s := range serviceConfs {
				if hp, ok := s.service.(service.HandlerProvider); ok {
					h := hp.GetHandler()
					if h != nil {
						services.RegisterLocalHandler(s.service.Type(), h, "")
						log.Info(ctx, "Registered local handler for %s", s.service.Type())
					}
				}
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := gw.serve(); err != nil {
					log.Error(ctx, "gateway> %+v", err)
				}
			}()
		}

		// Wait for all services to stop
		wg.Wait()
	},
}

func unregisterServices(ctx context.Context, serviceConfs []serviceConf) {
	// unregister all services
	for i := range serviceConfs {
		s := serviceConfs[i]
		fmt.Printf("%s> Unregister\n", s.service.Name())
		if err := s.service.Unregister(ctx); err != nil {
			log.Error(ctx, "%s> Unable to unregister: %v", s.service.Name(), err)
		}
	}

	if ctx.Err() != nil {
		fmt.Printf("Exiting: %+v\n", ctx.Err())
	}
}

func start(ctx context.Context, s service.Service, serviceName string, cfg interface{}) error {
	return startWithOpts(ctx, s, serviceName, cfg, nil)
}

func startWithOpts(ctx context.Context, s service.Service, serviceName string, cfg interface{}, localAPI service.LocalAPIProvider) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srvConfig, err := s.Init(cfg)
	if err != nil {
		return err
	}

	// If a local API provider is available and this is not the API itself, use local signin
	if localAPI != nil && s.Type() != sdk.TypeAPI {
		apiHandler := localAPI.Handler()
		if apiHandler != nil {
			if sc, ok := s.(service.ServiceCommon); ok {
				if err := sc.GetCommon().LocalSignin(ctx, apiHandler, localAPI.RegisterLocalService, cfg); err != nil {
					return sdk.WrapError(err, "unable to local signin: %s", serviceName)
				}
				log.Info(ctx, "%s> Service signed in locally (in-process)", serviceName)
			}
		}
	}

	// Fall back to regular signin if local signin was not performed
	if !isLocallySignedIn(s) {
		if err := s.Signin(ctx, srvConfig, cfg); err != nil {
			return sdk.WrapError(err, "unable to signin: %s", serviceName)
		}
		log.Info(ctx, "%s> Service signed in", serviceName)
	}

	if err := s.Start(ctx); err != nil {
		return sdk.WrapError(err, "unable to start service: %s", serviceName)
	}

	go func() {
		if err := s.Serve(ctx); err != nil && ctx.Err() == nil {
			log.Error(ctx, "%s> Error serve: %+v", serviceName, err)
			cancel()
		}
	}()

	// finally start the heartbeat goroutine
	go func() {
		if err := s.Heartbeat(ctx, s.Status); err != nil && ctx.Err() == nil {
			log.Error(ctx, "%s> Error heartbeat: %+v", serviceName, err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info(ctx, "%s> Service exiting", serviceName)
	return nil
}

// isLocallySignedIn checks if a service has already been signed in locally
// by verifying if its Client is set (LocalSignin sets Client).
func isLocallySignedIn(s service.Service) bool {
	if sc, ok := s.(service.ServiceCommon); ok {
		return sc.GetCommon().Client != nil
	}
	return false
}
