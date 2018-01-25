package main

import (
	tm "github.com/buger/goterm"
	"math"
)

func main() {
	tm.Clear()
	tm.MoveCursor(0, 0)

	chart := tm.NewLineChart(100, 20)
	data := new(tm.DataTable)
	data.AddColumn("Time")
	data.AddColumn("Sin(x)")
	data.AddColumn("Cos(x+1)")

	for i := 0.1; i < 10; i += 0.1 {
		data.AddRow(i, math.Sin(i), math.Cos(i+1))
	}

	tm.Println(chart.Draw(data))
	tm.Flush()
}
