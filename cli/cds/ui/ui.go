package ui

import (
	"fmt"

	"github.com/gizak/termui"
	"github.com/skratchdot/open-golang/open"

	"github.com/ovh/cds/sdk"
)

// Termui wrapper designed for dashboard creation
type Termui struct {
	header *termui.Par
	msg    string

	current  string
	selected string

	// monitoring
	queue                   *ScrollableList
	building                *ScrollableList
	statusWorkerList        *ScrollableList
	statusHatcheriesWorkers *ScrollableList
	statusWorkerModels      *ScrollableList
	status                  *ScrollableList
	currentURL              string
}

// Constants for each view of cds ui
const (
	QueueSelected             = "queue"
	BuildingSelected          = "building"
	WorkersListSelected       = "workersList"
	WorkerModelsSelected      = "workerModels"
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
	go ui.showMonitoring()
}

func (ui *Termui) draw(i int) {
	ui.header.Text = fmt.Sprintf(" [CDS | (q)uit | Legend: Checking:%s Waiting:%s Building:%s Success:%s Fail:%s Disabled:%s](fg-cyan) | %s",
		statusShort(sdk.StatusChecking.String()),
		statusShort(sdk.StatusWaiting.String()),
		statusShort(sdk.StatusBuilding.String()),
		statusShort(sdk.StatusSuccess.String()),
		statusShort(sdk.StatusFail.String()),
		statusShort(sdk.StatusDisabled.String()),
		ui.msg)
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
