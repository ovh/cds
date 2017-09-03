package repositoriesmanager

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func addReposManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds reposmanager add <STASH|GITHUB> <name> <url> <option=value> ...",
		Long:  ``,
		Run:   addReposManager,
	}

	return cmd
}

func addReposManager(cmd *cobra.Command, args []string) {
	if ok, err := sdk.IsAdmin(); !ok {
		if err != nil {
			fmt.Printf("Error : %v\n", err)
		}
		sdk.Exit("You are not allowed to run this command")
	}

	options := map[string]string{}
	if len(args) < 3 {
		cmd.Help()
		sdk.Exit("Wrong usage")
	}
	for i, arg := range args {
		switch i {
		case 0:
			options["type"] = arg
		case 1:
			options["name"] = arg
		case 2:
			options["url"] = arg
		default:
			o := strings.Split(arg, "=")
			if len(o) != 2 {
				cmd.Help()
				sdk.Exit("Wrong usage")
			}
			options[o[0]] = o[1]
		}
	}

	rm, err := sdk.AddReposManager(options)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}
	fmt.Printf("%s %s %s\n", rm.Type, rm.Name, rm.URL)
}
