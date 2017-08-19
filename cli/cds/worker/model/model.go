package model

import "github.com/spf13/cobra"

func init() {
	Cmd.AddCommand(cmdWorkerModelAdd())
	Cmd.AddCommand(cmdWorkerModelRemove())
	Cmd.AddCommand(cmdWorkerModelUpdate())
	Cmd.AddCommand(cmdWorkerModelList())
}

// Cmd model
var Cmd = &cobra.Command{
	Use:   "model",
	Short: "",
	Long:  ``,
}
