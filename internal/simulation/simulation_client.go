package simulation

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	app2 "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/EnderCHX/DSMS-go/utils/log"
	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"go.uber.org/zap"
	"image/color"
	"os"
	"strconv"
	"strings"
	"time"
)

var clientUsername string
var clientStepLen float64 = 0.5

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

	clientStepLenInput := &widget.Entry{
		Text:        fmt.Sprintf("%v", clientStepLen),
		PlaceHolder: "每次移动距离",
		Validator: func(s string) error {
			_, err := strconv.ParseFloat(s, 64)
			return err
		},
	}

	vectorClockList := widget.NewListWithData(
		vectorClockBinding,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		},
	)
	chatInput := &widget.Entry{
		Text:        "",
		PlaceHolder: "消息",
		MultiLine:   true,
	}
	chatList := widget.NewListWithData(
		chatListBinding,
		func() fyne.CanvasObject {
			return widget.NewEntry()
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			text, _ := i.(binding.String).Get()
			spit := strings.Split(text, "\n")
			spitNum := len(spit)
			o.(*widget.Entry).MultiLine = true
			o.(*widget.Entry).SetMinRowsVisible(spitNum)
			o.(*widget.Entry).Disable()
			o.(*widget.Entry).Bind(i.(binding.String))
		},
	)

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
						"channel": "simulation/join",
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
						"channel": "simulation/client/" + clientUsername,
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
						"channel": "simulation/setting/tick",
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
						"channel": "simulation/setting/point",
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
						"channel": "simulation/setting/remove_point",
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
						"channel": "simulation/chat",
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

	mainContainer = container.NewGridWithColumns(2,
		bigMap,
		container.NewVBox(
			container.New(
				layout.NewFormLayout(),
				&widget.Label{
					Text: "当前节点数量:",
				},
				widget.NewLabelWithData(clientsCount),
				&widget.Label{
					Text: "当前步长:",
				},
				widget.NewLabelWithData(tickBinding),
				clientStepLenInput,
				&widget.Button{
					Text: "设置每次移动距离",
					OnTapped: func() {
						clientStepLen__, err := strconv.ParseFloat(clientStepLenInput.Text, 64)
						if err != nil {
							dialog.NewInformation("错误", "请输入数字", w).Show()
							return
						}
						clientStepLen = clientStepLen__
					},
				},
				chatInput,
				&widget.Button{
					Text: "发送",
					OnTapped: func() {
						if chatInput.Text == "" {
							dialog.NewInformation("错误", "请输入内容", w).Show()
							return
						}

						vectorClockAdd(clientUsername)
						msg, _ := sonic.Marshal(map[string]any{
							"option": "publish",
							"data": map[string]any{
								"channel": "simulation/chat",
								"data": map[string]any{
									"chat":         chatInput.Text,
									"vector_clock": vectorClockToMap(&vectorClock),
								},
							},
						})
						dstpConn.Send(msg, false)

						chatInput.SetText("")
					},
				},
			),
			&widget.Button{
				Text: "退出",
				OnTapped: func() {
					msg, _ := sonic.Marshal(map[string]any{
						"option": "publish",
						"data": map[string]any{
							"channel": "simulation/quit",
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
		),
		widget.NewCard("向量时间戳", "", vectorClockList),
		widget.NewCard("消息记录", "", chatList),
	)

	w.Canvas().SetOnTypedKey(func(channel *fyne.KeyEvent) {
		if channel.Name == fyne.KeyUp || channel.Name == fyne.KeyW {
			clientMove(0, -clientStepLen)
		} else if channel.Name == fyne.KeyDown || channel.Name == fyne.KeyS {
			clientMove(0, clientStepLen)
		} else if channel.Name == fyne.KeyLeft || channel.Name == fyne.KeyA {
			clientMove(-clientStepLen, 0)
		} else if channel.Name == fyne.KeyRight || channel.Name == fyne.KeyD {
			clientMove(clientStepLen, 0)
		}
	})
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
					"channel": "simulation/client/" + clientUsername,
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
				tickBinding.Set(fmt.Sprintf("%d tick/s", tick))
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
			channel, _ := data.Get("channel").String()
			go handleEventClient(channel, data)
		}
	}
}

func handleEventClient(channel string, data ast.Node) {
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

	switch channel {
	case "simulation/chat":
		from_user, _ := data.Get("from_user").String()
		chat, _ := data.Get("data").Get("chat").String()
		date := time.Now().Format("[2006-01-02 15:04:05]")
		chatListBinding.Append(fmt.Sprintf("%s%s: %s", date, from_user, chat))
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
			Radius:   5,
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
		logger.Warn("unknown channel", zap.String("channel", channel))
	}
}

func clientMove(move_x, move_y float64) {
	vectorClockAdd(clientUsername)
	point, _ := clientsPoint.Load(clientUsername)
	msg, _ := sonic.Marshal(map[string]any{
		"option": "publish",
		"data": map[string]any{
			"channel": "simulation/client/" + clientUsername,
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
