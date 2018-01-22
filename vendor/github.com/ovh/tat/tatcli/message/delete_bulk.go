package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

func init() {
	cmdMessageDeleteBulk.Flags().BoolVarP(&cascade, "cascade", "", false, "--cascade : delete messages and replies")
	cmdMessageDeleteBulk.Flags().BoolVarP(&cascadeForce, "cascadeForce", "", false, "--cascadeForce : delete messages and replies, event if it's in a Tasks Topic of one user")

	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.IDMessage, "idMessage", "", "", "Search by IDMessage")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.InReplyOfID, "inReplyOfID", "", "", "Search by IDMessage InReply")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.InReplyOfIDRoot, "inReplyOfIDRoot", "", "", "Search by IDMessage IdRoot")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.AllIDMessage, "allIDMessage", "", "", "Search in All ID Message (idMessage, idReply, idRoot)")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.Text, "text", "", "", "Search by text")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.Topic, "topic", "", "", "Search by topic")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.Label, "label", "", "", "Search by label: could be labelA,labelB")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.NotLabel, "notLabel", "", "", "Search by label (exclude): could be labelA,labelB")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.AndLabel, "andLabel", "", "", "Search by label (and) : could be labelA,labelB")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.Tag, "tag", "", "", "Search by tag : could be tagA,tagB")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.NotTag, "notTag", "", "", "Search by tag (exclude) : could be tagA,tagB")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.AndTag, "andTag", "", "", "Search by tag (and) : could be tagA,tagB")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateMinCreation, "dateMinCreation", "", "", "Search by dateCreation (timestamp), select messages where dateCreation >= dateMinCreation")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateMaxCreation, "dateMaxCreation", "", "", "Search by dateCreation (timestamp), select messages where dateCreation <= dateMaxCreation")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateMinUpdate, "dateMinUpdate", "", "", "Search by dateUpdate (timestamp), select messages where dateUpdate >= dateMinUpdate")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateMaxUpdate, "dateMaxUpdate", "", "", "Search by dateUpdate (timestamp), select messages where dateUpdate <= dateMaxUpdate")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateRefCreation, "dateRefCreation", "", "", "This have to be used with dateRefDeltaMinCreation and / or dateRefDeltaMaxCreation. This could be BeginningOfMinute, BeginningOfHour, BeginningOfDay, BeginningOfWeek, BeginningOfMonth, BeginningOfQuarter, BeginningOfYear")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateRefDeltaMinCreation, "dateRefDeltaMinCreation", "", "", "Add seconds to dateRefCreation flag")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateRefDeltaMaxCreation, "dateRefDeltaMaxCreation", "", "", "Add seconds to dateRefCreation flag")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateRefUpdate, "dateRefUpdate", "", "", "This have to be used with dateRefDeltaMinUpdate and / or dateRefDeltaMaxUpdate. This could be BeginningOfMinute, BeginningOfHour, BeginningOfDay, BeginningOfWeek, BeginningOfMonth, BeginningOfQuarter, BeginningOfYear")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateRefDeltaMinUpdate, "dateRefDeltaMinUpdate", "", "", "Add seconds to dateRefUpdate flag")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.DateRefDeltaMaxUpdate, "dateRefDeltaMaxUpdate", "", "", "Add seconds to dateRefUpdate flag")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LastMinCreation, "lastMinCreation", "", "", "Search by dateCreation (duration in second), select messages where dateCreation >= now - lastMinCreation")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LastMaxCreation, "lastMaxCreation", "", "", "Search by dateCreation (duration in second), select messages where dateCreation <= now - lastMaxCreation")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LastMinUpdate, "lastMinUpdate", "", "", "Search by dateUpdate (duration in second), select messages where dateUpdate >= now - lastMinCreation")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LastMaxUpdate, "lastMaxUpdate", "", "", "Search by dateUpdate (duration in second), select messages where dateUpdate <= now - lastMaxCreation")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LastHourMinCreation, "lastHourMinCreation", "", "", "Search by dateCreation, select messages where dateCreation >= Now Beginning Of Hour - (60 * lastHourMinCreation)")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LastHourMaxCreation, "lastHourMaxCreation", "", "", "Search by dateCreation, select messages where dateCreation <= Now Beginning Of Hour - (60 * lastHourMaxCreation)")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LastHourMinUpdate, "lastHourMinUpdate", "", "", "Search by dateUpdate, select messages where dateUpdate >= Now Beginning Of Hour - (60 * lastHourMinCreation)")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LastHourMaxUpdate, "lastHourMaxUpdate", "", "", "Search by dateUpdate, select messages where dateUpdate <= Now Beginning Of Hour - (60 * lastHourMaxCreation)")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.Username, "username", "", "", "Search by username : could be usernameA,usernameB")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LimitMinNbReplies, "limitMinNbReplies", "", "", "In onetree mode, filter root messages with more or equals minNbReplies")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LimitMaxNbReplies, "limitMaxNbReplies", "", "", "In onetree mode, filter root messages with min or equals maxNbReplies")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LimitMinNbVotesUP, "limitMinNbVotesUP", "", "", "Search by nbVotesUP")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LimitMaxNbVotesUP, "limitMaxNbVotesUP", "", "", "Search by nbVotesUP")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LimitMinNbVotesDown, "limitMinNbVotesDown", "", "", "Search by nbVotesDown")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.LimitMaxNbVotesDown, "limitMaxNbVotesDown", "", "", "Search by nbVotesDown")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.OnlyMsgRoot, "onlyMsgRoot", "", "", "--onlyMsgRoot=true: restricts to root message only (inReplyOfIDRoot empty). If treeView is used, limit search criteria to root message, replies are still given, independently of search criteria.")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.OnlyCount, "onlyCount", "", "", "--onlyCount=true: only count messages, without retrieve msg. limit, skip, treeview criterias are ignored.")
	cmdMessageDeleteBulk.Flags().StringVarP(&criteria.TreeView, "treeView", "", "", "Tree View of messages: onetree or fulltree. Default: onetree")
}

var cmdMessageDeleteBulk = &cobra.Command{
	Use:   "deletebulk",
	Short: "Delete a list of messages: tatcli message deletebulk <topic> <skip> <limit> [--cascade] [--cascadeForce]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			criteria.Skip, criteria.Limit = internal.GetSkipLimit(args)
			out, err := internal.Client().MessagesDeleteBulk(args[0], cascade, cascadeForce, criteria)
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to delete message: tatcli message delete --help\n")
		}
	},
}
