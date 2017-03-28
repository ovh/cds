package dashboard

import (
	"fmt"
	"time"

	"github.com/gizak/termui"

	"github.com/ovh/cds/sdk"
)

func (ui *Termui) showDashboard() {
	ui.selected = ProjectSelected
	ui.current = DashboardView
	termui.Body.Rows = nil

	ui.dashboard = termui.NewRow()
	ui.appsLayout = termui.NewCol(10, 0)
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, ui.header),
		),
		ui.dashboard,
	)

	var loggers []termui.GridBufferer
	for i := 0; i < 5; i++ {
		l := termui.NewPar("")
		l.Height = termui.TermHeight()
		l.Border = false
		ui.logs = append(ui.logs, l)
		loggers = append(loggers, l)
	}

	var apps []termui.GridBufferer
	for i := 0; i < 20; i++ {
		header := termui.NewPar("")
		header.TextFgColor = termui.ColorCyan
		header.Height = 3
		header.BorderFg = termui.ColorCyan
		header.Border = false
		ui.apps = append(ui.apps, header)
		apps = append(apps, header)
	}

	var pipelines [5][]termui.GridBufferer
	for i := 0; i < 5; i++ {
		for i2 := 0; i2 < 20; i2++ {
			p := termui.NewGauge()
			p.Height = 3
			p.BarColor = termui.ColorBlack
			p.PercentColorHighlighted = termui.ColorBlack
			p.PercentColor = termui.ColorBlack
			p.Border = false
			p.BorderLabel = ""
			p.Percent = 0
			p.Label = ""
			p.PercentColor = termui.ColorDefault
			p.PercentColorHighlighted = termui.ColorDefault
			ui.pipelines[i] = append(ui.pipelines[i], p)
			pipelines[i] = append(pipelines[i], p)
		}
	}

	ui.dashboard.Cols = append(ui.dashboard.Cols, termui.NewCol(1, 0, ui.projects))
	ui.dashboard.Cols = append(ui.dashboard.Cols, termui.NewCol(1, 0, apps...))
	ui.dashboard.Cols = append(ui.dashboard.Cols, termui.NewCol(1, 0, pipelines[0]...))
	ui.dashboard.Cols = append(ui.dashboard.Cols, termui.NewCol(1, 0, pipelines[1]...))
	ui.dashboard.Cols = append(ui.dashboard.Cols, termui.NewCol(1, 0, pipelines[2]...))
	ui.dashboard.Cols = append(ui.dashboard.Cols, termui.NewCol(1, 0, pipelines[3]...))
	ui.dashboard.Cols = append(ui.dashboard.Cols, termui.NewCol(1, 0, pipelines[4]...))
	ui.dashboard.Cols = append(ui.dashboard.Cols, termui.NewCol(5, 0, loggers...))

	termui.Clear()
	ui.draw(0)
	ui.updateProjects()
}

func (ui *Termui) updateLogs(index int, height int, pbJob sdk.PipelineBuildJob) {
	//TODO pipelinebuild
	/*
		ui.logs[index].Height = height
		ui.logs[index].Border = true

		// If offset is not 0, then split logs
		// Remove offset line from logs
		if index+1 == ui.selectedLogs && ui.offset > 0 {

			//l := strings.Split(ab.Logs, "\n")
			l := strings.Count(ab.Logs, "\n")
			if ui.offset > l {
				ui.offset = l - (height - 4)
			}
			if ui.offset < 0 {
				ui.offset = 0
			}
			var lineCount int
			var begin, end int
			for i := 0; i < len(ab.Logs)-1; i++ {
				if ab.Logs[i] == '\n' {
					lineCount++
				}
				if lineCount == ui.offset {
					begin = i
					continue
				}
				end = i
				if lineCount == ui.offset+height {
					break
				}

			}
			ui.logs[index].Text = ab.Logs[begin : end+1]
			//ui.logs[index].Text = strings.Join(l[ui.offset:], "\n")

		} else {
			// For other log view, print last logs
			var lineCount int
			for i := len(ab.Logs) - 1; i > 0; i-- {
				if ab.Logs[i] == '\n' {
					lineCount++
				}
				if lineCount == height-4 {
					ab.Logs = ab.Logs[i:]

					// If offset is 0 but it's the selected log box, save offset
					if index+1 == ui.selectedLogs {
						ui.offset = i
					}
					break
				}
			}
			ui.logs[index].Text = ab.Logs
		}

		// Highligt action name for user
		if index+1 == ui.selectedLogs {
			ui.logs[index].BorderLabel = fmt.Sprintf("[%s](fg-black,bg-white) [%s] (%d lines above)", ab.ActionName, ab.Status, ui.offset)
		} else {
			ui.logs[index].BorderLabel = fmt.Sprintf("%s [%s]", ab.ActionName, ab.Status)
		}

		switch ab.Status {
		case sdk.StatusSuccess:
			ui.logs[index].BorderFg = termui.ColorGreen
			ui.logs[index].BorderLabelFg = termui.ColorGreen
			break
		case sdk.StatusFail:
			ui.logs[index].BorderFg = termui.ColorRed
			ui.logs[index].BorderLabelFg = termui.ColorRed
			break
		case sdk.StatusBuilding:
			ui.logs[index].BorderFg = termui.ColorBlue
			ui.logs[index].BorderLabelFg = termui.ColorBlue
			break
		default:
			ui.logs[index].BorderFg = termui.ColorYellow
			ui.logs[index].BorderLabelFg = termui.ColorYellow
		}
	*/
}

func (ui *Termui) getLogs(proj, app, env, pip string) {
	// Get build state
	state, err := sdk.GetBuildState(proj, app, pip, env, "last")
	if err != nil {
		ui.msg = fmt.Sprintf("[Cannot get build state for %s/%s/%s: %s](bg-red)", proj, app, pip, err)
		return
	}

	// Get all logs data
	var pbJobs []sdk.PipelineBuildJob
	for _, s := range state.Stages {
		for _, pbj := range s.PipelineBuildJobs {
			pbJobs = append(pbJobs, pbj)
		}
	}

	if len(pbJobs) == 0 {
		return
	}

	logParHeight := (termui.TermHeight() - 3) / len(pbJobs)
	var logCount int
	for i := range pbJobs {
		logCount++
		ui.updateLogs(i, logParHeight, pbJobs[i])
	}
	for i := logCount; i < 5; i++ {
		ui.logs[i].Text = ""
		ui.logs[i].BorderLabel = ""
		ui.logs[i].Border = false
	}

	ui.msg = fmt.Sprintf("Logs loaded. Hit <tab> to select, then <j><k>")
}

func (ui *Termui) drawApplications() {
	ui.Lock()
	defer ui.Unlock()

	if len(ui.proj) == 0 || ui.selectedProject <= 0 {
		return
	}
	p := ui.proj[ui.selectedProject-1]

	var appCount int
	for appIndex, app := range p.Applications {
		if appIndex >= 19 {
			break
		}
		appCount++

		var c string
		if appIndex == ui.selectedApp-1 {
			c = fmt.Sprintf("\t[%s](fg-black,bg-white)", app.Name)
		} else {
			c = fmt.Sprintf("\t%s", app.Name)
		}
		// Set application info in application slot
		ui.apps[appIndex].Text = c
		ui.apps[appIndex].Border = true

		// Update application pipelines
		var pipCount int
		for i, pb := range app.PipelinesBuild {

			// Display logs
			if appIndex == ui.selectedApp-1 && i == ui.selectedPipeline-1 {
				go ui.getLogs(p.Key, app.Name, pb.Environment.Name, pb.Pipeline.Name)
			}

			c = fmt.Sprintf("V%d", pb.Version)
			var pipw *termui.Gauge
			if appIndex == ui.selectedApp-1 && i == ui.selectedPipeline-1 {
				name := fmt.Sprintf("[%s](fg-white, bg-black)", pb.Pipeline.Name)
				pipw = newPipelineWidget(name, pb.Environment.Name, c, string(pb.Status))
			} else {
				pipw = newPipelineWidget(pb.Pipeline.Name, pb.Environment.Name, c, string(pb.Status))
			}
			if i < 5 {

				ui.pipelines[i][appIndex].BorderLabel = pipw.BorderLabel
				ui.pipelines[i][appIndex].BarColor = pipw.BarColor
				ui.pipelines[i][appIndex].Percent = 100
				ui.pipelines[i][appIndex].Border = true
				ui.pipelines[i][appIndex].PercentColor = termui.ColorBlack
				ui.pipelines[i][appIndex].PercentColorHighlighted = termui.ColorBlack
				pipCount++
			}
		}

		// Save pipCount of current application for navigation keys
		if appIndex == ui.selectedApp-1 {
			ui.pipCount = pipCount
		}
		// Clear all unused pipeline slot
		for i := pipCount; i < 5; i++ {
			ui.pipelines[i][appIndex].Border = false
			ui.pipelines[i][appIndex].BorderLabel = ""
			ui.pipelines[i][appIndex].Percent = 0
			ui.pipelines[i][appIndex].Label = ""
			ui.pipelines[i][appIndex].PercentColor = termui.ColorDefault
			ui.pipelines[i][appIndex].PercentColorHighlighted = termui.ColorDefault
		}
	}

	// Hide all applications slots not used anymore
	for i := appCount; i < 20; i++ {
		ui.apps[i].Text = ""
		ui.apps[i].Border = false
		for pipi := 0; pipi < 5; pipi++ {
			ui.pipelines[pipi][i].Border = false
			ui.pipelines[pipi][i].BorderLabel = ""
			ui.pipelines[pipi][i].Percent = 0
			ui.pipelines[pipi][i].Label = ""
			ui.pipelines[pipi][i].PercentColor = termui.ColorDefault
			ui.pipelines[pipi][i].PercentColorHighlighted = termui.ColorDefault
		}
	}

	// Set application count (used for key navigation)
	ui.appCount = appCount
}

func newPipelineWidget(name, env, content, status string) *termui.Gauge {
	g := termui.NewGauge()
	g.Percent = 100
	g.Y = 11
	g.BarColor = termui.ColorRed
	g.BorderFg = termui.ColorWhite
	g.BorderLabelFg = termui.ColorCyan
	g.PercentColor = termui.ColorYellow
	g.PercentColorHighlighted = termui.ColorBlack
	g.Height = 3

	if env != "NoEnv" {
		g.Label = fmt.Sprintf("%s %s", env, content)
	} else {
		g.Label = content
	}

	g.BorderLabel = name

	switch status {
	case string(sdk.StatusSuccess):
		g.BarColor = termui.ColorGreen
		break
	case string(sdk.StatusFail):
		g.BarColor = termui.ColorRed
		break
	case string(sdk.StatusBuilding):
		g.BarColor = termui.ColorBlue
		break
	default:
		g.BarColor = termui.ColorYellow
	}

	return g
}

func (ui *Termui) initProjects() {
	ui.selectedProject = 1
	strs := []string{
		"Loading...",
	}

	ls := termui.NewList()
	ls.Items = strs
	ls.ItemFgColor = termui.ColorYellow
	ls.BorderLabel = "Projects"
	ls.Height = 3

	ui.projects = ls
}

func (ui *Termui) updateProjects() {
	projects, err := sdk.ListProject(sdk.WithApplicationStatus())
	if err == nil {
		ui.proj = projects
		ui.drawProjects()
	}

	go func() {
		for {
			time.Sleep(2 * time.Second)
			if ui.current != DashboardView {
				return
			}

			begin := time.Now()
			projects, err := sdk.ListProject(sdk.WithApplicationStatus())
			if err != nil {
				ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())
				continue
			}
			ui.msg = fmt.Sprintf("[Delay: %s](fg-cyan)", time.Since(begin).String())

			ui.proj = projects
			ui.drawProjects()
		}
	}()
}

func (ui *Termui) drawProjects() {
	if ui.selectedProject <= 0 {
		ui.selectedProject = 1
	}
	if ui.selectedProject > len(ui.proj) {
		ui.selectedProject = len(ui.proj)
	}

	var strs []string
	for i := range ui.proj {
		if i+1 == ui.selectedProject {
			strs = append(strs, fmt.Sprintf("[%s](fg-black,bg-white)", ui.proj[i].Name))
		} else {
			strs = append(strs, fmt.Sprintf("%s", ui.proj[i].Name))
		}
	}

	ui.projects.Items = strs
	ui.projects.Height = len(strs) + 2
	ui.drawApplications()
	//termui.Clear()
	ui.draw(0)
}
