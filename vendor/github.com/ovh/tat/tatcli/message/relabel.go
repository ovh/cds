package message

import (
	"strings"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdOptions []string

func init() {
	cmdMessageRelabel.Flags().StringSliceVar(&cmdLabel, "label", nil, "add labels : --label=\"#EEEE;myLabel1,#EEEE;myLabel2\"")
	cmdMessageRelabel.Flags().StringSliceVar(&cmdOptions, "options", nil, "remove only theses Labels on relabel : --options=\"myLabelToRemove1,myLabelToRemove2\"")
}

var cmdMessageRelabel = &cobra.Command{
	Use:   "relabel",
	Short: "Remove all labels and add new ones to a message: tatcli msg relabel <topic> <idMessage> --label=\"#EEEE;myLabel1,#EEEE;myLabel2\" --options=\"myLabelToRemove1,myLabelToRemove2\"",
	Long: `Remove all labels and add new ones to a message:
	tatcli message relabel <topic> <idMessage> --label="#EEEE;myLabel1,#EEEE;myLabel2"
	Example in bash:
	tatcli message relabel /MyTopic 01234567890 --label="#EEEE;myLabel1,#EEEE;myLabel2"

	Only remove some labels :

	tatcli message relabel /MyTopic 01234567890 --label="#EEEE;myLabel1,#EEEE;myLabel2" --options="myLabelToRemove1,myLabelToRemove2"

	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			labels := []tat.Label{}
			for _, label := range cmdLabel {
				s := strings.Split(label, ";")
				if len(s) == 2 {
					labels = append(labels, tat.Label{Text: s[1], Color: s[0]})
				} else {
					internal.Exit("Invalid argument label %s to 'relabel' a message: tatcli msg relabel --help\n", label)
				}
			}
			out, err := internal.Client().MessageRelabel(args[0], args[1], labels, cmdOptions)
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to 'relabel': tatcli message relabel --help\n")
		}
	},
}
