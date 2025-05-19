package simulation

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/EnderCHX/DSMS-go/internal/auth"
	"github.com/EnderCHX/DSMS-go/internal/dstp"
	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"math"
	"net"
	"sync"
	"time"
)

var (
	dstpConn           *dstp.Conn
	vectorClock                      = sync.Map{}
	tick               time.Duration = 20
	tickBinding                      = binding.NewString()
	ticker                           = time.NewTicker(time.Millisecond * (1000 / tick))
	bigMap                           = NewCoordinateWidget()
	logger             *zap.Logger
	clientsPoint       = sync.Map{}
	clientsSet         = sync.Map{}
	clientsCount       = binding.NewString()
	vectorClockBinding = binding.NewStringList()
	chatListBinding    = binding.NewStringList()
	ctx, cancel        = context.WithCancel(context.Background())
)

type Point struct {
	X, Y     float64
	Username string
	Color    color.Color
	Radius   float64
}

type CoordinateWidget struct {
	widget.BaseWidget
	offsetX, offsetY float64
	scale            float64
	raster           *canvas.Raster
	points           sync.Map
	selectedPoint    int               // 当前选中的点索引
	popUpMenu        *widget.PopUpMenu // 右键菜单
}

func NewCoordinateWidget() *CoordinateWidget {
	cw := &CoordinateWidget{
		scale:  1.0,
		points: sync.Map{},
	}
	cw.raster = canvas.NewRaster(cw.draw)
	cw.ExtendBaseWidget(cw)
	return cw
}

// 添加点的方法
func (cw *CoordinateWidget) AddPoint(point Point) {
	cw.points.Store(point.Username, point)
	cw.raster.Refresh()
}

// 删除点的方法
func (cw *CoordinateWidget) RemovePoint(username string) {
	cw.points.Delete(username)
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
	cw.points.Range(func(key, value interface{}) bool {
		p := value.(Point)
		screenX := p.X*cw.scale + cw.offsetX
		screenY := p.Y*cw.scale + cw.offsetY
		if p.Color == nil {
			p.Color = color.Black
		}
		if p.Radius == 0 {
			p.Radius = 5
		}
		drawCircle(img, int(screenX), int(screenY), int(p.Radius), p.Color)
		drawText(img, int(screenX-2*cw.scale), int(screenY-2*cw.scale), p.Username, p.Color)
		return true
	})

	//for _, p := range cw.points {
	//	screenX := p.X*cw.scale + cw.offsetX
	//	screenY := p.Y*cw.scale + cw.offsetY
	//	drawCircle(img, int(screenX), int(screenY), int(p.Radius), p.Color)
	//	drawText(img, int(screenX-2*cw.scale), int(screenY-2*cw.scale), p.Username, p.Color)
	//}

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

func drawText(img *image.RGBA, x, y int, text string, c color.Color) {
	col := color.RGBAModel.Convert(c).(color.RGBA)
	point := fixed.Point26_6{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13, // 使用内置基础字体
		Dot:  point,
	}
	d.DrawString(text)
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

func connectMsgHub(addr, port, username, password string) error {
	_, access_token, err := auth.Login(username, password)
	if err != nil {
		logger.Error("login failed", zap.Error(err))
		return err
	}

	c, err := net.Dial("tcp", fmt.Sprintf("%v:%v", addr, port))
	if err != nil {
		logger.Error("dstp failed", zap.Error(err))
		return err
	}

	dstpConn = dstp.NewConn(&c)

	loginByte, _ := sonic.Marshal(map[string]any{
		"option": "login",
		"data": map[string]any{
			"access_token": access_token,
		},
	})

	dstpConn.Send(loginByte, true)

	return nil
}

func disconnectMsgHub() {
	dstpConn.Close()
	dstpConn = nil
	vectorClock.Clear()
	logger.Debug("disconnect")
}

func getSyncMapLen(m *sync.Map) int {
	var count int
	m.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func vectorClockAdd(id string) {
	v, ok := vectorClock.Load(id)
	if ok {
		vectorClock.Store(id, v.(uint64)+1)
	} else {
		vectorClock.Store(id, uint64(1))
	}
}

func vectorClockToMap(m *sync.Map) map[string]uint64 {
	vmap := make(map[string]uint64)
	m.Range(func(key, value interface{}) bool {
		vmap[key.(string)] = value.(uint64)
		return true
	})
	return vmap
}

func vectorIsConcurrent(v1, v2 map[string]uint64) bool {
	var greater, less bool = false, false
	if len(v1) > len(v2) {
		for k1, v1v := range v1 {
			v2v, ok := v2[k1]
			if ok {
				if v1v > v2v {
					greater = true
				} else if v1v < v2v {
					less = true
				}
			} else {
				greater = true
			}
		}
	} else {
		for k2, v2v := range v2 {
			v1v, ok := v1[k2]
			if ok {
				if v1v > v2v {
					greater = true
				} else if v1v < v2v {
					less = true
				}
			} else {
				less = true
			}
		}
	}

	if greater && less {
		return true
	} else {
		return false
	}
}

type ControlPad struct {
	widget.BaseWidget
	pointUsername string
	OnKeyDown     func(key *fyne.KeyEvent)
	OnKeyUp       func(key *fyne.KeyEvent)
}

func (c *ControlPad) FocusGained() {
}

func (c *ControlPad) FocusLost() {
}

func (c *ControlPad) TypedRune(r rune) {

}

func (c *ControlPad) TypedKey(event *fyne.KeyEvent) {

}
