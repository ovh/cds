package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	configFile            string
	cfg                   *cdsclient.Config
	verbose               bool
	noWarnings            bool
	insecureSkipVerifyTLS bool
	client                cdsclient.Interface
)

func main() {
	login := cli.NewCommand(loginCmd, loginRun, nil, cli.CommandWithoutExtraFlags)
	signup := cli.NewCommand(signupCmd, signupRun, nil, cli.CommandWithoutExtraFlags)
	version := cli.NewCommand(versionCmd, versionRun, nil, cli.CommandWithoutExtraFlags)
	monitoring := cli.NewGetCommand(monitoringCmd, monitoringRun, nil, cli.CommandWithoutExtraFlags)

	root := cli.NewCommand(mainCmd, mainRun,
		[]*cobra.Command{
			action,
			login,
			signup,
			application,
			environment,
			pipeline,
			group,
			health,
			project,
			worker,
			workflow,
			usr,
			monitoring,
			health,
			version,
		},
	)

	root.PersistentFlags().StringVarP(&configFile, "file", "f", "", "set configuration file")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	root.PersistentFlags().BoolVarP(&noWarnings, "no-warnings", "w", false, "do not display warnings")
	root.PersistentFlags().BoolVarP(&insecureSkipVerifyTLS, "insecure", "k", false, `(SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.`)
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		//Do not load config nor display warnings on login
		if cmd == login || cmd == signup || (cmd.Run == nil && cmd.RunE == nil) {
			return
		}

		var err error
		cfg, err = loadConfig(configFile)
		cli.ExitOnError(err, login.Help)

		if cfg.Host != "" && cfg.User != "" && cfg.Token != "" {
			client = cdsclient.New(*cfg)
		} else {
			client, err = loadClient(cfg)
			cli.ExitOnError(err)
		}

		//Manage warnings
		/*		if !internal.NoWarnings && cmd != user.Cmd {
					displayWarnings()
				}
		*/
	}

	if err := root.Execute(); err != nil {
		cli.ExitOnError(err)
	}

}

var mainCmd = cli.Command{
	Name:  "cdsctl",
	Short: "CDS Command line utility",
}

func mainRun(vals cli.Values) error {
	fmt.Println("Welcome to CDS")
	return nil
}
