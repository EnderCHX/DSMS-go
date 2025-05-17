package simulation

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	app2 "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/EnderCHX/DSMS-go/utils/log"
	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"go.uber.org/zap"
	"image/color"
	"os"
	"time"
)

var clientUsername string

func RunClient() {
	pid := fmt.Sprintf("%v", os.Getpid())
	logger = log.NewLogger("[simulation_client_"+pid+"]", "log/simulation_center_"+pid+".log", "debug")

	a := app2.New()
	w := a.NewWindow("仿真客户端")
	w.Resize(fyne.NewSize(400, 100))
	w.SetFixedSize(true)

	var connectContainer *fyne.Container
	var mainContainer *fyne.Container

	addr := &widget.Entry{
		Text:        "127.0.0.1",
		PlaceHolder: "地址",
	}
	port := &widget.Entry{
		Text:        "1314",
		PlaceHolder: "端口",
	}
	username := &widget.Entry{
		Text:        "test",
		PlaceHolder: "账号",
	}
	password := &widget.Entry{
		Text:        "test",
		PlaceHolder: "密码",
		Password:    true,
	}

	connectContainer = container.NewVBox(
		container.NewGridWithRows(1,
			&widget.Label{
				Text: "地址端口:",
			},
			addr,
			port,
		),
		container.NewGridWithRows(1,
			&widget.Label{
				Text: "账号密码:",
			},
			username,
			password,
		),
		&widget.Button{
			Text: "连接",
			OnTapped: func() {
				logger.Debug(fmt.Sprintf("addr: %v:%v user: %v pass: %v", addr.Text, port.Text, username.Text, password.Text))
				err := connectMsgHub(addr.Text, port.Text, username.Text, password.Text)
				if err != nil {
					logger.Error("dstp failed", zap.Error(err))
					return
				}
				clientUsername = username.Text
				clientsPoint.Store(username.Text, Point{
					X:        0,
					Y:        0,
					Username: username.Text,
				})
				clientsSet.Store(username.Text, struct{}{})

				vectorClockAdd(username.Text)
				msg, _ := sonic.Marshal(map[string]any{
					"option": "publish",
					"data": map[string]any{
						"event": "simulation/join",
						"data": map[string]any{
							"vector_clock": vectorClockToMap(&vectorClock),
						},
					},
				})
				dstpConn.Send(msg, false)

				point, _ := clientsPoint.Load(clientUsername)

				vectorClockAdd(clientUsername)
				msg, _ = sonic.Marshal(map[string]any{
					"option": "publish",
					"data": map[string]any{
						"event": "simulation/client/" + clientUsername,
						"data": map[string]any{
							"vector_clock": vectorClockToMap(&vectorClock),
							"point": map[string]any{
								"x": point.(Point).X,
								"y": point.(Point).Y,
							},
						},
					},
				})
				dstpConn.Send(msg, false)

				vectorClockAdd(clientUsername)
				msg, _ = sonic.Marshal(map[string]any{
					"option": "subscribe",
					"data": map[string]any{
						"event": "simulation/setting/tick",
						"data": map[string]any{
							"vector_clock": vectorClockToMap(&vectorClock),
						},
					},
				})
				dstpConn.Send(msg, false)

				vectorClockAdd(clientUsername)
				msg, _ = sonic.Marshal(map[string]any{
					"option": "subscribe",
					"data": map[string]any{
						"event": "simulation/setting/point",
						"data": map[string]any{
							"vector_clock": vectorClockToMap(&vectorClock),
						},
					},
				})
				dstpConn.Send(msg, false)

				vectorClockAdd(clientUsername)
				msg, _ = sonic.Marshal(map[string]any{
					"option": "subscribe",
					"data": map[string]any{
						"event": "simulation/setting/remove_point",
						"data": map[string]any{
							"vector_clock": vectorClockToMap(&vectorClock),
						},
					},
				})
				dstpConn.Send(msg, false)

				go clientStep()
				go handleMsgClient()

				w.SetContent(mainContainer)
				w.Resize(fyne.NewSize(800, 600))
				w.SetFixedSize(false)
			},
		},
	)

	mainContainer = container.NewGridWithColumns(1,
		bigMap,
		container.NewVBox(
			&widget.Button{
				Text: "退出",
				OnTapped: func() {
					msg, _ := sonic.Marshal(map[string]any{
						"option": "publish",
						"data": map[string]any{
							"event": "simulation/quit",
						},
					})
					dstpConn.Send(msg, false)
					cancel()
					disconnectMsgHub()
					clientsPoint.Range(func(key, value any) bool {
						bigMap.RemovePoint(key.(string))
						return true
					})
					clientsSet.Clear()
					clientsPoint.Clear()
					vectorClock.Clear()
					w.SetFixedSize(true)
					w.SetContent(connectContainer)
					w.Resize(fyne.NewSize(400, 100))
					ctx, cancel = context.WithCancel(context.Background())
				},
			},
			container.NewGridWithRows(4,
				&widget.Button{
					Text: "上",
					OnTapped: func() {
						clientMove(0, -50)
					},
				},
				&widget.Button{
					Text: "下",
					OnTapped: func() {
						clientMove(0, 50)
					},
				},
				&widget.Button{
					Text: "左",
					OnTapped: func() {
						clientMove(-50, 0)
					},
				},
				&widget.Button{
					Text: "右",
					OnTapped: func() {
						clientMove(50, 0)
					},
				},
			),
		),
	)

	w.SetContent(connectContainer)
	w.ShowAndRun()
}

func clientStep() {
	for {
		select {
		case <-ticker.C:
			vectorClockAdd(clientUsername)
			point, _ := clientsPoint.Load(clientUsername)
			msg, _ := sonic.Marshal(map[string]any{
				"option": "publish",
				"data": map[string]any{
					"event": "simulation/client/" + clientUsername,
					"data": map[string]any{
						"vector_clock": vectorClockToMap(&vectorClock),
						"point": map[string]any{
							"x": point.(Point).X,
							"y": point.(Point).Y,
						},
					},
				},
			})
			dstpConn.Send(msg, false)

			fyne.Do(func() {
				bigMap.points = clientsPoint
				bigMap.raster.Refresh()
				clientsCount.Set(fmt.Sprintf("%v", getSyncMapLen(&clientsSet)))
				vectorClockBinding.Set(func() []string {

					list := make([]string, 0)
					vectorClock.Range(func(k, v any) bool {
						list = append(list, fmt.Sprintf("%v: %v", k, v))
						return true
					})
					return list
				}())
			})
		case <-ctx.Done():
			return
		}
	}
}

func handleMsgClient() {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			data_, type_, err := dstpConn.Receive()
			if err != nil {
				logger.Error("receive failed", zap.Error(err))
				return
			}
			if type_ != 1 {
				logger.Warn("invalid message type")
				continue
			}
			option, _ := sonic.Get(data_, "option")
			optionStr, _ := option.String()

			if optionStr == "ping" {
				go func() {
					msg, _ := sonic.Marshal(map[string]any{
						"option": "pong",
					})
					dstpConn.Send(msg, false)
				}()
				continue
			}
			logger.Debug(string(data_))

			data, _ := sonic.Get(data_, "data")
			event, _ := data.Get("event").String()
			go handleEventClient(event, data)
		}
	}
}

func handleEventClient(event string, data ast.Node) {
	recVC, _ := data.Get("data").Get("vector_clock").MarshalJSON()
	recVectorClock := make(map[string]uint64)
	sonic.Unmarshal(recVC, &recVectorClock)
	for k, v := range recVectorClock {
		vv, ok := vectorClock.Load(k)
		if ok {
			if v > vv.(uint64) {
				vectorClock.Store(k, v)
			}
		} else {
			vectorClock.Store(k, v)
		}
	}

	switch event {
	case "simulation/setting/tick":
		tick_, _ := data.Get("data").Get("tick").Int64()
		fmt.Println(tick_)
		tick = time.Duration(tick_)
		ticker = time.NewTicker(time.Millisecond * (1000 / tick))
		vectorClockAdd(clientUsername)
	case "simulation/setting/point":
		point_x, _ := data.Get("data").Get("point").Get("X").Float64()
		point_y, _ := data.Get("data").Get("point").Get("Y").Float64()
		point_username, _ := data.Get("data").Get("point").Get("Username").String()
		point_color_r, _ := data.Get("data").Get("point").Get("Color").Get("R").Int64()
		point_color_g, _ := data.Get("data").Get("point").Get("Color").Get("G").Int64()
		point_color_b, _ := data.Get("data").Get("point").Get("Color").Get("B").Int64()
		point := Point{
			X: point_x,
			Y: point_y,
			Color: color.RGBA{
				R: uint8(point_color_r),
				G: uint8(point_color_g),
				B: uint8(point_color_b),
				A: 255,
			},
			Radius:   3,
			Username: point_username,
		}
		clientsPoint.Store(point_username, point)
		clientsSet.Store(point_username, struct{}{})
		vectorClockAdd(clientUsername)
	case "simulation/setting/remove_point":
		point_name, _ := data.Get("data").Get("point").String()
		clientsPoint.Delete(point_name)
		clientsSet.Delete(point_name)
		vectorClockAdd(clientUsername)
	default:
		logger.Warn("unknown event", zap.String("event", event))
	}
}

func clientMove(move_x, move_y float64) {
	vectorClockAdd(clientUsername)
	point, _ := clientsPoint.Load(clientUsername)
	msg, _ := sonic.Marshal(map[string]any{
		"option": "publish",
		"data": map[string]any{
			"event": "simulation/client/" + clientUsername,
			"data": map[string]any{
				"vector_clock": vectorClockToMap(&vectorClock),
				"point": map[string]any{
					"x": point.(Point).X + move_x,
					"y": point.(Point).Y + move_y,
				},
			},
		},
	})
	dstpConn.Send(msg, false)
}
