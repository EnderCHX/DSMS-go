package main

import (
	"fmt"
	"fyne.io/fyne/v2/dialog"
	"github.com/joho/godotenv"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/draw"
)

type Point struct {
	X, Y   float64 // 虚拟坐标系中的位置
	Color  color.Color
	Radius float64
}

type CoordinateWidget struct {
	widget.BaseWidget
	offsetX, offsetY float64
	scale            float64
	raster           *canvas.Raster
	points           []Point           // 存储所有点
	selectedPoint    int               // 当前选中的点索引
	popUpMenu        *widget.PopUpMenu // 右键菜单
}

func NewCoordinateWidget() *CoordinateWidget {
	cw := &CoordinateWidget{
		scale:  1.0,
		points: make([]Point, 0),
	}
	cw.raster = canvas.NewRaster(cw.draw)
	cw.ExtendBaseWidget(cw)
	return cw
}

// 添加点的方法
func (cw *CoordinateWidget) AddPoint(virtualX, virtualY float64) {
	cw.points = append(cw.points, Point{
		X:      virtualX,
		Y:      virtualY,
		Color:  color.RGBA{R: 255, A: 255},
		Radius: 3,
	})
	cw.raster.Refresh()
}

// 删除点的方法
func (cw *CoordinateWidget) RemovePoint(index int) {
	if index >= 0 && index < len(cw.points) {
		cw.points = append(cw.points[:index], cw.points[index+1:]...)
		cw.raster.Refresh()
	}
}

func (cw *CoordinateWidget) draw(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	// 计算可视区域边界（虚拟坐标系）
	viewLeft := (-cw.offsetX) / cw.scale
	viewRight := (float64(w) - cw.offsetX) / cw.scale
	viewTop := (-cw.offsetY) / cw.scale
	viewBottom := (float64(h) - cw.offsetY) / cw.scale

	// 动态计算网格间距
	gridSize := cw.calculateGridSize(viewRight - viewLeft)

	// 绘制垂直网格线
	startX := math.Floor(viewLeft/gridSize) * gridSize
	endX := math.Ceil(viewRight/gridSize) * gridSize
	for x := startX; x <= endX; x += gridSize {
		screenX := x*cw.scale + cw.offsetX
		if screenX >= 0 && screenX < float64(w) {
			drawVerticalLine(img, int(screenX), color.RGBA{200, 200, 200, 255})
		}
	}

	// 绘制水平网格线
	startY := math.Floor(viewTop/gridSize) * gridSize
	endY := math.Ceil(viewBottom/gridSize) * gridSize
	for y := startY; y <= endY; y += gridSize {
		screenY := y*cw.scale + cw.offsetY
		if screenY >= 0 && screenY < float64(h) {
			drawHorizontalLine(img, int(screenY), color.RGBA{200, 200, 200, 255})
		}
	}

	// 绘制坐标轴
	drawAxis(img, cw.offsetX, cw.offsetY, w, h, cw.scale)

	// 绘制所有点
	for _, p := range cw.points {
		screenX := p.X*cw.scale + cw.offsetX
		screenY := p.Y*cw.scale + cw.offsetY
		drawCircle(img, int(screenX), int(screenY), int(p.Radius), p.Color)
	}

	return img
}

// 新增：绘制圆形
func drawCircle(img *image.RGBA, x, y, radius int, c color.Color) {
	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			if dx*dx+dy*dy <= radius*radius {
				img.Set(x+dx, y+dy, c)
			}
		}
	}
}

// 实现右键菜单功能
func (cw *CoordinateWidget) TappedSecondary(e *fyne.PointEvent) {
	// 转换屏幕坐标到虚拟坐标
	//virtualX := (float64(e.Position.X) - cw.offsetX) / cw.scale
	//virtualY := (float64(e.Position.Y) - cw.offsetY) / cw.scale

	// 检查是否点击在已有点上
	clickTolerance := 5.0 // 像素容差
	for i, p := range cw.points {
		screenX := p.X*cw.scale + cw.offsetX
		screenY := p.Y*cw.scale + cw.offsetY
		if math.Abs(float64(e.Position.X)-screenX) < clickTolerance &&
			math.Abs(float64(e.Position.Y)-screenY) < clickTolerance {
			// 显示删除菜单
			cw.selectedPoint = i
			cw.showContextMenu(e.AbsolutePosition, true)
			return
		}
	}

	// 显示添加菜单
	cw.showContextMenu(e.AbsolutePosition, false)
}

func (cw *CoordinateWidget) showContextMenu(pos fyne.Position, isPoint bool) {
	var items []*fyne.MenuItem

	if isPoint {
		items = []*fyne.MenuItem{
			fyne.NewMenuItem("删除该点", func() {
				cw.RemovePoint(cw.selectedPoint)
			}),
		}
	} else {
		items = []*fyne.MenuItem{
			fyne.NewMenuItem("在此处添加点", func() {
				virtualX := (float64(pos.X) - cw.offsetX) / cw.scale
				virtualY := (float64(pos.Y) - cw.offsetY) / cw.scale
				cw.AddPoint(virtualX, virtualY)
			}),
		}
	}

	menu := fyne.NewMenu("操作", items...)
	cw.popUpMenu = widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(cw))
	cw.popUpMenu.ShowAtPosition(pos)
}

func (cw *CoordinateWidget) MouseDown(e *desktop.MouseEvent) {
	if e.Button == desktop.MouseButtonSecondary {
		cw.TappedSecondary(&fyne.PointEvent{Position: e.Position})
	}
}

func (cw *CoordinateWidget) calculateGridSize(viewWidth float64) float64 {
	// 根据可视区域宽度自动调整网格密度
	baseSize := 100.0
	minSpacing := 50.0 // 最小像素间距

	gridSize := baseSize
	for viewWidth*gridSize/cw.scale < minSpacing {
		gridSize *= 2
	}
	for viewWidth*gridSize/cw.scale > minSpacing*4 {
		gridSize /= 2
	}

	return math.Max(gridSize, 10) // 保证最小网格尺寸
}

func drawVerticalLine(img *image.RGBA, x int, c color.Color) {
	rect := image.Rect(x, 0, x+1, img.Bounds().Dy())
	draw.Draw(img, rect, image.NewUniform(c), image.Point{}, draw.Over)
}

func drawHorizontalLine(img *image.RGBA, y int, c color.Color) {
	rect := image.Rect(0, y, img.Bounds().Dx(), y+1)
	draw.Draw(img, rect, image.NewUniform(c), image.Point{}, draw.Over)
}

func drawAxis(img *image.RGBA, offsetX, offsetY float64, w, h int, scale float64) {
	// 绘制X轴
	if y := int(offsetY); y >= 0 && y < h {
		drawHorizontalLine(img, y, color.Black)
	}

	// 绘制Y轴
	if x := int(offsetX); x >= 0 && x < w {
		drawVerticalLine(img, x, color.Black)
	}
}

func (cw *CoordinateWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(cw.raster)
}

// 处理鼠标拖动
func (cw *CoordinateWidget) Dragged(e *fyne.DragEvent) {
	cw.offsetX += float64(e.Dragged.DX)
	cw.offsetY += float64(e.Dragged.DY)
	cw.raster.Refresh()
}

func (cw *CoordinateWidget) DragEnd() {}

// 处理鼠标滚轮缩放
func (cw *CoordinateWidget) Scrolled(e *fyne.ScrollEvent) {
	delta := e.Scrolled.DY
	oldScale := cw.scale

	if delta > 0 {
		cw.scale *= 1.1
	} else {
		cw.scale *= 0.9
	}

	// 保持鼠标位置对应的虚拟坐标不变
	mouseX := float64(e.Position.X)
	mouseY := float64(e.Position.Y)
	virtualX := (mouseX - cw.offsetX) / oldScale
	virtualY := (mouseY - cw.offsetY) / oldScale

	cw.offsetX = mouseX - virtualX*cw.scale
	cw.offsetY = mouseY - virtualY*cw.scale

	cw.raster.Refresh()
}

// 启用桌面扩展实现更精确的输入处理
func (cw *CoordinateWidget) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	secret := os.Getenv("ACCESS_SECRET")
	fmt.Println(secret)
	a := app.New()
	w := a.NewWindow("超大坐标系")

	go func() {
		time.Sleep(time.Second * 2)
		dialog.ShowInformation(secret, secret, w)
	}()

	coord := NewCoordinateWidget()
	w.SetContent(coord)

	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}
