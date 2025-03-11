package main

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	cfg            *cdsclient.Config
	configFilePath string
	client         cdsclient.Interface
	root           *cobra.Command
)

func main() {
	root = rootFromSubCommands([]*cobra.Command{
		doc(), // hidden command
		action(),
		admin(),
		application(),
		consumer(),
		encrypt(),
		contexts(),
		environment(),
		experimental(),
		events(),
		group(),
		health(),
		login(),
		reset(),
		signup(),
		pipeline(),
		project(),
		queue(),
		template(),
		tools(),
		update(),
		usr(),
		session(),
		version(),
		worker(),
		workflow(),
	})
	if err := root.Execute(); err != nil {
		cli.ExitOnError(err)
	}
}

func rootFromSubCommands(cmds []*cobra.Command) *cobra.Command {
	root := cli.NewCommand(mainCmd, nil, cmds)

	root.PersistentFlags().StringP("context", "c", "", "cdsctl context name")
	root.PersistentFlags().StringP("file", "f", "", "set configuration file")
	root.PersistentFlags().BoolP("no-interactive", "n", false, "Set to disable interaction with ctl")
	root.PersistentFlags().BoolP("verbose", "", false, "Enable verbose output")
	root.PersistentFlags().BoolP("insecure", "", false, `(SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.`)

	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		var err error
		configFilePath, cfg, err = loadConfig(cmd)

		if err == nil && cfg != nil {
			client = cdsclient.New(*cfg)
		}

		// Do not load config on login
		if cmd.Name() == "login" ||
			cmd.Name() == "signup" ||
			cmd.Name() == "verify" ||
			cmd.Name() == "reset-password" ||
			cmd.Name() == "confirm" ||
			cmd.Name() == "version" ||
			cmd.Name() == "doc" || strings.HasPrefix(cmd.Use, "doc ") || (cmd.Run == nil && cmd.RunE == nil) {
			return
		}

		cli.ExitOnError(err, root.Help)
	}

	return root
}

var mainCmd = cli.Command{
	Name:  "cdsctl",
	Short: "CDS Command line utility",
	Long: `

## Download

You can find the cdsctl version corresponding to your CDS version on the UI: Menu -> Settings -> Command line cdsctl. You'll also find the instructions to configure your cdsctl according to your CDS instance.

The latest release is available on [GitHub Releases](https://github.com/ovh/cds/releases/latest).


## Authentication

Per default, the command line ` + "`cdsctl`" + ` uses your keychain on your os:

* OSX: Keychain Access
* Linux System: Secret-tool (libsecret)

You can use a "sign in" token attached to a consumer:

	CDS_API_URL="https://instance.cds.api" CDS_TOKEN="token-consumer" cdsctl [command]


Want to debug something? You can use ` + "`CDS_VERBOSE`" + ` environment variable.

	CDS_VERBOSE=true cdsctl [command]


If you're using a self-signed certificate on CDS API, you probably want to use ` + "`CDS_INSECURE`" + ` variable.

	CDS_INSECURE=true cdsctl [command]

Advanced usages:

* you can use a session-token instead of a token:

	CDS_API_URL="https://instance.cds.api" CDS_USER="username" CDS_SESSION_TOKEN="yourtoken" cdsctl [command]

* you define a maximum number of retries for HTTP calls:

	CDS_API_URL="https://instance.cds.api" CDS_SESSION_TOKEN="yourtoken" CDS_HTTP_MAX_RETRY=10 cdsctl [command]

* you can override CDN url if needed using the variable CDS_CDN_URL:

  CDS_API_URL="https://instance.cds.api" CDS_TOKEN="yourtoken" CDS_CDN_URL="https://instance.cds.cdn" cdsctl [command]

`,
}
