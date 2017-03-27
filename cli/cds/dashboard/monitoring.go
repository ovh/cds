package dashboard

import (
	"fmt"
	"time"

	"github.com/gizak/termui"

	"github.com/ovh/cds/sdk"
)

const (
	// MaxRows is the maximum rows displayed
	MaxRows = 80
)

func (ui *Termui) showMonitoring() {
	ui.Lock()
	ui.current = MonitoringView
	termui.Body.Rows = nil
	ui.monitoring = termui.NewRow()
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, ui.header),
		),
		ui.monitoring,
	)

	var titles []termui.GridBufferer
	for i := 0; i < MaxRows; i++ {
		t := termui.NewPar("")
		t.TextFgColor = termui.ColorWhite
		t.Height = 1
		t.Border = false
		ui.titles = append(ui.titles, t)
		titles = append(titles, t)
	}

	var actions [5][]termui.GridBufferer
	for i := 0; i < 5; i++ {
		for i2 := 0; i2 < MaxRows; i2++ {
			p := termui.NewPar("")
			p.Height = 1
			p.TextFgColor = termui.ColorWhite
			p.Border = false
			ui.actions[i] = append(ui.actions[i], p)
			actions[i] = append(actions[i], p)
		}
	}

	ui.monitoring.Cols = append(ui.monitoring.Cols, termui.NewCol(5, 0, titles...))
	ui.monitoring.Cols = append(ui.monitoring.Cols, termui.NewCol(2, 0, actions[0]...))
	ui.monitoring.Cols = append(ui.monitoring.Cols, termui.NewCol(2, 0, actions[1]...))
	ui.monitoring.Cols = append(ui.monitoring.Cols, termui.NewCol(1, 0, actions[2]...))
	ui.monitoring.Cols = append(ui.monitoring.Cols, termui.NewCol(1, 0, actions[3]...))
	ui.monitoring.Cols = append(ui.monitoring.Cols, termui.NewCol(1, 0, actions[4]...))
	ui.Unlock()

	termui.Clear()
	ui.draw(0)
	ui.updateMonitoringPipeline()
}

func newMonitoringPipeline(projKey string, app sdk.Application, pb sdk.PipelineBuild) string {
	var txt string

	buildingChar := "[↻](fg-blue)"
	okChar := "[✓](fg-green)"
	koChar := "[✗](fg-red)"

	branch := pb.Trigger.VCSChangesBranch

	switch pb.Status {
	case sdk.StatusBuilding:
		txt = buildingChar
	case sdk.StatusSuccess:
		txt = okChar
	case sdk.StatusFail:
		txt = koChar
	}
	txt = fmt.Sprintf("%s %s[➤](fg-cyan)%s[➤](fg-cyan)%s[#%d](fg-cyan)", txt, projKey, app.Name, pb.Pipeline.Name, pb.Version)
	if branch != "" {
		txt = fmt.Sprintf("%s ([%s](fg-magenta))", txt, branch)
	}
	if pb.Environment.Name != "NoEnv" {
		txt += fmt.Sprintf(" [%s](fg-magenta)", pb.Environment.Name)
	}

	return txt
}

func (ui *Termui) updateMonitoringPipeline() {
	pbs, err := sdk.GetBuildingPipelines()
	if err == nil {
		ui.pbs = pbs
		ui.drawMonitoringPipelines()
	}

	go func() {
		for {
			time.Sleep(2 * time.Second)
			if ui.current != MonitoringView {
				return
			}

			begin := time.Now()
			pbs, err := sdk.GetBuildingPipelines()
			if err != nil {
				ui.msg = fmt.Sprintf("Cannot load building pipeline: %s", err.Error())
				continue
			}
			ui.msg = fmt.Sprintf("Delay: %s", time.Since(begin).String())

			ui.pbs = pbs
			ui.drawMonitoringPipelines()
		}
	}()

}

func (ui *Termui) drawMonitoringPipelines() {
	ui.Lock()
	defer ui.Unlock()

	if len(ui.pbs) == 0 {
		return
	}

	var pbsCount int
	for pbsI, pb := range ui.pbs {
		if pbsI >= MaxRows-1 {
			break
		}
		pbsCount++
		// Set build info in title slot
		ui.titles[pbsI].Text = newMonitoringPipeline(pb.Pipeline.ProjectKey, pb.Application, pb)
		ui.titles[pbsI].Border = false

		var acount int
		// Update action build, so for each stage
		for _, s := range pb.Stages {
			// add action with its stage
			for _, pbj := range s.PipelineBuildJobs {
				if acount < 5 {
					w := ui.actions[acount][pbsI]
					newActionWidget(pbj.Job.Action.Name, pbj.Status, w)
					acount++
				}
			}
		}
		// Hide unused action slots
		for i := acount; i < 5; i++ {
			ui.actions[i][pbsI].Text = ""
		}

	}

	// Hide all title slots not used anymore
	for i := pbsCount; i < MaxRows; i++ {
		ui.titles[i].Text = ""
		ui.titles[i].Border = false
		for ai := 0; ai < 5; ai++ {
			ui.actions[ai][i].Border = false
			ui.actions[ai][i].BorderLabel = ""
			ui.actions[ai][i].Text = ""
		}
	}

}

func newActionWidget(name string, status string, g *termui.Par) {
	g.Border = false
	g.TextFgColor = termui.ColorWhite

	switch status {
	case string(sdk.StatusSuccess):
		g.Text = fmt.Sprintf("[[%s]](fg-green)", name)
		break
	case string(sdk.StatusFail):
		g.Text = fmt.Sprintf("[[%s]](fg-red)", name)
		break
	case string(sdk.StatusBuilding):
		g.Text = fmt.Sprintf("[[%s]](fg-blue)", name)
		break
	case string(sdk.StatusWaiting):
		g.Text = fmt.Sprintf("[[%s]](fg-yellow)", name)
	default:
		g.Text = fmt.Sprintf("[[%s]](fg-black)", name)
	}
}
