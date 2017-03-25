package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/gizak/termui"

	"github.com/ovh/cds/sdk"
)

func (ui *Termui) showStatus() {
	ui.current = StatusView
	termui.Body.Rows = nil

	p := termui.NewPar("")
	p.Height = 3
	p.TextFgColor = termui.ColorWhite
	p.BorderLabel = "Status"
	p.BorderFg = termui.ColorCyan
	ui.status = p

	ui.queue = NewScrollableList()
	ui.queue.ItemFgColor = termui.ColorWhite
	ui.queue.ItemBgColor = termui.ColorBlack

	ui.queue.BorderLabel = "Queue"
	ui.queue.Height = termui.TermHeight() - 16
	ui.queue.Width = termui.TermWidth()
	ui.queue.Items = []string{"Loading..."}

	ui.queueSelect = NewScrollableList()
	ui.queueSelect.ItemFgColor = termui.ColorWhite
	ui.queueSelect.ItemBgColor = termui.ColorBlack

	ui.queueSelect.BorderLabel = "Job selected"
	ui.queueSelect.Height = 10
	ui.queueSelect.Width = termui.TermWidth()
	ui.queueSelect.Items = []string{"[select a job](fg-cyan,bg-default)"}

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(6, 0, ui.header),
			termui.NewCol(6, 0, ui.msg),
		),
	)

	termui.Body.AddRows(
		termui.NewCol(12, 0, p),
	)

	termui.Body.AddRows(
		termui.NewCol(12, 0, ui.queue),
	)
	termui.Body.AddRows(
		termui.NewCol(12, 0, ui.queueSelect),
	)

	ui.distribQueue = make(map[string]int64)

	go ui.updateQueue()
	ui.updateVersion()
	termui.Clear()
	ui.draw(0)
}

func (ui *Termui) updateQueue() {
	for {
		time.Sleep(2 * time.Second)

		if ui.current != StatusView {
			return
		}

		pbJobs, err := sdk.GetBuildQueue()
		if err != nil {
			ui.msg.Text = err.Error()
			continue
		}

		if err != nil {
			sdk.Exit("Error: %s\n", err)
		}

		var maxQueued time.Duration
		booked := make(map[string]int)

		items := []string{
			fmt.Sprintf("[  %s %s%s %s➤ %s ➤ %s ➤ %s ➤ %s %s](fg-cyan,bg-default)", pad("since", 9), pad("booked", 40), pad("job", 13), pad("project", 7), pad("application", 15), pad("pipeline", 15), pad("enviromnent", 10), pad("branch", 8), "requirements"),
		}

		for i, job := range pbJobs {
			req := ""
			for _, r := range job.Job.Action.Requirements {
				req += fmt.Sprintf("%s(%s):%s ", r.Name, r.Type, r.Value)
			}
			prj := getVarsInPbj("cds.project", job.Parameters)
			app := getVarsInPbj("cds.application", job.Parameters)
			pip := getVarsInPbj("cds.pipeline", job.Parameters)
			env := getVarsInPbj("cds.environment", job.Parameters)
			bra := getVarsInPbj("git.branch", job.Parameters)
			duration := time.Since(job.Queued)
			if maxQueued < duration {
				maxQueued = duration
			}

			row := make([]string, 5)
			var c string
			if duration > 20*time.Second {
				c = "bg-red"
			} else if duration > 10*time.Second {
				c = "bg-yellow"
			} else {
				c = "bg-default"
			}
			row[0] = pad(fmt.Sprintf(sdk.Round(duration, time.Second).String()), 9)

			if job.BookedBy.ID != 0 {
				row[1] = pad(fmt.Sprintf(" BOOKED(%s.%d) ", job.BookedBy.Name, job.BookedBy.ID), 40)
				booked[fmt.Sprintf("%s.%d", job.BookedBy.Name, job.BookedBy.ID)] = booked[job.BookedBy.Name] + 1
			} else {
				row[1] = pad("", 40)
			}
			row[2] = pad(fmt.Sprintf("job:%d", job.ID), 13)
			row[3] = fmt.Sprintf("%s ➤ %s ➤ %s ➤ %s ➤ %s", pad(prj, 6), pad(app, 15), pad(pip, 15), pad(env, 10), pad(bra, 8))
			row[4] = fmt.Sprintf("%s", req)

			item := fmt.Sprintf("  [%s](%s)[%s %s %s %s](bg-default)", row[0], c, row[1], row[2], row[3], row[4])
			items = append(items, item)

			if i == ui.queue.Cursor-1 {
				var booked string
				if job.BookedBy.ID != 0 {
					booked = fmt.Sprintf(" booked by hatchery %s with id %d", job.BookedBy.Name, job.BookedBy.ID)
				}

				infos := []string{
					fmt.Sprintf("[job:%d%s](bg-default)", job.ID, booked),
					fmt.Sprintf("[project:%s application:%s pipeline:%s env:%s branch:%s](bg-default)", prj, app, pip, env, bra),
					fmt.Sprintf("[requirements:%s](bg-default)", req),
					fmt.Sprintf("[spawninfos:](bg-default)"),
				}
				for _, s := range job.SpawnInfos {
					infos = append(infos, fmt.Sprintf("[%s  %s](bg-default)", s.APITime, s.UserMessage))
				}
				ui.queueSelect.Items = infos
			}
		}
		ui.queue.Items = items

		t := fmt.Sprintf("queue:%d max:%s | ", len(pbJobs), sdk.Round(maxQueued, time.Second).String())
		for name, total := range booked {
			t += fmt.Sprintf("%s:%d ", name, total)
		}
		ui.infoQueue = t
		termui.Render()
	}
}

func pad(t string, size int) string {
	if len(t) > size {
		return t[0:size-3] + "..."
	}
	return t + strings.Repeat(" ", size-len(t))
}

func getVarsInPbj(key string, ps []sdk.Parameter) string {
	for _, p := range ps {
		if p.Name == key {
			return p.Value
		}
	}
	return ""
}

func (ui *Termui) updateVersion() {
	version, err := sdk.GetVersion()
	if err != nil {
		ui.status.Text = fmt.Sprintf("CDS is down (%s)", err)
	}
	ui.status.Text = fmt.Sprintf("CDS is up and running (%s)", version)

	go func() {
		for {
			time.Sleep(2 * time.Second)

			if ui.current != StatusView {
				return
			}

			if ui.queue == nil {
				continue
			}

			version, err := sdk.GetVersion()
			if err != nil {
				ui.status.Text = fmt.Sprintf("CDS is down (%s)", err)
				continue
			}
			ui.status.Text = fmt.Sprintf("CDS is up and running (%s) | %s", version, ui.infoQueue)
		}
	}()
}
