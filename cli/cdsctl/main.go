package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	configFile string
	cfg        *cdsclient.Config
	client     cdsclient.Interface
	root       *cobra.Command
)

func main() {
	root = rootFromSubCommands([]*cobra.Command{
		doc(),         // hidden command
		accesstoken(), // experimental command
		action(),
		login(),             // nearly deprecated
		loginExperimental(), // experimental command to handle JWT
		signup(),
		application(),
		environment(),
		events(),
		pipeline(),
		group(),
		health(),
		project(),
		worker(),
		workflow(),
		update(),
		usr(),
		shell(),
		monitoring(),
		version(),
		encrypt(),
		token(), // nearly deprecated
		template(),
		admin(),
		tools(),
	})
	if err := root.Execute(); err != nil {
		cli.ExitOnError(err)
	}
}

func rootFromSubCommands(cmds []*cobra.Command) *cobra.Command {
	root := cli.NewCommand(mainCmd, mainRun, cmds)

	root.PersistentFlags().StringVarP(&configFile, "file", "f", "", "set configuration file")
	root.PersistentFlags().BoolP("verbose", "", false, "verbose output")
	root.PersistentFlags().BoolP("insecure", "", false, `(SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.`)

	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		var err error
		cfg, err = loadConfig(cmd, configFile)

		if err == nil && cfg != nil {
			client = cdsclient.New(*cfg)
		}

		//Do not load config on login
		if cmd.Name() == "login" ||
			cmd.Name() == "signup" ||
			cmd.Name() == "version" ||
			cmd.Name() == "doc" || strings.HasPrefix(cmd.Use, "doc ") || (cmd.Run == nil && cmd.RunE == nil) {
			return
		}

		cli.ExitOnError(err, login().Help)
	}

	return root
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

	CDS_API_URL="https://instance.cds.api" CDS_USER="username" CDS_TOKEN="yourtoken" cdsctl [command]


Want to debug something? You can use ` + "`CDS_VERBOSE`" + ` environment variable.

	CDS_VERBOSE=true cdsctl [command]


If you're using a self-signed certificate on CDS API, you probably want to use ` + "`CDS_INSECURE`" + ` variable.

	CDS_INSECURE=true cdsctl [command]

`,
}

func mainRun(vals cli.Values) error {
	fmt.Println("Welcome to CDS")

	urlUI, err := client.ConfigUser()
	if err != nil {
		return nil
	}

	var uiURL string
	if b, ok := urlUI[sdk.ConfigURLUIKey]; ok {
		uiURL = b
		fmt.Printf("UI: %s\n", uiURL)
	}

	navbarInfos, err := client.Navbar()
	if err != nil {
		return nil
	}

	projFavs := []sdk.NavbarProjectData{}
	wfFavs := []sdk.NavbarProjectData{}
	for _, elt := range navbarInfos {
		if elt.Favorite {
			switch elt.Type {
			case "workflow":
				wfFavs = append(wfFavs, elt)
			case "project":
				projFavs = append(projFavs, elt)
			}
		}
	}

	fmt.Println("\n -=-=-=-=- Projects bookmarked -=-=-=-=-")
	for _, prj := range projFavs {
		fmt.Printf("- %s %s\n", prj.Name, uiURL+"/project/"+prj.Key)
	}

	fmt.Println("\n -=-=-=-=- Workflows bookmarked -=-=-=-=-")
	for _, wf := range wfFavs {
		fmt.Printf("- %s %s\n", wf.WorkflowName, uiURL+"/project/"+wf.Key+"/workflow/"+wf.WorkflowName)
	}

	return nil
}
