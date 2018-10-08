package main

import (
	"context"
	"fmt"
	"math"
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

	if err := termui.Init(); err != nil {
		return nil, err
	}
	defer termui.Close()

	ui := newTermui()
	ui.init()
	ui.staticRender()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ui.loadChan = make(chan func() error)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case f := <-ui.loadChan:
				if err := f(); err != nil {
					ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
				}
				ui.render()
			}
		}
	}()

	ui.renderChan = make(chan func())
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case f := <-ui.renderChan:
				f()
			}
		}
	}()

	termui.Loop()

	return nil, nil
}

func newTermui() *Termui {
	return &Termui{baseURL: "http://cds.ui/"}
}

// Termui wrapper designed for dashboard creation
type Termui struct {
	baseURL          string
	selected         int
	queueTabSelected int
	statusSelected   []sdk.Status
	currentJobURL    string
	msg              string

	me                             *sdk.User
	status                         *sdk.MonitoringStatus
	elapsedStatus                  time.Duration
	workers                        []sdk.Worker
	elapsedWorkers                 time.Duration
	services                       []sdk.Service
	elapsedWorkerModels            time.Duration
	pipelineBuildJob               []sdk.PipelineBuildJob
	workflowNodeJobRun             []sdk.WorkflowNodeJobRun
	elapsedWorkflowNodeJobRun      time.Duration
	elapsedWorkflowNodeJobRunCount time.Duration

	header, times           *termui.Par
	queue                   *cli.ScrollableList
	statusHatcheriesWorkers *cli.ScrollableList
	statusServices          *cli.ScrollableList

	loadChan   chan func() error
	renderChan chan func()
}

func (ui *Termui) loadData() { ui.loadChan <- ui.execLoadData }

func (ui *Termui) execLoadData() error {
	urlUI, err := client.ConfigUser()
	if err != nil {
		return err
	}
	if b, ok := urlUI[sdk.ConfigURLUIKey]; ok {
		ui.baseURL = b
	}

	ui.me, err = client.UserGet(cfg.User)
	if err != nil {
		return fmt.Errorf("Can't get current user: %v", err)
	}

	start := time.Now()
	ui.status, err = client.MonStatus()
	if err != nil {
		return err
	}
	ui.elapsedStatus = time.Since(start)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start = time.Now()
	ui.workers, err = client.WorkerList(ctx)
	if err != nil {
		return err
	}
	ui.elapsedWorkers = time.Since(start)

	if ui.me.Admin {
		ui.services, err = client.ServicesByType("hatchery")
		if err != nil {
			return err
		}
	}

	start = time.Now()
	if _, err := client.WorkerModels(); err != nil {
		return err
	}
	ui.elapsedWorkerModels = time.Since(start)

	ui.pipelineBuildJob, err = client.QueuePipelineBuildJob()
	if err != nil {
		return err
	}

	start = time.Now()
	if _, err := client.QueueCountWorkflowNodeJobRun(nil, nil, "", nil); err != nil {
		return err
	}
	ui.elapsedWorkflowNodeJobRunCount = time.Since(start)

	return nil
}

func (ui *Termui) loadQueue() { ui.loadChan <- ui.execLoadQueue }

func (ui *Termui) execLoadQueue() error {
	var err error

	start := time.Now()
	ui.workflowNodeJobRun, err = client.QueueWorkflowNodeJobRun(ui.statusSelected...)
	if err != nil {
		return err
	}
	ui.elapsedWorkflowNodeJobRun = time.Since(start)

	return nil
}

// Constants for each view of cds ui
const (
	nothingSelected  = -1
	queueSelected    = 0
	servicesSelected = 1
)

func (ui *Termui) init() {
	// init termui handlers
	termui.Merge("/timer/2s", termui.NewTimerCh(time.Second*2))
	termui.Handle("/timer/2s", func(e termui.Event) {
		ui.loadData()
		ui.loadQueue()
	})
	termui.Handle("/sys/kbd/h", func(termui.Event) {
		ui.msg = fmt.Sprintf("shortcuts: ⇥ to select panel, esc to deselect panel, ↑ and ↓ to select line, ← and → to change filters, ↩ to open in ui")
		ui.render()
	})
	termui.Handle("/sys/kbd/q", func(termui.Event) { termui.StopLoop() })
	termui.Handle("/sys/kbd/<tab>", func(e termui.Event) {
		if ui.selected < 1 {
			ui.selected++
		} else {
			ui.selected = 0
		}
		ui.render()
	})
	termui.Handle("/sys/kbd/<escape>", func(e termui.Event) {
		ui.selected = nothingSelected
		ui.render()
	})
	termui.Handle("/sys/kbd", func(e termui.Event) {
		ui.msg = fmt.Sprintf("No command for %v", e)
		ui.render()
	})
	termui.Handle("/sys/kbd/<down>", func(e termui.Event) { ui.moveDown() })
	termui.Handle("/sys/kbd/<up>", func(e termui.Event) { ui.moveUp() })
	termui.Handle("/sys/kbd/<left>", func(e termui.Event) { ui.moveLeft() })
	termui.Handle("/sys/kbd/<right>", func(e termui.Event) { ui.moveRight() })
	termui.Handle("/sys/kbd/<enter>", func(e termui.Event) { ui.enter() })

	ui.header = newPar()
	ui.times = newPar()

	ui.selected = nothingSelected
	ui.updateSelectStatus()

	// prepare queue list
	ui.queue = cli.NewScrollableList()
	ui.queue.BorderLabel = " Queue "
	ui.queue.Height = int(math.Max(float64(termui.TermHeight()-heightBottom), 4))
	ui.queue.Width = termui.TermWidth()
	ui.queue.SetItems("[loading...](fg-cyan)")
	ui.queue.BorderBottom = false
	ui.queue.BorderLeft = false
	ui.queue.BorderRight = false

	// prepare list of hatcheries and workers status
	ui.statusHatcheriesWorkers = cli.NewScrollableList()
	ui.statusHatcheriesWorkers.BorderLabel = " Hatcheries "
	ui.statusHatcheriesWorkers.Height = heightBottom
	ui.statusHatcheriesWorkers.SetItems("[loading...](fg-cyan)")
	ui.statusHatcheriesWorkers.BorderBottom = false
	ui.statusHatcheriesWorkers.BorderLeft = true
	ui.statusHatcheriesWorkers.BorderRight = false

	// prepare services status list
	ui.statusServices = cli.NewScrollableList()
	ui.statusServices.BorderLabel = " Status "
	ui.statusServices.Height = heightBottom
	ui.statusServices.SetItems("[loading...](fg-cyan)")
	ui.statusServices.BorderBottom = false
	ui.statusServices.BorderLeft = false
	ui.statusServices.BorderRight = false

	termui.Body.Rows = nil
	termui.Body.AddRows(
		termui.NewRow(termui.NewCol(12, 0, ui.header)),
		termui.NewRow(termui.NewCol(12, 0, ui.times)),
	)
	termui.Body.AddRows(termui.NewCol(12, 0, ui.queue))
	termui.Body.AddRows(termui.NewRow(
		termui.NewCol(7, 0, ui.statusServices),
		termui.NewCol(5, 0, ui.statusHatcheriesWorkers),
	))
}

func newPar() *termui.Par {
	p := termui.NewPar("")
	p.Height = 1
	p.TextFgColor = termui.ColorWhite
	p.BorderLabel = ""
	p.BorderFg = termui.ColorCyan
	p.Border = false
	return p
}

const (
	heightBottom int = 25
)

func (ui *Termui) staticRender() {
	checking, checkingColor := statusShort(sdk.StatusChecking.String())
	waiting, waitingColor := statusShort(sdk.StatusWaiting.String())
	building, buildingColor := statusShort(sdk.StatusBuilding.String())
	disabled, disabledColor := statusShort(sdk.StatusDisabled.String())
	ui.header.Text = fmt.Sprintf("[CDS | (h)elp | (q)uit | Legend: ](fg-cyan)[Checking:%s](%s) [Waiting:%s](%s) [Building:%s](%s) [Disabled:%s](%s)",
		checking, checkingColor,
		waiting, waitingColor,
		building, buildingColor,
		disabled, disabledColor)

	ui.commonRender()
}

func (ui *Termui) render() { ui.renderChan <- ui.execRender }

func (ui *Termui) execRender() {
	if ui.msg == "" {
		ui.times.Text = fmt.Sprintf(
			"[count queue wf %s](fg-cyan) | [queue wf %s](fg-cyan) | [workers %s](fg-cyan) | [wModels %s](fg-cyan) | [status %s](fg-cyan)",
			sdk.Round(ui.elapsedWorkflowNodeJobRunCount, time.Millisecond).String(),
			sdk.Round(ui.elapsedWorkflowNodeJobRun, time.Millisecond).String(),
			sdk.Round(ui.elapsedWorkers, time.Millisecond).String(),
			sdk.Round(ui.elapsedWorkerModels, time.Millisecond).String(),
			sdk.Round(ui.elapsedStatus, time.Millisecond).String(),
		)
	} else {
		ui.times.Text = ui.msg
		ui.msg = ""
	}

	ui.monitoringColorSelected()
	ui.updateQueue(ui.baseURL)
	ui.computeStatusHatcheriesWorkers(ui.workers)
	ui.updateStatus()

	ui.commonRender()
}

func (ui *Termui) commonRender() {
	termui.Body.Align()
	termui.Render(termui.Body)
	termui.Render()
}

func (ui *Termui) moveDown() {
	switch ui.selected {
	case queueSelected:
		ui.queue.CursorDown()
	case servicesSelected:
		ui.statusServices.CursorDown()
	}
	ui.render()
}

func (ui *Termui) moveUp() {
	switch ui.selected {
	case queueSelected:
		ui.queue.CursorUp()
	case servicesSelected:
		ui.statusServices.CursorUp()
	}
	ui.render()
}

func (ui *Termui) moveLeft() {
	switch ui.selected {
	case queueSelected:
		ui.decrementQueueFilter()
	}
	ui.render()
}

func (ui *Termui) moveRight() {
	switch ui.selected {
	case queueSelected:
		ui.incrementQueueFilter()
	}
	ui.render()
}

func (ui *Termui) enter() {
	switch ui.selected {
	case queueSelected:
		if ui.currentJobURL != "" {
			if err := open.Run(ui.currentJobURL); err != nil {
				ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
				ui.render()
			}
		}
	case servicesSelected:
		item := ui.statusServices.GetItems()[ui.statusServices.GetCursor()]
		if ui.me != nil && ui.me.Admin && strings.Contains(item, "Global/hooks") {
			if err := open.Run(fmt.Sprintf("%s/admin/hooks-tasks", ui.baseURL)); err != nil {
				ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
				ui.render()
			}
		} else {
			ui.msg = "nothing to do for this service"
			ui.render()
		}
	}
}

func (ui *Termui) incrementQueueFilter() {
	if ui.queueTabSelected < 2 {
		ui.queueTabSelected++
	} else {
		ui.queueTabSelected = 0
	}
	ui.updateSelectStatus()
	ui.loadQueue()
}

func (ui *Termui) decrementQueueFilter() {
	if 0 < ui.queueTabSelected {
		ui.queueTabSelected--
	} else {
		ui.queueTabSelected = 2
	}
	ui.updateSelectStatus()
	ui.loadQueue()
}

func (ui *Termui) updateSelectStatus() {
	switch ui.queueTabSelected {
	case 0:
		ui.statusSelected = []sdk.Status{sdk.StatusWaiting}
	case 1:
		ui.statusSelected = []sdk.Status{sdk.StatusBuilding}
	case 2:
		ui.statusSelected = []sdk.Status{sdk.StatusBuilding, sdk.StatusWaiting}
	}
}

func (ui *Termui) monitoringColorSelected() {
	ui.queue.BorderFg = termui.ColorDefault
	ui.statusServices.BorderFg = termui.ColorDefault
	ui.statusHatcheriesWorkers.BorderFg = termui.ColorDefault

	ui.queue.SetCursorVisibility(ui.selected == queueSelected)
	ui.statusServices.SetCursorVisibility(ui.selected == servicesSelected)

	switch ui.selected {
	case queueSelected:
		ui.queue.BorderFg = termui.ColorRed
	case servicesSelected:
		ui.statusServices.BorderFg = termui.ColorRed
	}

	termui.Render(ui.queue, ui.statusHatcheriesWorkers, ui.statusServices)
}

func (ui *Termui) updateStatus() {
	var items []string
	if ui.status != nil {
		for _, l := range ui.status.Lines {
			lineSelected := ui.selected == servicesSelected && len(items) == ui.statusServices.GetCursor()

			if !lineSelected {
				if l.Status == sdk.MonitoringStatusWarn {
					items = append(items, fmt.Sprintf("[%s](fg-yellow)", l.String()))
				} else if l.Status != sdk.MonitoringStatusOK {
					items = append(items, fmt.Sprintf("[%s](bg-red)", l.String()))
				} else if strings.Contains(l.Component, "Global") {
					items = append(items, fmt.Sprintf("%s", l.String()))
				}
			} else if l.Status != sdk.MonitoringStatusOK ||
				strings.Contains(l.Component, "Global") {
				items = append(items, fmt.Sprintf("%s", l.String()))
			}
		}
	}
	ui.statusServices.SetItems(items...)
}

func (ui *Termui) computeStatusHatcheriesWorkers(workers []sdk.Worker) {
	hatcheryNames, statusTitle := []string{}, []string{}
	hatcheries := make(map[string]map[string]int64)
	status := make(map[string]int)

	if ui.me != nil && ui.me.Admin {
		for _, s := range ui.services {
			if _, ok := hatcheries[s.Name]; !ok {
				hatcheries[s.Name] = make(map[string]int64)
				hatcheryNames = append(hatcheryNames, s.Name)
			}
		}
	}

	without := "Without hatchery"
	hatcheries[without] = make(map[string]int64)
	hatcheryNames = append(hatcheryNames, without)

	for _, w := range workers {
		var name string
		if w.HatcheryName == "" {
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

	var items []string
	sort.Strings(hatcheryNames)
	for _, name := range hatcheryNames {
		v := hatcheries[name]
		var t string
		for _, status := range statusTitle {
			if v[status] > 0 {
				icon, color := statusShort(status)
				t = fmt.Sprintf("%s[ %d%s ](%s)", t, v[status], icon, color)
			}
		}
		if len(t) == 0 {
			t = fmt.Sprintf("%s[ _ ](fg-white)", t)
		}
		t = fmt.Sprintf("%s[ %s](fg-white)", t, name)
		items = append(items, t)
	}
	ui.statusHatcheriesWorkers.SetItems(items...)

	sort.Strings(statusTitle)
	title := " Hatcheries "
	for _, s := range statusTitle {
		icon, color := statusShort(s)
		title = fmt.Sprintf("%s[%d%s](%s) ", title, status[s], icon, color)
	}
	ui.statusHatcheriesWorkers.BorderLabel = title
}

func (ui *Termui) updateQueue(baseURL string) {
	mapLines := map[sdk.Status][]string{}
	var lineCount int
	var maxWaiting, maxBuilding time.Duration
	for _, job := range ui.pipelineBuildJob {
		duration := time.Since(job.Queued)
		s := sdk.Status(job.Status)

		switch s {
		case sdk.StatusWaiting:
			if maxWaiting == 0 || maxWaiting < duration {
				maxWaiting = duration
			}
		case sdk.StatusBuilding:
			if maxBuilding == 0 || maxBuilding < duration {
				maxBuilding = duration
			}
		}

		mapLines[s] = append(mapLines[s], ui.generateQueueJobLine(false, lineCount, job.ID,
			job.Parameters, job.Job, duration, job.BookedBy, baseURL, job.Status))

		if lineCount == ui.queue.GetCursor() {
			ui.currentJobURL = generateQueueJobURL(false, baseURL, job.Parameters)
		}

		lineCount++
	}
	for _, job := range ui.workflowNodeJobRun {
		duration := time.Since(job.Queued)
		s := sdk.Status(job.Status)

		if (maxWaiting == 0 || maxWaiting < duration) && job.Status == sdk.StatusWaiting.String() {
			maxWaiting = duration
		}
		if (maxBuilding == 0 || maxBuilding < duration) && job.Status == sdk.StatusBuilding.String() {
			maxBuilding = duration
		}

		mapLines[s] = append(mapLines[s], ui.generateQueueJobLine(true, lineCount, job.ID, job.Parameters, job.Job,
			time.Since(job.Queued), job.BookedBy, baseURL, job.Status))

		if lineCount == ui.queue.GetCursor() {
			ui.currentJobURL = generateQueueJobURL(true, baseURL, job.Parameters)
		}

		lineCount++
	}

	ui.queue.SetHeader(fmt.Sprintf("[_ %s %s%s %s ➤ %s ➤ %s ➤ %s](fg-cyan)",
		pad("since", 9), pad("by", 27), pad("run", 7), pad("project/workflow", 30),
		pad("node", 20), pad("triggered by", 17), "requirements"))
	var items []string
	for _, s := range ui.statusSelected {
		if m, ok := mapLines[s]; ok {
			for _, l := range m {
				items = append(items, l)
			}
		}
	}
	ui.queue.SetItems(items...)

	ui.queue.BorderLabel = fmt.Sprintf(" Queue(%s):%d ",
		strings.Join(sdk.StatusToStrings(ui.statusSelected), ","), len(items))

	for _, s := range ui.statusSelected {
		switch s {
		case sdk.StatusBuilding:
			ui.queue.BorderLabel = fmt.Sprintf("%s- Max Building:%s ", ui.queue.BorderLabel,
				sdk.Round(maxBuilding, time.Second).String())
		case sdk.StatusWaiting:
			ui.queue.BorderLabel = fmt.Sprintf("%s- Max Waiting:%s ", ui.queue.BorderLabel,
				sdk.Round(maxWaiting, time.Second).String())
		}
	}
}

func generateQueueJobURL(isWJob bool, baseURL string, parameters []sdk.Parameter) string {
	prj := getVarsInPbj("cds.project", parameters)
	workflow := getVarsInPbj("cds.workflow", parameters)
	runNumber := getVarsInPbj("cds.run.number", parameters)
	if isWJob {
		return fmt.Sprintf("%s/project/%s/workflow/%s/run/%s", baseURL, prj, workflow, runNumber)
	}

	app := getVarsInPbj("cds.application", parameters)
	pip := getVarsInPbj("cds.pipeline", parameters)
	build := getVarsInPbj("cds.buildNumber", parameters)
	env := getVarsInPbj("cds.environment", parameters)
	bra := getVarsInPbj("git.branch", parameters)
	version := getVarsInPbj("cds.version", parameters)
	return fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%s?envName=%s&branch=%s&version=%s",
		baseURL, prj, app, pip, build, url.QueryEscape(env), url.QueryEscape(bra), version)
}

func (ui *Termui) generateQueueJobLine(isWJob bool, idx int, id int64, parameters []sdk.Parameter, executedJob sdk.ExecutedJob,
	duration time.Duration, bookedBy sdk.Service, baseURL, status string) string {
	var req string
	for _, r := range executedJob.Job.Action.Requirements {
		req = fmt.Sprintf("%s%s:%s ", req, r.Type, r.Value)
	}
	prj := getVarsInPbj("cds.project", parameters)
	app := getVarsInPbj("cds.application", parameters)
	pip := getVarsInPbj("cds.pipeline", parameters)
	workflow := getVarsInPbj("cds.workflow", parameters)
	node := getVarsInPbj("cds.node", parameters)
	run := getVarsInPbj("cds.run", parameters)
	env := getVarsInPbj("cds.environment", parameters)
	bra := getVarsInPbj("git.branch", parameters)
	triggeredBy := getVarsInPbj("cds.triggered_by.username", parameters)

	row := make([]string, 6)
	row[0] = pad(fmt.Sprintf(sdk.Round(duration, time.Second).String()), 9)
	if isWJob {
		row[2] = pad(fmt.Sprintf("%s", run), 7)
		row[3] = fmt.Sprintf("%s ➤ %s", pad(prj+"/"+workflow, 30), pad(node, 20))
	} else {
		row[2] = pad(fmt.Sprintf("%d", id), 7)
		row[3] = fmt.Sprintf("%s ➤ %s", pad(prj+"/"+app, 30), pad(pip+"/"+bra+"/"+env, 20))
	}

	if status == sdk.StatusBuilding.String() {
		row[1] = pad(fmt.Sprintf(" %s.%s ", executedJob.WorkerName, executedJob.WorkerID), 27)
	} else if bookedBy.ID != 0 {
		row[1] = pad(fmt.Sprintf(" %s.%d ", bookedBy.Name, bookedBy.ID), 27)
	} else {
		row[1] = pad("", 27)
	}

	row[4] = fmt.Sprintf("➤ %s", pad(triggeredBy, 17))
	row[5] = fmt.Sprintf("➤ %s", req)

	_, color := statusShort(status)
	color = strings.Replace(color, "fg", "bg", 1)

	var c string
	if status == sdk.StatusWaiting.String() {
		if duration > 60*time.Second {
			c = "fg-black,bg-red"
		} else if duration > 15*time.Second {
			c = "fg-black,bg-yellow"
		}
	}

	if isWJob {
		return fmt.Sprintf("[ ](%s) [%s](%s)%s %s %s %s %s",
			color, row[0], c, row[1], row[2], row[3], row[4], row[5])
	}

	return fmt.Sprintf("[ ](%s) [%s](%s)[%s %s %s %s %s](fg-magenta)",
		color, row[0], c, row[1], row[2], row[3], row[4], row[5])
}

func statusShort(status string) (string, string) {
	switch status {
	case sdk.StatusWaiting.String():
		return "w", "fg-cyan"
	case sdk.StatusBuilding.String():
		return "b", "fg-blue"
	case sdk.StatusDisabled.String():
		return "d", "fg-grey"
	case sdk.StatusChecking.String():
		return "c", "fg-yellow"
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
