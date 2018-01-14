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
	root                  *cobra.Command
)

func main() {
	login := cli.NewCommand(loginCmd, loginRun, nil, cli.CommandWithoutExtraFlags)
	signup := cli.NewCommand(signupCmd, signupRun, nil, cli.CommandWithoutExtraFlags)
	update := cli.NewCommand(updateCmd, updateRun, nil, cli.CommandWithoutExtraFlags)
	version := cli.NewCommand(versionCmd, versionRun, nil, cli.CommandWithoutExtraFlags)
	doc := cli.NewCommand(docCmd, docRun, nil, cli.CommandWithoutExtraFlags)
	monitoring := cli.NewGetCommand(monitoringCmd, monitoringRun, nil, cli.CommandWithoutExtraFlags)

	root = cli.NewCommand(mainCmd, mainRun,
		[]*cobra.Command{
			doc, // hidden command
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
			update,
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
		//Do not load config on login
		if cmd == login || cmd == signup || (cmd.Run == nil && cmd.RunE == nil) {
			return
		}

		var err error
		cfg, err = loadConfig(configFile)
		cli.ExitOnError(err, login.Help)

		client = cdsclient.New(*cfg)
	}

	if err := root.Execute(); err != nil {
		cli.ExitOnError(err)
	}
}

var mainCmd = cli.Command{
	Name:  "cdsctl",
	Short: "CDS Command line utility",
	Long: `

## Download

You'll find last release of ` + "`cdsctl`" + ` on [Github Releases](https://github.com/ovh/cds/releases/latest).


## Authentication

Per default, the command line ` + "`cdsctl`" + ` uses your keychain on your os:

* OSX: Keychain Access
* Linux System: Secret-tool (libsecret) 
* Windows: Windows Credentials service

You can bypass keychain tools by using environment variables:

	CDS_API_URL="https://instance.cds.api"  CDS_USER="username" CDS_TOKEN="yourtoken" cdsctl [command]


Want to debug something? You can use ` + "`CDS_VERBOSE`" + ` environment variable.

	CDS_VERBOSE=true cdsctl [command]


If you're using a self-signed certificate on CDS API, you probably want to use ` + "`CDS_INSECURE`" + ` variable.

	CDS_INSECURE=true cdsctl [command]

`,
}

func mainRun(vals cli.Values) error {
	fmt.Println("Welcome to CDS")
	return nil
}
