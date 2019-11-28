package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/termbox"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/text"

	"github.com/ovh/cds/cli"
)

var queueCmd = cli.Command{
	Name:  "queue",
	Short: "CDS Queue",
}

func queue() *cobra.Command {
	return cli.NewListCommand(queueCmd, queueRun, []*cobra.Command{
		cli.NewCommand(queueUICmd, queueUIRun, nil, withAllCommandModifiers()...),
	})
}

var queueUICmd = cli.Command{
	Name:  "interactive",
	Short: "Show the current queue",
}

func queueRun(v cli.Values) (cli.ListResult, error) {
	jobList, err := getJobQueue(sdk.StatusWaiting, sdk.StatusBuilding)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(jobList), nil
}

type jobCLI struct {
	Run          string        `cli:"run,key"`
	ProjectKey   string        `cli:"project_key"`
	WorkflowName string        `cli:"workflow_name"`
	NodeName     string        `cli:"pipeline_name"`
	Status       string        `cli:"status"`
	URL          string        `cli:"url"`
	Since        string        `cli:"since"`
	Duration     time.Duration `cli:"-"`
	BookedBy     string        `cli:"booked_by"`
	TriggeredBy  string        `cli:"triggered_by"`
}

func getJobQueue(status ...string) ([]jobCLI, error) {
	jobs, err := client.QueueWorkflowNodeJobRun(status...)
	if err != nil {
		return nil, err
	}

	config, err := client.ConfigUser()
	if err != nil {
		return nil, err
	}
	baseURL := config.URLUI
	jobsUI := make([]jobCLI, len(jobs))

	for k, jr := range jobs {
		jobsUI[k] = jobCLI{
			Run:          getVarsInPbj("cds.run", jr.Parameters),
			ProjectKey:   getVarsInPbj("cds.project", jr.Parameters),
			WorkflowName: getVarsInPbj("cds.workflow", jr.Parameters),
			NodeName:     getVarsInPbj("cds.node", jr.Parameters),
			Status:       jr.Status,
			URL:          generateQueueJobURL(baseURL, jr.Parameters),
			Since:        fmt.Sprintf(sdk.Round(time.Since(jr.Queued), time.Second).String()),
			Duration:     time.Since(jr.Queued),
			BookedBy:     jr.BookedBy.Name,
			TriggeredBy:  getVarsInPbj("cds.triggered_by.username", jr.Parameters),
		}
	}

	return jobsUI, nil
}

func getVarsInPbj(key string, ps []sdk.Parameter) string {
	for _, p := range ps {
		if p.Name == key {
			return p.Value
		}
	}
	return ""
}

func generateQueueJobURL(baseURL string, parameters []sdk.Parameter) string {
	prj := getVarsInPbj("cds.project", parameters)
	workflow := getVarsInPbj("cds.workflow", parameters)
	runNumber := getVarsInPbj("cds.run.number", parameters)
	return fmt.Sprintf("%s/project/%s/workflow/%s/run/%s", baseURL, prj, workflow, runNumber)
}

func queueUIRun(v cli.Values) error {
	t, err := termbox.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()

	ctx, cancel := context.WithCancel(context.Background())

	building, err := text.New(text.RollContent(), text.WrapAtWords())
	if err != nil {
		panic(err)
	}
	waiting, err := text.New(text.RollContent(), text.WrapAtWords())
	if err != nil {
		panic(err)
	}
	go writeLines(ctx, building, sdk.StatusBuilding)
	go writeLines(ctx, waiting, sdk.StatusWaiting)

	c, err := container.New(
		t,
		container.BorderTitle("PRESS Q TO QUIT"),
		container.SplitHorizontal(
			container.Top(
				container.Border(linestyle.Round),
				container.BorderTitle(sdk.StatusWaiting),
				container.PlaceWidget(waiting),
			),
			container.Bottom(
				container.Border(linestyle.Light),
				container.BorderTitle(sdk.StatusBuilding),
				container.PlaceWidget(building),
			),
		),
	)
	if err != nil {
		panic(err)
	}

	quit := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' {
			cancel()
		}
	}

	if err := termdash.Run(ctx, t, c, termdash.KeyboardSubscriber(quit)); err != nil {
		panic(err)
	}

	return nil
}

func writeLines(ctx context.Context, t *text.Text, status string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			jobList, err := getJobQueue(status)
			if err != nil {
				if err := t.Write(fmt.Sprintf("error: %s\n", err)); err != nil {
					panic(err)
				}
			}
			t.Reset()
			for _, j := range jobList {
				var wo text.WriteOption
				if status == sdk.StatusWaiting {
					if j.Duration > 120*time.Second {
						wo = text.WriteCellOpts(cell.FgColor(cell.ColorRed))
					} else if j.Duration > 60*time.Second {
						wo = text.WriteCellOpts(cell.FgColor(cell.ColorYellow))
					}
				}
				if wo != nil {
					err = t.Write(generateQueueJobLine(j), wo)
				} else {
					err = t.Write(generateQueueJobLine(j))
				}
				if err != nil {
					panic(err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func generateQueueJobLine(job jobCLI) string {
	row := make([]string, 4)
	row[0] = pad(job.Since, 8)
	row[1] = pad(job.Run, 6)
	row[2] = fmt.Sprintf("%s ➤ %s", pad(job.ProjectKey+"/"+job.WorkflowName, 30), pad(job.NodeName, 20))
	row[3] = fmt.Sprintf("➤ %s", pad(job.TriggeredBy, 17))
	return fmt.Sprintf("%s %s %s %s\n", row[0], row[1], row[2], row[3])
}

func pad(t string, size int) string {
	if len(t) > size {
		return t[0:size-3] + "..."
	}
	return t + strings.Repeat(" ", size-len(t))
}
