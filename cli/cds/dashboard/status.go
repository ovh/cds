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

	ui.queue = NewScrollableList()
	ui.queue.ItemFgColor = termui.ColorWhite
	ui.queue.ItemBgColor = termui.ColorBlack

	ui.queue.BorderLabel = "Queue"
	ui.queue.Height = termui.TermHeight() - 11
	ui.queue.Width = termui.TermWidth()
	ui.queue.Items = []string{"Loading..."}
	ui.queue.BorderBottom = false
	ui.queue.BorderLeft = false
	ui.queue.BorderRight = false

	ui.queueSelect = NewScrollableList()
	ui.queueSelect.ItemFgColor = termui.ColorWhite
	ui.queueSelect.ItemBgColor = termui.ColorBlack

	ui.queueSelect.BorderLabel = "Job selected"
	ui.queueSelect.Height = 11
	ui.queueSelect.Width = termui.TermWidth()
	ui.queueSelect.Items = []string{"[select a job](fg-cyan,bg-default)"}
	ui.queueSelect.BorderBottom = false
	ui.queueSelect.BorderLeft = false
	ui.queueSelect.BorderRight = false

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, ui.header),
		),
	)

	termui.Body.AddRows(
		termui.NewCol(12, 0, ui.queue),
	)
	termui.Body.AddRows(
		termui.NewCol(12, 0, ui.queueSelect),
	)

	ui.distribQueue = make(map[string]int64)

	baseURL := "http://cds.ui/"
	urlUI, err := sdk.GetConfigUser()
	if err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
	}

	if b, ok := urlUI[sdk.ConfigURLUIKey]; ok {
		baseURL = b
	}

	go ui.updateQueue(baseURL)
	termui.Clear()
	ui.draw(0)
}

func (ui *Termui) updateQueue(baseURL string) {
	for {
		time.Sleep(2 * time.Second)

		if ui.current != StatusView {
			return
		}

		start := time.Now()
		pbJobs, err := sdk.GetBuildQueue()
		if err != nil {
			ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
			continue
		}
		elapsed := time.Since(start)
		ui.msg = fmt.Sprintf("[getQueue in %s](fg-cyan)", sdk.Round(elapsed, time.Millisecond).String())

		if err != nil {
			sdk.Exit("Error: %s\n", err)
		}

		var maxQueued time.Duration
		booked := make(map[string]int)

		items := []string{
			fmt.Sprintf("[  %s %s%s %s ➤ %s ➤ %s](fg-cyan,bg-default)", pad("since", 9), pad("booked", 27), pad("job", 7), pad("project/application", 35), pad("pipeline/branch/env", 33), "requirements"),
		}

		for i, job := range pbJobs {
			req := ""
			for _, r := range job.Job.Action.Requirements {
				req += fmt.Sprintf("%s(%s):%s ", r.Name, r.Type, r.Value)
			}
			prj := getVarsInPbj("cds.project", job.Parameters)
			app := getVarsInPbj("cds.application", job.Parameters)
			pip := getVarsInPbj("cds.pipeline", job.Parameters)
			build := getVarsInPbj("cds.buildNumber", job.Parameters)
			env := getVarsInPbj("cds.environment", job.Parameters)
			bra := getVarsInPbj("git.branch", job.Parameters)
			duration := time.Since(job.Queued)
			if maxQueued < duration {
				maxQueued = duration
			}

			row := make([]string, 5)
			var c string
			if duration > 60*time.Second {
				c = "bg-red"
			} else if duration > 15*time.Second {
				c = "bg-yellow"
			} else {
				c = "bg-default"
			}
			row[0] = pad(fmt.Sprintf(sdk.Round(duration, time.Second).String()), 9)

			if job.BookedBy.ID != 0 {
				row[1] = pad(fmt.Sprintf(" %s.%d ", job.BookedBy.Name, job.BookedBy.ID), 27)
				booked[fmt.Sprintf("%s.%d", job.BookedBy.Name, job.BookedBy.ID)] = booked[job.BookedBy.Name] + 1
			} else {
				row[1] = pad("", 27)
			}
			row[2] = pad(fmt.Sprintf("%d", job.ID), 7)
			row[3] = fmt.Sprintf("%s ➤ %s", pad(prj+"/"+app, 35), pad(pip+"/"+bra+"/"+env, 33))
			row[4] = fmt.Sprintf("➤ %s", req)

			item := fmt.Sprintf("  [%s](%s)[%s %s %s %s](bg-default)", row[0], c, row[1], row[2], row[3], row[4])
			items = append(items, item)

			if i == ui.queue.Cursor-1 {
				var booked string
				if job.BookedBy.ID != 0 {
					booked = fmt.Sprintf(" booked by hatchery %s with id %d", job.BookedBy.Name, job.BookedBy.ID)
				}
				u := computeURL(baseURL, prj, app, pip, build, env)
				infos := []string{
					fmt.Sprintf("[job:%d%s](bg-default)", job.ID, booked),
					fmt.Sprintf("[project:%s application:%s pipeline:%s env:%s branch:%s](bg-default)", prj, app, pip, env, bra),
					fmt.Sprintf("[requirements:%s](bg-default)", req),
					fmt.Sprintf("[%s](bg-default)", u),
					fmt.Sprintf("[spawninfos:](bg-default)"),
				}
				for _, s := range job.SpawnInfos {
					infos = append(infos, fmt.Sprintf("[%s  %s](bg-default)", s.APITime, s.UserMessage))
				}
				ui.queueSelect.Items = infos
				ui.queueCurrentJobURL = u
			}
		}
		ui.queue.Items = items

		t := fmt.Sprintf(" queue:%d max:%s ", len(pbJobs), sdk.Round(maxQueued, time.Second).String())
		for name, total := range booked {
			t += fmt.Sprintf("%s:%d ", name, total)
		}
		ui.queue.BorderLabel = t
		termui.Render()
	}
}

func computeURL(baseURL, prj, app, pip, build, env string) string {
	return fmt.Sprintf("%s/#/project/%s/application/%s/pipeline/%s/build/%s?env=%s",
		baseURL, prj, app, pip, build, env,
	)
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
