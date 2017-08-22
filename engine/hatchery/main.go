package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/gops/agent"

	"github.com/ovh/cds/engine/hatchery/docker"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/marathon"
	"github.com/ovh/cds/engine/hatchery/openstack"
	"github.com/ovh/cds/engine/hatchery/swarm"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "hatchery",
	Short: "hatchery <mode> --api=<cds.domain> --token=<token>",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Initialize(&log.Conf{
			Level:             viper.GetString("log_level"),
			GraylogProtocol:   viper.GetString("graylog_protocol"),
			GraylogHost:       viper.GetString("graylog_host"),
			GraylogPort:       viper.GetString("graylog_port"),
			GraylogExtraKey:   viper.GetString("graylog_extra_key"),
			GraylogExtraValue: viper.GetString("graylog_extra_value"),
		})

		if cmd.Name() == "version" {
			// no check other args for ./hatchery version
			return
		}

		sdk.SetAgent(sdk.HatcheryAgent)

		if viper.GetInt("max-worker") < 1 {
			sdk.Exit("max-worker have to be > 0\n")
		}

		if viper.GetString("api") == "" {
			sdk.Exit("CDS api endpoint not provided. See help on flag --api\n")
		}

		if viper.GetString("token") == "" {
			sdk.Exit("Worker token not provided. See help on flag --token\n")
		}

		if viper.GetString("remote-debug-url") != "" {
			log.Info("Starting gops agent on %s", viper.GetString("remote-debug-url"))
			if err := agent.Listen(&agent.Options{Addr: viper.GetString("remote-debug-url")}); err != nil {
				sdk.Exit("Error on starting gops agent", err)
			}
		}
	},
}

var (
	//VERSION is set with -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}"
	VERSION = "snapshot"
)

func main() {
	addFlags()
	addCommands()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func addCommands() {
	rootCmd.AddCommand(local.Cmd)
	rootCmd.AddCommand(docker.Cmd)
	rootCmd.AddCommand(marathon.Cmd)
	rootCmd.AddCommand(swarm.Cmd)
	rootCmd.AddCommand(openstack.Cmd)
	rootCmd.AddCommand(cmdVersion)
}

// Cannot rely on viper.AutomaticEnv here because of the presence of hyphen '-'
// in the flag (ie. for openstack cmd).
// AutomaticEnv will only lookup for a strings.ToUpper
// which will not replace hypen '-' with underscore '_'.
// i.e "openstack-tenant" -> "CDS_OPENSTACK-TENANT" instead of "CDS_OPENSTACK_TENANT"
// Hence SetEnvKeyReplacer.
// Sadly, this isn't used by Flags().*Var functions, so no automatic binding on variables.
func addFlags() {
	viper.SetEnvPrefix("cds")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	rootCmd.PersistentFlags().String("log-level", "notice", "Log Level: debug, info, warning, notice, critical")
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	rootCmd.PersistentFlags().String("api", "", "CDS api endpoint")
	viper.BindPFlag("api", rootCmd.PersistentFlags().Lookup("api"))

	rootCmd.PersistentFlags().String("token", "", "CDS token")
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))

	rootCmd.PersistentFlags().Int("request-api-timeout", 10, "Request CDS API: timeout in seconds")
	viper.BindPFlag("request-api-timeout", rootCmd.PersistentFlags().Lookup("request-api-timeout"))

	rootCmd.PersistentFlags().Bool("provision-disabled", false, "Disabled provisionning")
	viper.BindPFlag("provision-disabled", rootCmd.PersistentFlags().Lookup("provision-disabled"))

	rootCmd.PersistentFlags().Int("provision-seconds", 30, "Check provisioning each n Seconds")
	viper.BindPFlag("provision-seconds", rootCmd.PersistentFlags().Lookup("provision-seconds"))

	rootCmd.PersistentFlags().Int("register-seconds", 60, "Check if some worker model have to be registered each n Seconds")
	viper.BindPFlag("register-seconds", rootCmd.PersistentFlags().Lookup("register-seconds"))

	rootCmd.PersistentFlags().Int("max-worker", 10, "Maximum allowed simultaneous workers")
	viper.BindPFlag("max-worker", rootCmd.PersistentFlags().Lookup("max-worker"))

	rootCmd.PersistentFlags().Int("max-failures-heartbeat", 10, "Maximum allowed consecutives failures on heatbeat routine")
	viper.BindPFlag("max-failures-heartbeat", rootCmd.PersistentFlags().Lookup("max-failures-heartbeat"))

	rootCmd.PersistentFlags().BoolP("insecure", "k", false, `(SSL) This option explicitly allows hatchery to perform "insecure" SSL connections on CDS API.`)
	viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))

	rootCmd.PersistentFlags().String("name", "", "The name for hatchery <name>-<type>")
	viper.BindPFlag("name", rootCmd.PersistentFlags().Lookup("name"))

	rootCmd.PersistentFlags().Int64("grace-time-queued", 4, "if worker is queued less than this value (seconds), hatchery does not take care of it")
	viper.BindPFlag("grace-time-queued", rootCmd.PersistentFlags().Lookup("grace-time-queued"))

	rootCmd.PersistentFlags().String("graylog-protocol", "", "Ex: --graylog-protocol=xxxx-yyyy")
	viper.BindPFlag("graylog_protocol", rootCmd.PersistentFlags().Lookup("graylog-protocol"))

	rootCmd.PersistentFlags().String("graylog-host", "", "Ex: --graylog-host=xxxx-yyyy")
	viper.BindPFlag("graylog_host", rootCmd.PersistentFlags().Lookup("graylog-host"))

	rootCmd.PersistentFlags().String("graylog-port", "", "Ex: --graylog-port=12202")
	viper.BindPFlag("graylog_port", rootCmd.PersistentFlags().Lookup("graylog-port"))

	rootCmd.PersistentFlags().String("graylog-extra-key", "", "Ex: --graylog-extra-key=xxxx-yyyy")
	viper.BindPFlag("graylog_extra_key", rootCmd.PersistentFlags().Lookup("graylog-extra-key"))

	rootCmd.PersistentFlags().String("graylog-extra-value", "", "Ex: --graylog-extra-value=xxxx-yyyy")
	viper.BindPFlag("graylog_extra_value", rootCmd.PersistentFlags().Lookup("graylog-extra-value"))

	rootCmd.PersistentFlags().String("worker-graylog-protocol", "", "Ex: --worker-graylog-protocol=xxxx-yyyy")
	viper.BindPFlag("worker_graylog_protocol", rootCmd.PersistentFlags().Lookup("worker-graylog-protocol"))

	rootCmd.PersistentFlags().String("worker-graylog-host", "", "Ex: --worker-graylog-host=xxxx-yyyy")
	viper.BindPFlag("worker_graylog_host", rootCmd.PersistentFlags().Lookup("worker-graylog-host"))

	rootCmd.PersistentFlags().String("worker-graylog-port", "", "Ex: --worker-graylog-port=12202")
	viper.BindPFlag("worker_graylog_port", rootCmd.PersistentFlags().Lookup("worker-graylog-port"))

	rootCmd.PersistentFlags().String("worker-graylog-extra-key", "", "Ex: --worker-graylog-extra-key=xxxx-yyyy")
	viper.BindPFlag("worker_graylog_extra_key", rootCmd.PersistentFlags().Lookup("worker-graylog-extra-key"))

	rootCmd.PersistentFlags().String("worker-graylog-extra-value", "", "Ex: --worker-graylog-extra-value=xxxx-yyyy")
	viper.BindPFlag("worker_graylog_extra_value", rootCmd.PersistentFlags().Lookup("graylog-extra-value"))

	rootCmd.PersistentFlags().String("grpc-api", "", "CDS GRPC tcp address")
	viper.BindPFlag("grpc_api", rootCmd.PersistentFlags().Lookup("grpc-api"))

	rootCmd.PersistentFlags().Bool("grpc-insecure", false, "Disable GRPC TLS encryption")
	viper.BindPFlag("grpc_insecure", rootCmd.PersistentFlags().Lookup("grpc-insecure"))

	rootCmd.PersistentFlags().String("remote-debug-url", "", "If not empty, start a gops agent on specified URL. Ex: --remote-debug-url=localhost:9999")
	viper.BindPFlag("remote-debug-url", rootCmd.PersistentFlags().Lookup("remote-debug-url"))
}
