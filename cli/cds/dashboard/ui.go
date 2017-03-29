package dashboard

import (
	"fmt"
	"sync"

	"github.com/gizak/termui"
	"github.com/skratchdot/open-golang/open"

	"github.com/ovh/cds/sdk"
)

// Termui wrapper designed for dashboard creation
type Termui struct {
	header *termui.Par
	msg    string

	current string

	// home
	welcomeText *termui.Par

	// dashboard
	proj             []sdk.Project
	selected         string
	selectedProject  int
	selectedApp      int
	selectedPipeline int
	selectedLogs     int
	offset           int
	projects         *termui.List
	appsLayout       *termui.Row
	dashboard        *termui.Row
	logs             []*termui.Par
	apps             []*termui.Par
	appCount         int
	pipCount         int
	pipelines        [5][]*termui.Gauge

	// monitoring
	monitoring *termui.Row
	titles     []*termui.Par
	actions    [5][]*termui.Par
	rowCount   int
	pbs        []sdk.PipelineBuild

	buildingPipelines []*termui.Row

	// status
	infoQueue          string
	distribQueue       map[string]int64
	queue              *ScrollableList
	queueSelect        *ScrollableList
	statusWorkers      *termui.List
	status             *termui.List
	queueCurrentJobURL string

	// mutex
	sync.Mutex
}

// Constants for each view of cds ui
const (
	HomeView         = "home"
	DashboardView    = "dashboard"
	MonitoringView   = "monitoring"
	StatusView       = "status"
	ProjectSelected  = "project"
	AppSelected      = "app"
	PipelineSelected = "pipeline"
	LogsSelected     = "logs"
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

	termui.Handle("/sys/kbd/h", func(termui.Event) {
		ui.showHome()
	})

	termui.Handle("/sys/kbd/d", func(e termui.Event) {
		ui.current = DashboardView
		ui.selectedApp = 1
		ui.showDashboard()
	})

	termui.Handle("/sys/kbd/m", func(e termui.Event) {
		ui.current = "monitoring"
		ui.showMonitoring()
	})

	termui.Handle("/sys/kbd/s", func(e termui.Event) {
		ui.current = "status"
		ui.showStatus()
	})

	termui.Handle("/sys/kbd/k", func(e termui.Event) {
		if ui.selected == LogsSelected && ui.offset > 0 {
			ui.offset -= 5
			ui.drawApplications()
		}
	})
	termui.Handle("/sys/kbd/j", func(e termui.Event) {
		if ui.selected == LogsSelected {
			ui.offset += 5
			ui.drawApplications()
		}
	})
	termui.Handle("/sys/kbd/<tab>", func(e termui.Event) {
		if ui.selected == PipelineSelected && ui.current == DashboardView {
			ui.selected = LogsSelected
			ui.selectedLogs = 1
			ui.drawProjects()
		}
	})

	termui.Handle("/sys/kbd/<down>", func(e termui.Event) {
		switch ui.current {
		case StatusView:
			ui.queue.CursorDown()
		case DashboardView:
			switch ui.selected {
			case ProjectSelected:
				ui.selectedProject++
				ui.selectedApp = 0
				ui.drawProjects()
				break
			case AppSelected:
				if ui.selectedApp < ui.appCount {
					ui.selectedApp++
					ui.drawProjects()
				}
				break
			case LogsSelected:
				ui.selectedLogs++
				ui.offset = 0
				ui.drawProjects()
				break
			case PipelineSelected:
				if ui.selectedApp < ui.appCount {
					ui.selectedApp++
					ui.drawProjects()
					if ui.selectedPipeline > ui.pipCount {
						ui.selectedPipeline = ui.pipCount
					}
				}
				break
			}
		}
	})
	termui.Handle("/sys/kbd/<up>", func(e termui.Event) {
		switch ui.current {
		case StatusView:
			ui.queue.CursorUp()
		case DashboardView:
			switch ui.selected {
			case ProjectSelected:
				ui.selectedProject--
				ui.selectedApp = 0
				ui.drawProjects()
				break
			case AppSelected:
				if ui.selectedApp > 1 {
					ui.selectedApp--
					ui.drawProjects()
				}
				break
			case LogsSelected:
				if ui.selectedLogs > 1 {
					ui.selectedLogs--
					ui.offset = 0
					ui.drawProjects()
				}
				break
			case PipelineSelected:
				if ui.selectedApp > 1 {
					ui.selectedApp--
					ui.drawProjects()
					if ui.selectedPipeline > ui.pipCount {
						ui.selectedPipeline = ui.pipCount
					}
				}
				break
			}
		}
	})
	termui.Handle("/sys/kbd/<left>", func(e termui.Event) {
		if ui.current == DashboardView {
			switch ui.selected {
			case AppSelected:
				ui.selected = ProjectSelected
				ui.selectedApp = 0
				ui.drawProjects()
				break
			case PipelineSelected:
				if ui.selectedPipeline == 1 {
					ui.selected = AppSelected
					ui.selectedPipeline = 0
					ui.drawProjects()
				} else {
					ui.selectedPipeline--
					ui.drawProjects()
				}
				break
			case LogsSelected:
				ui.selected = PipelineSelected
				ui.selectedLogs = 0
				ui.drawProjects()
				break
			}
		}
	})
	termui.Handle("/sys/kbd/<right>", func(e termui.Event) {
		if ui.current == DashboardView {
			switch ui.selected {
			case ProjectSelected:
				ui.selected = AppSelected
				ui.selectedApp = 1
				ui.drawProjects()
				break
			case AppSelected:
				ui.selected = PipelineSelected
				ui.selectedPipeline = 1
				ui.drawProjects()
				break
			case PipelineSelected:
				if ui.selectedPipeline == ui.pipCount {
					ui.selected = LogsSelected
					ui.selectedLogs = 1
					ui.drawProjects()
				} else {
					ui.selectedPipeline++
					ui.drawProjects()
				}
				break
			}
		}
	})

	termui.Handle("/sys/kbd/<enter>", func(e termui.Event) {
		if ui.current == StatusView {
			open.Run(ui.queueCurrentJobURL)
		}
	})

	ui.initProjects()
	ui.initHeader()

	ui.showHome()
}

func (ui *Termui) draw(i int) {
	ui.Lock()
	defer ui.Unlock()

	ui.header.Text = " [CDS | (h)ome | (d)ashboard | (m)onitoring | (s)tatus | (q)uit](fg-cyan) | " + ui.msg
	// calculate layout
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
