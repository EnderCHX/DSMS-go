package simulation_mgr

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

const (
	eventName = "simulation"
)

var gMap [][]int

type SimulationMgr struct {
	eventChan   chan string
	VectorClock map[string]uint64
}

func Run() {
	a := app.New()
	w := a.NewWindow("仿真中心")
	w.Resize(fyne.NewSize(800, 600))

	img := canvas.NewImageFromFile("C:\\Users\\c\\Desktop\\chart\\example\\bestof.png")

	c := container.NewGridWithColumns(1,
		img,
	)

	w.SetContent(c)
	w.ShowAndRun()
}
