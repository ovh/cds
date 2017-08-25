package ui

import (
	"fmt"

	"github.com/gizak/termui"
	"github.com/skratchdot/open-golang/open"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

// Termui wrapper designed for dashboard creation
type Termui struct {
	header *termui.Par
	msg    string

	current  string
	selected string

	// monitoring
	queue                   *cli.ScrollableList
	building                *cli.ScrollableList
	statusWorkerList        *cli.ScrollableList
	statusHatcheriesWorkers *cli.ScrollableList
	statusWorkerModels      *cli.ScrollableList
	status                  *cli.ScrollableList
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
	checking, checkingColor := statusShort(sdk.StatusChecking.String())
	waiting, waitingColor := statusShort(sdk.StatusWaiting.String())
	building, buildingColor := statusShort(sdk.StatusBuilding.String())
	success, successColor := statusShort(sdk.StatusSuccess.String())
	fail, failColor := statusShort(sdk.StatusFail.String())
	disabled, disabledColor := statusShort(sdk.StatusDisabled.String())
	ui.header.Text = fmt.Sprintf(" [CDS | (q)uit | Legend: ](fg-cyan) [Checking:%s](%s)  [Waiting:%s](%s)  [Building:%s](%s)  [Success:%s](%s)  [Fail:%s](%s)  [Disabled:%s](%s) | %s",
		checking, checkingColor,
		waiting, waitingColor,
		building, buildingColor,
		success, successColor,
		fail, failColor,
		disabled, disabledColor,
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
