package ui

import (
	"github.com/gizak/termui"
)

func (ui *tatui) showResult(cmdResult, result string) {
	ui.current = uiResult
	ui.send.BorderLabel = " ✎ Action"
	termui.Body.Rows = nil

	p := termui.NewPar(`Result of ` + cmdResult + `:
` + result + `
`)

	p.Height = termui.TermHeight() - uiHeightTop - uiHeightSend
	p.TextFgColor = termui.ColorWhite
	p.BorderTop = true

	ui.prepareTopMenu()
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, p),
		),
	)
	ui.prepareSendRow()
	termui.Clear()
}
