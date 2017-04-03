package dashboard

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gizak/termui"

	"github.com/ovh/cds/sdk"
)

func (ui *Termui) showMonitoring() {
	ui.current = MonitoringView
	termui.Body.Rows = nil

	ui.queue = NewScrollableList()
	ui.queue.ItemFgColor = termui.ColorWhite
	ui.queue.ItemBgColor = termui.ColorBlack

	heightBottom := 13

	ui.queue.BorderLabel = " Queue "
	ui.queue.Height = (termui.TermHeight() - heightBottom) / 2
	ui.queue.Width = termui.TermWidth()
	ui.queue.Items = []string{"Loading..."}
	ui.queue.BorderBottom = false
	ui.queue.BorderLeft = false
	ui.queue.BorderRight = false

	ui.selected = QueueSelected

	ui.building = NewScrollableList()
	ui.building.ItemFgColor = termui.ColorWhite
	ui.building.ItemBgColor = termui.ColorBlack

	ui.building.BorderLabel = " Building "
	ui.building.Height = (termui.TermHeight() - heightBottom) / 2
	ui.building.Width = termui.TermWidth()
	ui.building.Items = []string{"Loading..."}
	ui.building.BorderBottom = false
	ui.building.BorderLeft = false
	ui.building.BorderRight = false

	ui.statusWorkerList = NewScrollableList()
	ui.statusWorkerList.ItemFgColor = termui.ColorWhite
	ui.statusWorkerList.ItemBgColor = termui.ColorBlack

	ui.statusWorkerList.BorderLabel = " Workers List "
	ui.statusWorkerList.Height = heightBottom
	ui.statusWorkerList.Items = []string{"[select a job](fg-cyan,bg-default)"}
	ui.statusWorkerList.BorderBottom = false
	ui.statusWorkerList.BorderLeft = false

	ui.statusWorkerModels = NewScrollableList()
	ui.statusWorkerModels.BorderLabel = " Worker Models "
	ui.statusWorkerModels.Height = heightBottom
	ui.statusWorkerModels.Items = []string{"[loading...](fg-cyan,bg-default)"}
	ui.statusWorkerModels.BorderBottom = false
	ui.statusWorkerModels.BorderLeft = false

	ui.statusHatcheriesWorkers = NewScrollableList()
	ui.statusHatcheriesWorkers.BorderLabel = " Hatcheries Workers "
	ui.statusHatcheriesWorkers.Height = heightBottom
	ui.statusHatcheriesWorkers.Items = []string{"[loading...](fg-cyan,bg-default)"}
	ui.statusHatcheriesWorkers.BorderBottom = false
	ui.statusHatcheriesWorkers.BorderLeft = false
	ui.statusHatcheriesWorkers.BorderRight = false

	ui.status = NewScrollableList()
	ui.status.BorderLabel = " Status "
	ui.status.Height = heightBottom
	ui.status.Items = []string{"[loading...](fg-cyan,bg-default)"}
	ui.status.BorderBottom = false
	ui.status.BorderLeft = true
	ui.status.BorderRight = false

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, ui.header),
		),
	)

	termui.Body.AddRows(
		termui.NewCol(12, 0, ui.queue),
		termui.NewCol(12, 0, ui.building),
	)
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(4, 0, ui.statusWorkerList),
			termui.NewCol(3, 0, ui.statusWorkerModels),
			termui.NewCol(3, 0, ui.statusHatcheriesWorkers),
			termui.NewCol(2, 0, ui.status),
		),
	)

	baseURL := "http://cds.ui/"
	urlUI, err := sdk.GetConfigUser()
	if err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
	}

	if b, ok := urlUI[sdk.ConfigURLUIKey]; ok {
		baseURL = b
	}

	ticker := time.NewTicker(2 * time.Second).C

	for {
		if ui.current != MonitoringView {
			return
		}

		var a, b, c, d string
		select {
		case <-ticker:
			ui.monitoringColorSelected()
			a = ui.updateQueue(baseURL)
			b = ui.updateQueueWorkers()
			c = ui.updateBuilding(baseURL)
			d = ui.updateStatus()
		}
		ui.msg = fmt.Sprintf("%s | %s | %s | %s", a, b, c, d)
		termui.Render()
	}
}

func (ui *Termui) monitoringCursorDown() {
	switch ui.selected {
	case QueueSelected:
		ui.queue.CursorDown()
	case BuildingSelected:
		ui.building.CursorDown()
	case WorkersListSelected:
		ui.statusWorkerList.CursorDown()
	case WorkerModelsSelected:
		ui.statusWorkerModels.CursorDown()
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
	case BuildingSelected:
		ui.building.CursorUp()
	case WorkersListSelected:
		ui.statusWorkerList.CursorUp()
	case WorkerModelsSelected:
		ui.statusWorkerModels.CursorUp()
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
	case BuildingSelected:
		ui.selected = WorkersListSelected
		ui.building.Cursor = 0
	case WorkersListSelected:
		ui.selected = WorkerModelsSelected
		ui.statusWorkerList.Cursor = 0
	case WorkerModelsSelected:
		ui.selected = HatcheriesWorkersSelected
		ui.statusWorkerModels.Cursor = 0
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
	ui.building.BorderFg = termui.ColorDefault
	ui.statusWorkerList.BorderFg = termui.ColorDefault
	ui.statusWorkerModels.BorderFg = termui.ColorDefault
	ui.statusHatcheriesWorkers.BorderFg = termui.ColorDefault
	ui.status.BorderFg = termui.ColorDefault

	switch ui.selected {
	case QueueSelected:
		ui.queue.BorderFg = termui.ColorRed
	case BuildingSelected:
		ui.building.BorderFg = termui.ColorRed
	case WorkersListSelected:
		ui.statusWorkerList.BorderFg = termui.ColorRed
	case WorkerModelsSelected:
		ui.statusWorkerModels.BorderFg = termui.ColorRed
	case HatcheriesWorkersSelected:
		ui.statusHatcheriesWorkers.BorderFg = termui.ColorRed
	case StatusSelected:
		ui.status.BorderFg = termui.ColorRed
	}
	termui.Render(ui.queue,
		ui.building,
		ui.statusWorkerList,
		ui.statusWorkerModels,
		ui.statusHatcheriesWorkers,
		ui.status)
}

func (ui *Termui) updateStatus() string {
	start := time.Now()
	status, err := sdk.GetStatus()
	if err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
		return ""
	}
	elapsed := time.Since(start)
	msg := fmt.Sprintf("[getStatus %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())

	selected := "fg-white,bg-default"
	if ui.selected == StatusSelected {
		selected = "fg-white"
	}

	items := []string{}
	for _, l := range status {
		if strings.HasPrefix(l, "Version") ||
			strings.HasPrefix(l, "Uptime") ||
			strings.HasPrefix(l, "Nb of Panics: 0") ||
			strings.HasPrefix(l, "Secret Backend") ||
			strings.HasPrefix(l, "Cache: local") ||
			strings.HasPrefix(l, "Session-Store: In Memory") ||
			strings.Contains(l, "OK") {
			items = append(items, fmt.Sprintf("[%s](%s)", l, selected))
		} else {
			items = append(items, fmt.Sprintf("[%s](bg-red)", l))
		}
	}
	ui.status.Items = items
	return msg
}

func (ui *Termui) updateBuilding(baseURL string) string {
	start := time.Now()
	pbs, err := sdk.GetBuildingPipelines()
	if err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
		return ""
	}
	elapsed := time.Since(start)
	msg := fmt.Sprintf("[getBuildingPipelines %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())

	items := []string{fmt.Sprintf("[  %s➤ %s➤ %s➤ %s](fg-cyan,bg-default)", pad("project/application", 35), pad("pipeline", 25), pad("branch/env", 19), "jobs")}
	for i, pb := range pbs {
		t := ui.pipelineLine(pb.Application.ProjectKey, pb.Application, pb)
		for _, s := range pb.Stages {
			t += "[ ➤](fg-cyan,bg-default)"
			for _, pbj := range s.PipelineBuildJobs {
				t += jobLine(pbj.Job.Action.Name, pbj.Status)
			}
		}
		items = append(items, t)

		if i == ui.building.Cursor-1 {
			ui.currentURL = computeURL(baseURL, pb.Application.ProjectKey, pb.Application.Name, pb.Pipeline.Name, fmt.Sprintf("%d", pb.BuildNumber), pb.Environment.Name)
		}
	}
	ui.building.Items = items
	return msg
}

func (ui *Termui) pipelineLine(projKey string, app sdk.Application, pb sdk.PipelineBuild) string {
	var txt string

	selected := ",bg-default"
	if ui.selected == BuildingSelected {
		selected = "fg-white"
	}
	buildingChar := fmt.Sprintf("[↻](fg-blue%s)", selected)
	okChar := fmt.Sprintf("[✓](fg-green%s)", selected)
	koChar := fmt.Sprintf("[✗](fg-red%s)", selected)

	branch := pb.Trigger.VCSChangesBranch

	switch pb.Status {
	case sdk.StatusBuilding:
		txt = buildingChar
	case sdk.StatusSuccess:
		txt = okChar
	case sdk.StatusFail:
		txt = koChar
	}

	txt = fmt.Sprintf("%s[ %s](bg-default)[➤](fg-cyan,bg-default)[%s ](bg-default)[➤](fg-cyan,bg-default)[%s](bg-default)", txt, pad(projKey+"/"+app.Name, 35), pad(pb.Pipeline.Name, 25), pad(branch+"/"+pb.Environment.Name, 19))
	return txt
}

func jobLine(name string, status string) string {
	switch status {
	case string(sdk.StatusSuccess):
		return fmt.Sprintf("[ [%s]](fg-green,bg-default)", name)
	case string(sdk.StatusFail):
		return fmt.Sprintf("[ [%s]](fg-red,bg-default)", name)
	case string(sdk.StatusBuilding):
		return fmt.Sprintf("[ [%s]](fg-blue,bg-default)", name)
	case string(sdk.StatusWaiting):
		return fmt.Sprintf("[ [%s]](fg-yellow,bg-default)", name)
	default:
		return fmt.Sprintf("[ [%s]](fg-black,bg-default)", name)
	}
}

func (ui *Termui) updateQueueWorkers() string {
	start := time.Now()
	workers, err := sdk.GetWorkers()
	if err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
		return ""
	}
	elapsed := time.Since(start)
	msg := fmt.Sprintf("[getWorkers %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())

	ui.computeStatusHatcheriesWorkers(workers)

	msga, wmodels := ui.computeStatusWorkerModels(workers)
	ui.computeStatusWorkersList(workers, wmodels)
	return msg + msga
}

func (ui *Termui) computeStatusHatcheriesWorkers(workers []sdk.Worker) {
	idsHatcheries, statusTitle := []string{}, []string{}
	hatcheries := make(map[string]map[string]int64)
	status := make(map[string]int)

	for _, w := range workers {
		var id string
		if w.HatcheryID == 0 {
			id = "Without hatchery"
		} else {
			id = fmt.Sprintf("%d", w.HatcheryID)
		}

		if _, ok := hatcheries[id]; !ok {
			hatcheries[id] = make(map[string]int64)
			idsHatcheries = append(idsHatcheries, id)
		}
		hatcheries[id][w.Status.String()] = hatcheries[id][w.Status.String()] + 1

		if _, ok := status[w.Status.String()]; !ok {
			statusTitle = append(statusTitle, w.Status.String())
		}
		status[w.Status.String()] = status[w.Status.String()] + 1
	}

	selected := "fg-white,bg-default"
	if ui.selected == HatcheriesWorkersSelected {
		selected = "fg-white"
	}

	items := []string{}
	sort.Strings(idsHatcheries)
	for _, id := range idsHatcheries {
		v := hatcheries[id]
		t := fmt.Sprintf("%s ", id)
		for status, nb := range v {
			t += fmt.Sprintf("%s:%d ", status, nb)
		}
		items = append(items, fmt.Sprintf("[%s](%s)", t, selected))
	}
	sort.Strings(items)
	ui.statusHatcheriesWorkers.Items = items

	sort.Strings(statusTitle)
	title := " Hatcheries Workers "
	for _, s := range statusTitle {
		title += fmt.Sprintf("%s:%d ", s, status[s])
	}
	ui.statusHatcheriesWorkers.BorderLabel = title
}

func (ui *Termui) computeStatusWorkersList(workers []sdk.Worker, wModels map[string]sdk.Model) {
	items := []string{}
	selected := "fg-white,bg-default"
	if ui.selected == WorkersListSelected {
		selected = "fg-white"
	}
	for _, w := range workers {
		idModel := fmt.Sprintf("%d", w.Model)
		items = append(items, fmt.Sprintf("[%s %s](%s)", pad(wModels[idModel].Type+" "+wModels[idModel].Name+" "+w.Name, 60), w.Status, selected))
	}
	sort.Strings(items)
	ui.statusWorkerList.Items = items
}

func (ui *Termui) computeStatusWorkerModels(workers []sdk.Worker) (string, map[string]sdk.Model) {
	start := time.Now()
	workerModels, errwm := sdk.GetWorkerModels()
	if errwm != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", errwm.Error())
		return "", nil
	}
	elapsed := time.Since(start)
	msg := fmt.Sprintf(" | [getWorkerModels %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())

	wModels := make(map[string]sdk.Model, len(workerModels))
	for w := range workerModels {
		id := strconv.FormatInt(workerModels[w].ID, 10)
		wModels[id] = workerModels[w]
	}

	items, idsModels := []string{}, []string{}
	models := make(map[string]map[string]int64)

	for _, w := range workers {
		idModel := fmt.Sprintf("%d", w.Model)
		if _, ok := models[idModel]; !ok {
			models[idModel] = make(map[string]int64)
			idsModels = append(idsModels, idModel)
		}
		models[idModel][w.Status.String()] = models[idModel][w.Status.String()] + 1
	}

	selected := "fg-white,bg-default"
	if ui.selected == WorkerModelsSelected {
		selected = "fg-white"
	}

	sort.Strings(idsModels)
	for _, id := range idsModels {
		t := fmt.Sprintf("%s ", pad(wModels[id].Type+" "+wModels[id].Name, 28))
		v := models[id]
		for status, nb := range v {
			t += fmt.Sprintf("%s:%d ", status, nb)
		}
		items = append(items, fmt.Sprintf("[%s](%s)", t, selected))
	}
	sort.Strings(items)
	ui.statusWorkerModels.Items = items

	return msg, wModels
}

func (ui *Termui) updateQueue(baseURL string) string {
	start := time.Now()
	pbJobs, err := sdk.GetBuildQueue()
	if err != nil {
		ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
		return ""
	}
	elapsed := time.Since(start)
	msg := fmt.Sprintf("[getQueue %s](fg-cyan,bg-default)", sdk.Round(elapsed, time.Millisecond).String())

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
			ui.currentURL = computeURL(baseURL, prj, app, pip, build, env)
		}
	}
	ui.queue.Items = items

	t := fmt.Sprintf(" Queue:%d Max Waiting:%s ", len(pbJobs), sdk.Round(maxQueued, time.Second).String())
	for name, total := range booked {
		t += fmt.Sprintf("%s:%d ", name, total)
	}
	ui.queue.BorderLabel = t
	return msg
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
