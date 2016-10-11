package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdUserGenerate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "cds user generate",
		Long:  ``,
	}

	cmd.AddCommand(cmdUserGenerateWorker())
	return cmd
}

func cmdUserGenerateWorker() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "cds user generate worker",
		Long:  ``,
	}

	cmd.AddCommand(cmdUserGenerateWorkerKey())
	return cmd
}

func cmdUserGenerateWorkerKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "cds user generate worker key <expiry>",
		Long: `cds user generate worker key <expiry>

Available expiry options:
 - never
 - firstuse
`,
		Run: generateWorkerKey,
	}

	return cmd
}

func generateWorkerKey(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage, see:\n%s\n", cmd.Long)
	}

	var e sdk.Expiry
	switch args[0] {
	case "never":
		e = sdk.NeverExpire
	case "firstuse":
		e = sdk.FirstUseExpire
	default:
		sdk.Exit("Invalid expiry option")
	}

	key, err := sdk.GenerateWorkerKey(e)
	if err != nil {
		sdk.Exit("Error: cannot generate key (%s)\n", err)
	}

	fmt.Printf("%s\n", key)
}
