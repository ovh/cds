package model

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdWorkerModelAdd())
	Cmd.AddCommand(cmdWorkerModelRemove())
	Cmd.AddCommand(cmdWorkerModelUpdate())
	Cmd.AddCommand(cmdWorkerModelList())
	Cmd.AddCommand(cmdWorkerModelCapability())
}

// Cmd model
var Cmd = &cobra.Command{
	Use:   "model",
	Short: "",
	Long:  ``,
}

func cmdWorkerModelCapability() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "capability",
		Short:   "",
		Long:    ``,
		Aliases: []string{"capa"},
	}

	cmd.AddCommand(cmdWorkerModelCapabilityAdd())
	cmd.AddCommand(cmdWorkerModelCapabilityUpdate())
	cmd.AddCommand(cmdWorkerModelCapabilityRemove())
	return cmd
}
