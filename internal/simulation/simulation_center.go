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
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var centerUsername string

func RunCenter() {
	logger = log.NewLogger("[simulation_center]", "log/simulation_center.log", "debug")

	a := app2.New()
	w := a.NewWindow("仿真中心")
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
		Text:        "center",
		PlaceHolder: "账号",
	}
	password := &widget.Entry{
		Text:        "center",
		PlaceHolder: "密码",
		Password:    true,
	}

	tickInput := &widget.Entry{
		Text:        "20",
		PlaceHolder: "步长",
		Validator: func(s string) error {
			if _, err := strconv.Atoi(s); err != nil {
				return fmt.Errorf("请输入数字")
			}
			return nil
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

	mainContainer = container.NewGridWithColumns(2,
		bigMap,
		container.NewVBox(
			container.New(
				layout.NewFormLayout(),
				&widget.Label{
					Text: "当前节点数量",
				},
				widget.NewLabelWithData(clientsCount),
				tickInput,
				&widget.Button{
					Text: "修改步长(tick/s)",
					OnTapped: func() {
						tick_, err := strconv.Atoi(tickInput.Text)
						if err != nil {
							dialog.NewInformation("错误", "请输入数字", w).Show()
							logger.Error(fmt.Sprintf("请输入数字: %v", err))
							return
						}
						tick = time.Duration(tick_)
						ticker = time.NewTicker(time.Millisecond * (1000 / tick))

						vectorClockAdd(centerUsername)
						msg, _ := sonic.Marshal(map[string]any{
							"option": "publish",
							"data": map[string]any{
								"channel": "simulation/setting/tick",
								"data": map[string]any{
									"tick":         tick_,
									"vector_clock": vectorClockToMap(&vectorClock),
								},
							},
						})
						dstpConn.Send(msg, false)
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

						vectorClockAdd(centerUsername)
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
					cancel()
					disconnectMsgHub()
					clientsPoint.Range(func(key, value any) bool {
						bigMap.RemovePoint(key.(string))
						return true
					})
					clientsSet.Clear()
					clientsPoint.Clear()
					vectorClock.Clear()
					w.SetContent(connectContainer)
					w.SetFixedSize(true)
					w.Resize(fyne.NewSize(400, 100))
					ctx, cancel = context.WithCancel(context.Background())
				},
			},
		),
		widget.NewCard("向量时间戳", "", vectorClockList),
		widget.NewCard("消息记录", "", chatList),
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
				centerUsername = username.Text

				vectorClockAdd(username.Text)
				msg, _ := sonic.Marshal(map[string]any{
					"option": "subscribe",
					"data": map[string]any{
						"channel": "simulation/join",
						"data": map[string]any{
							"vector_clock": vectorClockToMap(&vectorClock),
						},
					},
				})
				dstpConn.Send(msg, false)

				vectorClockAdd(username.Text)
				msg, _ = sonic.Marshal(map[string]any{
					"option": "subscribe",
					"data": map[string]any{
						"channel": "simulation/quit",
						"data": map[string]any{
							"vector_clock": vectorClockToMap(&vectorClock),
						},
					},
				})
				dstpConn.Send(msg, false)

				vectorClockAdd(username.Text)
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

				vectorClockAdd(username.Text)
				go step()

				vectorClockAdd(username.Text)
				go handleMsgCenter()

				w.SetContent(mainContainer)
				w.Resize(fyne.NewSize(800, 600))
				w.SetFixedSize(false)
			},
		},
	)

	w.SetContent(connectContainer)
	w.ShowAndRun()
}

func step() {
	for {
		select {
		case <-ticker.C:
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

func handleMsgCenter() {
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
			logger.Debug(fmt.Sprintf("receive: %v", string(data_)))
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

			data, _ := sonic.Get(data_, "data")
			channel, _ := data.Get("channel").String()
			go handleEventCenter(channel, data)
		}
	}
}

func handleEventCenter(channel string, data ast.Node) {
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
	case "simulation/join":
		username, _ := data.Get("from_user").String()
		clientsSet.Store(username, struct{}{})

		vectorClockAdd(centerUsername)
		msg, _ := sonic.Marshal(map[string]any{
			"option": "subscribe",
			"data": map[string]any{
				"channel": "simulation/client/" + username,
				"data": map[string]any{
					"vector_clock": vectorClockToMap(&vectorClock),
				},
			},
		})
		dstpConn.Send(msg, false)

		msg, _ = sonic.Marshal(map[string]any{
			"option": "publish",
			"data": map[string]any{
				"channel": "simulation/setting/tick",
				"data": map[string]any{
					"tick":         int64(tick),
					"vector_clock": vectorClockToMap(&vectorClock),
				},
			},
		})
		dstpConn.Send(msg, false)
	case "simulation/quit":
		username, _ := data.Get("from_user").String()
		clientsSet.Delete(username)
		clientsPoint.Delete(username)
		vectorClock.Delete(username)

		vectorClockAdd(centerUsername)
		msg, _ := sonic.Marshal(map[string]any{
			"option": "unsubscribe",
			"data": map[string]any{
				"channel": "simulation/client/" + username,
				"data": map[string]any{
					"vector_clock": vectorClockToMap(&vectorClock),
				},
			},
		})
		dstpConn.Send(msg, false)

		vectorClockAdd(centerUsername)
		msg, _ = sonic.Marshal(map[string]any{
			"option": "publish",
			"data": map[string]any{
				"channel": "simulation/setting/remove_point",
				"data": map[string]any{
					"vector_clock": vectorClockToMap(&vectorClock),
					"point":        username,
				},
			},
		})
		dstpConn.Send(msg, false)
	default:
		if match, err := regexp.MatchString(`^simulation/client/(.+)$`, channel); err == nil && match {
			username := channel[len("simulation/client/"):]
			if _, ok := clientsSet.Load(username); ok {
				point_x, _ := data.Get("data").Get("point").Get("x").Float64()
				point_y, _ := data.Get("data").Get("point").Get("y").Float64()
				if v, okk := clientsPoint.Load(username); okk {
					p := v.(Point)
					p.X = point_x
					p.Y = point_y
					clientsPoint.Store(username, p)
				} else {
					randColor := color.RGBA{
						R: uint8(rand.Intn(255)),
						G: uint8(rand.Intn(255)),
						B: uint8(rand.Intn(255)),
						A: 255,
					}
					clientsPoint.Store(username, Point{
						X:        point_x,
						Y:        point_y,
						Color:    randColor,
						Radius:   5,
						Username: username,
					})
				}
				vectorClockAdd(centerUsername)

				point, _ := clientsPoint.Load(username)

				vectorClockAdd(centerUsername)
				msg, _ := sonic.Marshal(map[string]any{
					"option": "publish",
					"data": map[string]any{
						"channel": "simulation/setting/point",
						"data": map[string]any{
							"vector_clock": vectorClockToMap(&vectorClock),
							"point":        point.(Point),
						},
					},
				})
				dstpConn.Send(msg, false)

			}
			return
		}
		logger.Warn("unknown channel", zap.String("channel", channel))
	}
}
