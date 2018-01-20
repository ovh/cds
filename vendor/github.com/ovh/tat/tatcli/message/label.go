package message

import (
	"strings"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdMessageLabel = &cobra.Command{
	Use:   "label",
	Short: "Add a label to a message: tatcli message label <topic> <idMessage> <colorInHexa> <my Label>",
	Long: `Add a label to a message:
	tatcli message label <topic> <inReplyOfId> <my message...>
	Example in bash:
	tatcli message label /MyTopic 56bde943968b970001bac20a \#EEEEEE my White Label
	or works too:
	tatcli message label /MyTopic 56bde943968b970001bac20a EEEEEE my White Label
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 4 {
			text := strings.Join(args[3:], " ")
			color := args[2]
			if !strings.HasPrefix(color, "#") {
				color = "#" + color
			}
			out, err := internal.Client().MessageLabel(args[0], args[1], tat.Label{Text: text, Color: color})
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to add a label: tatcli message label --help\n")
		}
	},
}
