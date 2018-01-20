package topic

import (
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var (
	criteriaTopic            string
	criteriaTopicPath        string
	criteriaIDTopic          string
	criteriaDescription      string
	criteriaDateMinCreation  string
	criteriaDateMaxCreation  string
	criteriaGetNbMsgUnread   string
	criteriaGetOnlyFavorites string
	criteriaGetForTatAdmin   string
)

var (
	criteria tat.TopicCriteria
)

func init() {
	cmdTopicList.Flags().StringVarP(&criteria.Topic, "topic", "", "", "Search by Topic name, example: /topicA")
	cmdTopicList.Flags().StringVarP(&criteria.TopicPath, "topicPath", "", "", "Search by Topic Path, example: /topicA will return /topicA/subA, /topicA/subB")
	cmdTopicList.Flags().StringVarP(&criteria.IDTopic, "idTopic", "", "", "Search by id of topic")
	cmdTopicList.Flags().StringVarP(&criteria.Description, "description", "", "", "Search by description of topic")
	cmdTopicList.Flags().StringVarP(&criteria.DateMinCreation, "dateMinCreation", "", "", "Filter result on dateCreation, timestamp Unix format")
	cmdTopicList.Flags().StringVarP(&criteria.DateMaxCreation, "dateMaxCreation", "", "", "Filter result on dateCreation, timestamp Unix Format")
	cmdTopicList.Flags().StringVarP(&criteria.GetNbMsgUnread, "getNbMsgUnread", "", "", "If true, add new array to return, topicsMsgUnread with topic:nbUnreadMsgSinceLastPresenceOnTopic")
	cmdTopicList.Flags().StringVarP(&criteria.OnlyFavorites, "getOnlyFavorites", "", "", "If true, returns only favorites topics, except /Private/* (all /Private/* are returned)")
	cmdTopicList.Flags().StringVarP(&criteria.GetForTatAdmin, "getForTatAdmin", "", "", "(AdminOnly) If true, and requester is a Tat Admin, returns all topics (except /Private/*) without checking user / group access (RO or RW on Topic)")
}

var cmdTopicList = &cobra.Command{
	Use:     "list",
	Short:   "List all topics: tatcli topic list [<skip>] [<limit>], tatcli topic list -h for see all criterias",
	Aliases: []string{"l"},
	Run: func(cmd *cobra.Command, args []string) {
		criteria.Skip, criteria.Limit = internal.GetSkipLimit(args)
		c := internal.Client()
		out, err := c.TopicList(&criteria)
		internal.Check(err)
		internal.Print(out)
	},
}
