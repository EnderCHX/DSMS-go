package main

import (
	"image/color"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("盒子布局")

	text1 := canvas.NewText("你好", color.White)
	text2 := canvas.NewText("在那里", color.White)
	text3 := canvas.NewText("(右侧)", color.White)
	content := container.New(layout.NewVBoxLayout(), text1, text2, layout.NewSpacer(), text3)

	text4 := canvas.NewText("居中", color.White)
	centered := container.New(layout.NewVBoxLayout(), layout.NewSpacer(), text4, layout.NewSpacer())
	myWindow.SetContent(container.New(layout.NewHBoxLayout(), content, centered))
	myWindow.ShowAndRun()
}
