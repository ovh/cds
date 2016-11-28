package dashboard

import (
	"fmt"
	"time"

	"github.com/gizak/termui"

	"github.com/ovh/cds/sdk"
)

func (ui *Termui) drawStatus() {
	ui.workers.BarWidth = int(termui.TermWidth() / (len(ui.workers.DataLabels) + 1))
}

func (ui *Termui) showStatus() {
	ui.current = StatusView
	termui.Body.Rows = nil

	p := termui.NewPar("")
	p.Height = 3
	p.TextFgColor = termui.ColorWhite
	p.BorderLabel = "Status"
	p.BorderFg = termui.ColorCyan
	ui.status = p

	q := termui.NewPar("0")
	q.Height = 3
	q.TextFgColor = termui.ColorWhite
	q.BorderLabel = "Queue"
	q.BorderFg = termui.ColorCyan
	ui.queue = q

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(6, 0, ui.header),
			termui.NewCol(6, 0, ui.msg),
		//ui.NewCol(6, 0, widget1)
		),
	)

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(9, 0, p),
			termui.NewCol(3, 0, ui.queue),
		),
	)

	ui.workers = newWorkerBarChart()
	termui.Body.AddRows(
		termui.NewCol(12, 0, ui.workers),
	)

	ui.updateWorkerStatus()
	ui.updateQueue()
	ui.updateVersion()
	termui.Clear()
	ui.draw(0)
}

func (ui *Termui) updateQueue() {
	queue, err := sdk.GetBuildQueue()
	if err != nil {
		ui.msg.Text = err.Error()
	}
	ui.queue.Text = fmt.Sprintf("%d", len(queue))

	go func() {
		for {
			time.Sleep(1 * time.Second)

			if ui.queue == nil {
				continue
			}

			if ui.current != StatusView {
				return
			}

			queue, err := sdk.GetBuildQueue()
			if err != nil {
				ui.msg.Text = err.Error()
				continue
			}
			ui.queue.Text = fmt.Sprintf("%d", len(queue))
		}
	}()

}

func (ui *Termui) updateVersion() {
	version, err := sdk.GetVersion()
	if err != nil {
		ui.status.Text = fmt.Sprintf("CDS is down (%s)", err)
	}
	ui.status.Text = fmt.Sprintf("CDS is up and running (%s)", version)

	go func() {
		for {
			time.Sleep(1 * time.Second)

			if ui.current != StatusView {
				return
			}

			if ui.queue == nil {
				continue
			}

			version, err := sdk.GetVersion()
			if err != nil {
				ui.status.Text = fmt.Sprintf("CDS is down (%s)", err)
				continue
			}
			ui.status.Text = fmt.Sprintf("CDS is up and running (%s) | Building workers: %d (max:%d)", version, ui.totalBuildingWorkers, ui.maxBuildingWorkers)

		} // for
	}()
}

func (ui *Termui) updateWorkerStatus() {

	var max int64
	var totalBuilding int64

	go func() {
		for {
			max = 0
			totalBuilding = 0

			if ui.workers == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			begin := time.Now()
			wms, err := sdk.GetWorkerModelStatus()
			if err != nil {
				ui.msg.Text = err.Error()
				time.Sleep(1 * time.Second)
				continue
			}
			ui.msg.Text = fmt.Sprintf("Delay: %s", time.Since(begin).String())

			var available, building, missing []int
			var labels []string
			for _, s := range wms {
				if s.BuildingCount == 0 && s.CurrentCount == 0 && s.WantedCount == 0 {
					continue
				}
				totalBuilding += s.BuildingCount
				m := int(s.WantedCount - s.CurrentCount)
				if m < 0 {
					m = 0
				}
				missing = append(missing, m)
				building = append(building, int(s.BuildingCount))
				available = append(available, int(s.CurrentCount))
				labels = append(labels, s.ModelName)
				if s.CurrentCount+s.BuildingCount+(s.WantedCount-s.CurrentCount) > max {
					max = s.CurrentCount + s.BuildingCount + (s.WantedCount - s.CurrentCount)
				}
			}

			almostHeight := int64(termui.TermHeight() - 6)
			if max > almostHeight {
				max = almostHeight
			}

			ui.workers.Data[0] = available
			ui.workers.Data[1] = building
			ui.workers.Data[2] = missing
			ui.workers.DataLabels = labels
			ui.workers.Height = int(almostHeight) // was 15

			ui.totalBuildingWorkers = totalBuilding
			if totalBuilding > ui.maxBuildingWorkers {
				ui.maxBuildingWorkers = totalBuilding
			}
			time.Sleep(1 * time.Second)
		}
	}()

}

func newWorkerBarChart() *termui.MBarChart {
	bc := termui.NewMBarChart()
	data := []int{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	bclabels := []string{"Loading....", "1", "2,", "4", "5", "6", "7", "8", "9", "10"}
	bc.BorderLabel = "Workers"
	bc.ShowScale = false

	bc.Data[0] = data
	bc.Data[1] = data
	bc.Data[2] = data
	bc.Width = 10
	bc.Height = 10
	bc.DataLabels = bclabels
	bc.Y = 30
	bc.SetMax(15)
	bc.BarWidth = 2

	bc.TextColor = termui.ColorWhite
	// Available workers
	bc.BarColor[0] = termui.ColorGreen
	bc.NumColor[0] = termui.ColorBlack
	// Building workers
	bc.BarColor[1] = termui.ColorBlue
	bc.NumColor[1] = termui.ColorBlack
	// Missing workers
	bc.BarColor[2] = termui.ColorRed
	bc.NumColor[2] = termui.ColorBlack
	return bc
}
