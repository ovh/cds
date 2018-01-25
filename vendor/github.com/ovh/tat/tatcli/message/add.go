package message

import (
	"fmt"
	"strings"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cmdLabel []string

var (
	dateCreation int
)

func init() {
	cmdMessageAdd.Flags().IntVarP(&dateCreation, "dateCreation", "", -1, "Force date creation, only for system user")
	cmdMessageAdd.Flags().StringSliceVar(&cmdLabel, "label", nil, "add labels : --label=\"#EEEE;myLabel1,#EEEE;myLabel2\"")
}

var cmdMessageAdd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"a"},
	Short:   "tatcli message add [--dateCreation=timestamp] <topic> <my message>",
	Long: `Add a message to a Topic:
		tatcli message add /Private/firstname.lastname my new messsage
		`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 2 {
			topic := args[0]
			message := strings.Join(args[1:], " ")
			msg, err := Create(topic, message)
			internal.Check(err)
			if internal.Verbose {
				internal.Print(msg)
			}
		} else {
			internal.Exit("Invalid argument to add a message: tatcli msg add --help\n")
		}
	},
}

// Create creates a message in specified topic
func Create(topic, message string) (*tat.MessageJSONOut, error) {
	m := tat.MessageJSON{Text: message, Topic: topic}
	if dateCreation > 0 {
		m.DateCreation = float64(dateCreation)
	}
	for _, label := range cmdLabel {
		s := strings.Split(label, ";")
		if len(s) == 2 {
			m.Labels = append(m.Labels, tat.Label{Text: s[1], Color: s[0]})
		} else {
			return nil, fmt.Errorf("Invalid argument label %s to add a message: tatcli msg add --help\n", label)
		}
	}

	return internal.Client().MessageAdd(m)
}
