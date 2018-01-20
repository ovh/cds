package stats

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
	"gopkg.in/cheggaaa/pb.v1"

	"github.com/ovh/tat"
)

var minCount int
var withDedicated bool

func init() {
	cmdStatsDistribution.Flags().IntVar(&minCount, "minCount", -1, "Display topic with min count")
	cmdStatsDistribution.Flags().BoolVar(&withDedicated, "withDedicated", true, "Hide dedicated topic")
}

var cmdStatsDistribution = &cobra.Command{
	Use:   "distributiontopics",
	Short: "Distribution of messages per topics: tatcli stats distributiontopics",
	Run: func(cmd *cobra.Command, args []string) {
		b, errs := internal.Client().StatsDistributionTopics(0, 1)
		internal.Check(errs)

		topics := []tat.TopicDistributionJSON{}
		skip := 0
		limit := 10
		bar := pb.StartNew(b.Total)
		for skip < b.Total {
			out, err := internal.Client().StatsDistributionTopics(skip, limit)
			internal.Check(err)
			topics = append(topics, out.Topics...)
			skip += 10
			bar.Set(skip)
		}

		sort.Sort(byCount(topics))

		data := [][]string{}
		for _, t := range topics {
			if minCount > 0 && t.Count < minCount {
				continue
			}
			if !withDedicated && t.Dedicated {
				continue
			}
			data = append(data, []string{t.Topic, strconv.Itoa(t.Count), fmt.Sprintf("%t", t.Dedicated)})
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Topic", "Count", "Dedicated"})

		for _, v := range data {
			table.Append(v)
		}

		bar.Finish()

		table.Render()

	},
}

type byCount []tat.TopicDistributionJSON

func (a byCount) Len() int           { return len(a) }
func (a byCount) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byCount) Less(i, j int) bool { return a[i].Count < a[j].Count }
