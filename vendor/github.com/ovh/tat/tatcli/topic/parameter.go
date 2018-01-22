package topic

import (
	"strconv"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdTopicParameter.Flags().BoolVarP(&recursive, "recursive", "r", false, "Update param topic recursively")
}

var cmdTopicParameter = &cobra.Command{
	Use:     "parameter",
	Short:   "Update param on one topic: tatcli topic param [--recursive] <topic> <maxLength> <maxReplies> <canForceDate> <canUpdateMsg> <canDeleteMsg> <canUpdateAllMsg> <canDeleteAllMsg> <adminCanUpdateAllMsg> <adminCanDeleteAllMsg> <isAutoComputeTags> <isAutoComputeLabels>",
	Aliases: []string{"param"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 11 {
			internal.Exit("Invalid parameter to tatcli topic param. See tatcli topic param --help\n")
		}

		var err error

		p := tat.TopicParameters{
			Topic: args[0],
		}

		p.MaxLength, err = strconv.Atoi(args[1])
		internal.Check(err)
		p.MaxReplies, err = strconv.Atoi(args[2])
		internal.Check(err)
		p.CanForceDate, err = strconv.ParseBool(args[3])
		internal.Check(err)
		p.CanUpdateMsg, err = strconv.ParseBool(args[4])
		internal.Check(err)
		p.CanDeleteMsg, err = strconv.ParseBool(args[5])
		internal.Check(err)
		p.CanUpdateAllMsg, err = strconv.ParseBool(args[6])
		internal.Check(err)
		p.CanDeleteAllMsg, err = strconv.ParseBool(args[7])
		internal.Check(err)
		p.AdminCanUpdateAllMsg, err = strconv.ParseBool(args[8])
		internal.Check(err)
		p.AdminCanDeleteAllMsg, err = strconv.ParseBool(args[9])
		internal.Check(err)
		p.IsAutoComputeTags, err = strconv.ParseBool(args[10])
		internal.Check(err)
		p.IsAutoComputeLabels, err = strconv.ParseBool(args[11])
		internal.Check(err)
		out, err := internal.Client().TopicParameter(p)
		internal.Check(err)
		if internal.Verbose {
			internal.Print(out)
		}
	},
}
