package trigger

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

func listParamTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "cds trigger params list <srcproject>/<srcapp>/<srcpip>[/<env>] <destproject>/<destapp>/<desstpip>[/<destenv>]",
		Long:  ``,
		Run:   listParamTrigger,
	}

	return cmd
}

func listParamTrigger(cmd *cobra.Command, args []string) {

	data, err := yaml.Marshal(cmdParamPipelineTrigger.Parameters)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))

}
