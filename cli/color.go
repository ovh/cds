package cli

import "github.com/fatih/color"

// Colors... colors everywhere
var (
	Red          = color.New(color.FgRed).SprintfFunc()
	Blue         = color.New(color.FgBlue).SprintfFunc()
	Magenta      = color.New(color.FgMagenta).SprintfFunc()
	Green        = color.New(color.FgGreen).SprintfFunc()
	Cyan         = color.New(color.FgCyan).SprintfFunc()
	BuildingChar = Blue("↻")
	OKChar       = Green("✓")
	KOChar       = Red("✗")
	Arrow        = Cyan("➤")
)
