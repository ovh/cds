package main

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gizak/termui"
	"github.com/skratchdot/open-golang/open"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var monitoringCmd = cli.Command{
	Name:    "monitoring",
	Short:   "CDS monitoring",
	Aliases: []string{"ui"},
}

func monitoringRun(v cli.Values) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("cds UI crashed :(\n%s\n", r)
			termui.Close()
		}
	}()

	ui := &Termui{}
	ui.init()
	ui.draw(0)

	defer termui.Close()
	termui.Loop()
	return nil, nil
}

// Termui wrapper designed for dashboard creation
type Termui struct {
	header, times *termui.Par
	msg           string

	current  string
	selected string

	// monitoring
	queue                   *cli.ScrollableList
	statusHatcheriesWorkers *cli.ScrollableList
	status                  *cli.ScrollableList
	currentURL              string
}

// Constants for each view of cds ui
const (
	QueueSelected             = "queue"
	BuildingSelected          = "building"
	HatcheriesWorkersSelected = "hatcheriesWorkers"
	StatusSelected            = "status"
)

func (ui *Termui) init() {
	if err := termui.Init(); err != nil {
		panic(err)
	}

	termui.Handle("/timer/1s", func(e termui.Event) {
		t := e.Data.(termui.EvtTimer)
		ui.draw(int(t.Count))
	})

	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
	})

	termui.Handle("/sys/kbd", func(e termui.Event) {
		ui.msg = fmt.Sprintf("No command for %v", e)
	})

	termui.Handle("/sys/kbd/<tab>", func(e termui.Event) {
		ui.monitoringSelectNext()
	})

	termui.Handle("/sys/kbd/<down>", func(e termui.Event) {
		ui.monitoringCursorDown()
	})
	termui.Handle("/sys/kbd/<up>", func(e termui.Event) {
		ui.monitoringCursorUp()
	})

	termui.Handle("/sys/kbd/<enter>", func(e termui.Event) {
		if ui.currentURL != "" {
			open.Run(ui.currentURL)
		}
	})

	ui.initHeader()
	ui.initTimes()
	go ui.showMonitoring()
}

func (ui *Termui) draw(i int) {
	checking, checkingColor := statusShort(sdk.StatusChecking.String())
	waiting, waitingColor := statusShort(sdk.StatusWaiting.String())
	building, buildingColor := statusShort(sdk.StatusBuilding.String())
	success, successColor := statusShort(sdk.StatusSuccess.String())
	fail, failColor := statusShort(sdk.StatusFail.String())
	disabled, disabledColor := statusShort(sdk.StatusDisabled.String())
	ui.header.Text = fmt.Sprintf("[CDS | (q)uit | Legend: ](fg-cyan) [Checking:%s](%s)  [Waiting:%s](%s)  [Building:%s](%s)  [Success:%s](%s)  [Fail:%s](%s)  [Disabled:%s](%s)",
		checking, checkingColor,
		waiting, waitingColor,
		building, buildingColor,
		success, successColor,
		fail, failColor,
		disabled, disabledColor)
	ui.times.Text = fmt.Sprintf(ui.msg)
	termui.Body.Align()
	termui.Render(termui.Body)
}

func (ui *Termui) initHeader() {
	p := termui.NewPar("")
	p.Height = 1
	p.TextFgColor = termui.ColorWhite
	p.BorderLabel = ""
	p.BorderFg = termui.ColorCyan
	p.Border = false
	ui.header = p
}

func (ui *Termui) initTimes() {
	p := termui.NewPar("")
	p.Height = 1
	p.TextFgColor = termui.ColorWhite
	p.BorderLabel = ""
	p.BorderFg = termui.ColorCyan
	p.Border = false
	ui.times = p
}

////////////

func (ui *Termui) showMonitoring() {
	termui.Body.Rows = nil

	ui.queue = cli.NewScrollableList()
	ui.queue.ItemFgColor = termui.ColorWhite
	ui.queue.ItemBgColor = termui.ColorBlack

	heightBottom := 25
	heightQueue := (termui.TermHeight() - heightBottom)
	if heightQueue <= 0 {
		heightQueue = 4
	}
	ui.queue.BorderLabel = " Queue "
	ui.queue.Height = heightQueue
	ui.queue.Width = termui.TermWidth()
	ui.queue.Items = []string{"Loading..."}
	ui.queue.BorderBottom = false
	ui.queue.BorderLeft = false
	ui.queue.BorderRight = false

	ui.selected = QueueSelected

	ui.statusHatcheriesWorkers = cli.NewScrollableList()
	ui.statusHatcheriesWorkers.BorderLabel = " Hatcheries "
	ui.statusHatcheriesWorkers.Height = heightBottom
	ui.statusHatcheriesWorkers.Items = []string{"[loading...](fg-cyan,bg-default)"}
	ui.statusHatcheriesWorkers.BorderBottom = false
	ui.statusHatcheriesWorkers.BorderLeft = true
	ui.statusHatcheriesWorkers.BorderRight = false

	ui.status = cli.NewScrollableList()
	ui.status.BorderLabel = " Status "
	ui.status.Height = heightBottom
	ui.status.Items = []string{"[loading...](fg-cyan,bg-default)"}
	ui.status.BorderBottom = false
	ui.status.BorderLeft = false
	ui.status.BorderRight = false

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, ui.header),
		),
		termui.NewRow(
			termui.NewCol(12, 0, ui.times),
		),
	)

	termui.Body.AddRows(
		termui.NewCol(12, 0, ui.queue),
	)
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(7, 0, ui.status),
			termui.NewCol(5, 0, ui.statusHatcheriesWorkers),
		),
	)

	termui.Render()

	baseURL := "http://cds.ui/"
	urlUI, err := client.ConfigUser()
	if err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
	}

	if b, ok := urlUI[sdk.ConfigURLUIKey]; ok {
		baseURL = b
	}

	ticker := time.NewTicker(2 * time.Second).C

	for {
		var a, b, c string
		select {
		case <-ticker:
			ui.monitoringColorSelected()
			a = ui.updateQueue(baseURL)
			b = ui.updateQueueWorkers()
			c = ui.updateStatus()
		}
		ui.msg = fmt.Sprintf("%s | %s | %s", a, b, c)
		termui.Render()
	}
}

func (ui *Termui) monitoringCursorDown() {
	switch ui.selected {
	case QueueSelected:
		ui.queue.CursorDown()
	case HatcheriesWorkersSelected:
		ui.statusHatcheriesWorkers.CursorDown()
	case StatusSelected:
		ui.status.CursorDown()
	}
}

func (ui *Termui) monitoringCursorUp() {
	switch ui.selected {
	case QueueSelected:
		ui.queue.CursorUp()
	case HatcheriesWorkersSelected:
		ui.statusHatcheriesWorkers.CursorUp()
	case StatusSelected:
		ui.status.CursorUp()
	}
}

func (ui *Termui) monitoringSelectNext() {
	ui.currentURL = ""
	switch ui.selected {
	case QueueSelected:
		ui.selected = BuildingSelected
		ui.queue.Cursor = 0
	case HatcheriesWorkersSelected:
		ui.selected = StatusSelected
		ui.statusHatcheriesWorkers.Cursor = 0
	case StatusSelected:
		ui.selected = QueueSelected
		ui.status.Cursor = 0
	}
	ui.monitoringColorSelected()
}

func (ui *Termui) monitoringColorSelected() {
	ui.queue.BorderFg = termui.ColorDefault
	ui.statusHatcheriesWorkers.BorderFg = termui.ColorDefault
	ui.status.BorderFg = termui.ColorDefault

	switch ui.selected {
	case QueueSelected:
		ui.queue.BorderFg = termui.ColorRed
	case HatcheriesWorkersSelected:
		ui.statusHatcheriesWorkers.BorderFg = termui.ColorRed
	case StatusSelected:
		ui.status.BorderFg = termui.ColorRed
	}
	termui.Render(ui.queue,
		ui.statusHatcheriesWorkers,
		ui.status)
}

func (ui *Termui) updateStatus() string {
	start := time.Now()
	statusEngine, err := client.MonStatus()
	if err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
		return ""
	}
	elapsed := time.Since(start)
	msg := fmt.Sprintf("[status %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())

	items := []string{}
	for _, l := range statusEngine.Lines {
		if l.Status == sdk.MonitoringStatusWarn {
			items = append(items, fmt.Sprintf("[%s](fg-yellow,bg-default)", l.String()))
		} else if l.Status != sdk.MonitoringStatusOK {
			items = append(items, fmt.Sprintf("[%s](fg-white,bg-red)", l.String()))
		} else if strings.Contains(l.Component, "Global") {
			items = append(items, fmt.Sprintf("[%s](fg-white,bg-default)", l.String()))
		}
	}
	ui.status.Items = items
	return msg
}

func (ui *Termui) pipelineLine(projKey string, app sdk.Application, pb sdk.PipelineBuild) string {
	branch := pb.Trigger.VCSChangesBranch
	selected := ",bg-default"
	if ui.selected == BuildingSelected {
		selected = "fg-white"
	}
	icon, color := statusShort(pb.Status.String())
	return fmt.Sprintf("[%s](%s,%s)[ %s](bg-default)[➤ ](fg-cyan,bg-default)[%s ](bg-default)[➤ ](fg-cyan,bg-default)[%s](bg-default)", icon, color, selected, pad(projKey+"/"+app.Name, 35), pad(pb.Pipeline.Name, 25), pad(branch+"/"+pb.Environment.Name, 19))
}

func (ui *Termui) updateQueueWorkers() string {
	start := time.Now()
	workers, err := client.WorkerList()
	if err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
		return ""
	}
	elapsed := time.Since(start)
	msg := fmt.Sprintf("[workers %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())

	ui.computeStatusHatcheriesWorkers(workers)

	msga := ui.computeStatusWorkerModels(workers)
	return msg + msga
}

func (ui *Termui) computeStatusHatcheriesWorkers(workers []sdk.Worker) {
	hatcheryNames, statusTitle := []string{}, []string{}
	hatcheries := make(map[string]map[string]int64)
	status := make(map[string]int)

	for _, w := range workers {
		var name string
		if w.HatcheryID == 0 {
			name = "Without hatchery"
		} else {
			name = w.HatcheryName
		}

		if _, ok := hatcheries[name]; !ok {
			hatcheries[name] = make(map[string]int64)
			hatcheryNames = append(hatcheryNames, name)
		}
		hatcheries[name][w.Status.String()] = hatcheries[name][w.Status.String()] + 1

		if _, ok := status[w.Status.String()]; !ok {
			statusTitle = append(statusTitle, w.Status.String())
		}
		status[w.Status.String()] = status[w.Status.String()] + 1
	}

	selected := ",bg-default"
	if ui.selected == HatcheriesWorkersSelected {
		selected = ""
	}

	items := []string{}
	sort.Strings(hatcheryNames)
	for _, name := range hatcheryNames {
		v := hatcheries[name]
		var t string
		for _, status := range statusTitle {
			if v[status] > 0 {
				icon, color := statusShort(status)
				t += fmt.Sprintf("[ %d %s ](%s%s)", v[status], icon, color, selected)
			}
		}
		t += fmt.Sprintf("[ %s](fg-white%s)", name, selected)
		items = append(items, t)
	}
	ui.statusHatcheriesWorkers.Items = items

	sort.Strings(statusTitle)
	title := " Hatcheries "
	for _, s := range statusTitle {
		icon, color := statusShort(s)
		title += fmt.Sprintf("[%d %s](%s) ", status[s], icon, color)
	}
	ui.statusHatcheriesWorkers.BorderLabel = title
}

func (ui *Termui) computeStatusWorkerModels(workers []sdk.Worker) string {
	start := time.Now()
	if _, err := client.WorkerModels(); err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
		return ""
	}
	elapsed := time.Since(start)
	return fmt.Sprintf(" | [wModels %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())
}

func (ui *Termui) updateQueue(baseURL string) string {
	start := time.Now()
	wJobs, errw := client.QueueWorkflowNodeJobRun()
	if errw != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", errw.Error())
		return ""
	}
	elapsed := time.Since(start)
	msg := fmt.Sprintf("[queue wf %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())

	start = time.Now()
	pbJobs, errpb := client.QueuePipelineBuildJob()
	if errpb != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", errpb.Error())
		return ""
	}
	elapsed = time.Since(start)
	msg += fmt.Sprintf(" | [queue pb %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())

	var maxQueued time.Duration

	items := []string{
		fmt.Sprintf("[  %s %s%s %s ➤ %s ➤ %s ➤ %s](fg-cyan,bg-default)", pad("since", 9), pad("booked", 27), pad("run", 7), pad("project/workflow", 30), pad("node", 20), pad("triggered by", 17), "requirements"),
	}

	var idx int
	var item string
	for _, job := range pbJobs {
		item, maxQueued = ui.updateQueueJob(idx, maxQueued, job.ID, false, job.Parameters, job.Job.Action.Requirements, job.Queued, job.BookedBy, baseURL)
		items = append(items, item)
		idx++
	}
	start = time.Now()
	nWJobs, errw := client.QueueCountWorkflowNodeJobRun(nil, nil)
	elapsed = time.Since(start)
	if errw != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", errw.Error())
		return ""
	}
	msg = fmt.Sprintf("[count queue wf %s](fg-cyan,bg-default) | %s", sdk.Round(elapsed, time.Millisecond).String(), msg)

	for _, job := range wJobs {
		item, maxQueued = ui.updateQueueJob(idx, maxQueued, job.ID, true, job.Parameters, job.Job.Action.Requirements, job.Queued, job.BookedBy, baseURL)
		items = append(items, item)
		idx++
	}
	ui.queue.Items = items

	t := fmt.Sprintf("Queue:%d - Max Waiting:%s ", nWJobs.Count+int64(len(pbJobs)), sdk.Round(maxQueued, time.Second).String())
	ui.queue.BorderLabel = t
	return msg
}

func (ui *Termui) updateQueueJob(idx int, maxQueued time.Duration, id int64, isWJob bool, parameters []sdk.Parameter, requirements []sdk.Requirement, queued time.Time, bookedBy sdk.Hatchery, baseURL string) (string, time.Duration) {
	req := ""
	for _, r := range requirements {
		req += fmt.Sprintf("%s:%s ", r.Type, r.Value)
	}
	prj := getVarsInPbj("cds.project", parameters)
	app := getVarsInPbj("cds.application", parameters)
	pip := getVarsInPbj("cds.pipeline", parameters)
	workflow := getVarsInPbj("cds.workflow", parameters)
	node := getVarsInPbj("cds.node", parameters)
	run := getVarsInPbj("cds.run", parameters)
	runNumber := getVarsInPbj("cds.run.number", parameters)
	build := getVarsInPbj("cds.buildNumber", parameters)
	env := getVarsInPbj("cds.environment", parameters)
	bra := getVarsInPbj("git.branch", parameters)
	version := getVarsInPbj("cds.version", parameters)
	triggeredBy := getVarsInPbj("cds.triggered_by.username", parameters)
	duration := time.Since(queued)
	var currentURL, fgColor string

	row := make([]string, 6)
	var c string
	if duration > 60*time.Second {
		c = "bg-red"
	} else if duration > 15*time.Second {
		c = "bg-yellow"
	} else {
		c = "bg-default"
	}

	row[0] = pad(fmt.Sprintf(sdk.Round(duration, time.Second).String()), 9)
	if isWJob {
		fgColor = "fg-white"
		row[2] = pad(fmt.Sprintf("%s", run), 7)
		row[3] = fmt.Sprintf("%s ➤ %s", pad(prj+"/"+workflow, 30), pad(node, 20))
		currentURL = fmt.Sprintf("%s/project/%s/workflow/%s/run/%s", baseURL, prj, workflow, runNumber)
	} else {
		row[2] = pad(fmt.Sprintf("%d", id), 7)
		row[3] = fmt.Sprintf("%s ➤ %s", pad(prj+"/"+app, 30), pad(pip+"/"+bra+"/"+env, 20))
		currentURL = fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%s?envName=%s&branch=%s&version=%s",
			baseURL, prj, app, pip, build, url.QueryEscape(env), url.QueryEscape(bra), version,
		)
		fgColor = "fg-magenta"
	}

	if bookedBy.ID != 0 {
		row[1] = pad(fmt.Sprintf(" %s.%d ", bookedBy.Name, bookedBy.ID), 27)
	} else {
		row[1] = pad("", 27)
	}

	row[4] = fmt.Sprintf("➤ %s", pad(triggeredBy, 17))
	row[5] = fmt.Sprintf("➤ %s", req)

	item := fmt.Sprintf("  [%s](%s)[%s %s %s %s %s](%s,bg-default)", row[0], c, row[1], row[2], row[3], row[4], row[5], fgColor)

	if idx == ui.queue.Cursor-1 {
		ui.currentURL = currentURL
	}
	if maxQueued < duration {
		return item, duration
	}
	return item, maxQueued
}

func statusShort(status string) (string, string) {
	switch status {
	case sdk.StatusWaiting.String():
		return "☕", "fg-cyan"
	case sdk.StatusBuilding.String():
		return "▶", "fg-blue"
	case sdk.StatusDisabled.String():
		return "⏏", "fg-grey"
	case sdk.StatusChecking.String():
		return "♻", "fg-yellow"
	case sdk.StatusSuccess.String():
		return "✔", "fg-green"
	case sdk.StatusFail.String():
		return "✖", "fg-red"
	}
	return status, "fg-default"
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
