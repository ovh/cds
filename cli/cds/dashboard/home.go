package dashboard

import (
	"fmt"

	"github.com/gizak/termui"

	"github.com/ovh/cds/sdk"
)

func (ui *Termui) showHome() {
	ui.current = HomeView
	termui.Body.Rows = nil

	ui.welcomeText = termui.NewPar("")

	ui.dashboard = termui.NewRow()
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, ui.header),
		),
		termui.NewRow(
			termui.NewCol(8, 2, ui.welcomeText),
		),
	)

	ui.welcomeText.Border = false
	ui.welcomeText.Float = termui.AlignCenter
	ui.welcomeText.Height = (termui.TermHeight() / 3)
	ui.welcomeText.Text = fmt.Sprintf(`
	CDS

	version %s

	type 'd' to view your CDS dashboard
	type 'm' to monitor your building pipelines
	type 's' to monitor CDS Queue

	type 'q' to quit
	`, sdk.VERSION)

	termui.Clear()
	ui.draw(0)
}
