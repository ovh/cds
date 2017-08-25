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
	health := cli.NewGetCommand(healthCmd, healthRun, nil, cli.CommandWithoutExtraFlags)

	root := cli.NewCommand(mainCmd, mainRun,
		[]*cobra.Command{
			application,
			environment,
			login,
			signup,
			project,
			worker,
			workflow,
			usr,
			health,
		},
	)

	root.PersistentFlags().StringVarP(&configFile, "file", "f", "", "set configuration file")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	root.PersistentFlags().BoolVarP(&noWarnings, "no-warnings", "w", false, "do not display warnings")
	root.PersistentFlags().BoolVarP(&insecureSkipVerifyTLS, "insecure", "k", false, `(SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.`)
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		//Do not load config nor display warnings on login
		if cmd == login || (cmd.Run == nil && cmd.RunE == nil) {
			return
		}

		config, err := loadConfig(configFile)
		cli.ExitOnError(err, login.Help)

		client, err = loadClient(config)
		cli.ExitOnError(err)

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
	fmt.Println("Welcome on CDS")
	return nil
}
