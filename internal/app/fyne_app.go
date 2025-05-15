package app

import (
	"encoding/json"
	"fyne.io/fyne/v2"
	fyneApp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/EnderCHX/DSMS-go/internal/auth"
)

var messageList = binding.NewBytesList()

var list = widget.NewList(
	func() int {
		return messageList.Length()
	},
	func() fyne.CanvasObject {
		return widget.NewLabel("")
	},
	func(i widget.ListItemID, o fyne.CanvasObject) {
		strBytes, _ := messageList.GetValue(i)
		o.(*widget.Label).SetText(string(strBytes))
	})

func Run() {
	a := fyneApp.New()
	w := a.NewWindow("DSMS-go")
	icon, err := fileFS.ReadFile("resources/img/bronya.jpg")
	if err != nil {
		logger.Error("Read icon error: " + err.Error())
	}
	w.SetIcon(fyne.NewStaticResource("bronya", icon))
	w.Resize(fyne.NewSize(800, 0))
	w.SetCloseIntercept(func() {
		client.Close()
		a.Quit()
	})

	messageList.Append([]byte("hello world"))

	inputServerIp := widget.NewEntry()
	inputServerIp.SetPlaceHolder("输入服务器IP")
	inputServerIp.SetText("127.0.0.1")

	inputServerPort := widget.NewEntry()
	inputServerPort.SetPlaceHolder("输入服务器端口")
	inputServerPort.SetText("1314")

	inputPublish := widget.NewEntry()
	inputPublish.SetPlaceHolder("发布事件")

	inputMsg := widget.NewEntry()
	inputMsg.SetPlaceHolder("输入消息内容")

	inputSubscribe := widget.NewEntry()
	inputSubscribe.SetPlaceHolder("订阅事件")

	inputContent := container.NewVBox(
		inputPublish,
		inputMsg,
		widget.NewButton("发布事件", func() {
			logger.Info("发布事件: " + inputPublish.Text)
			send <- func() []byte {
				msg_, _ := json.Marshal(&Msg{
					Option: "publish",
					Data: func() json.RawMessage {
						data, _ := json.Marshal(map[string]string{
							"event": inputPublish.Text,
							"data":  inputMsg.Text,
						})
						return data
					}(),
				})
				return msg_
			}()
		}),
		inputSubscribe,
		widget.NewButton("订阅事件", func() {
			logger.Info("订阅事件: " + inputSubscribe.Text)
			send <- func() []byte {
				msg_, _ := json.Marshal(&Msg{
					Option: "subscribe",
					Data: func() json.RawMessage {
						data, _ := json.Marshal(map[string]string{
							"event": inputSubscribe.Text,
						})
						return data
					}(),
				})
				return msg_
			}()
		}),
	)

	connectedContent := container.NewGridWithColumns(1,
		inputContent,
		list,
	)
	inputUsername := widget.NewEntry()
	inputUsername.SetPlaceHolder("输入用户名")
	inputPassword := widget.NewEntry()
	inputPassword.SetPlaceHolder("输入密码")
	inputPassword.Password = true

	connContent := container.NewVBox(
		inputServerIp,
		inputServerPort,
		inputUsername,
		inputPassword,
		widget.NewButton("连接", func() {
			logger.Info("连接到: " + inputServerIp.Text + ":" + inputServerPort.Text)
			err := connectServer(inputServerIp.Text, inputServerPort.Text)
			if err != nil {
				logger.Error("连接错误: ")
				return
			}
			w.SetContent(connectedContent)
			send <- func() []byte {
				username := inputUsername.Text
				password := inputPassword.Text
				_, access_token, err := auth.Login(username, password)
				if err != nil {
					logger.Error("登录错误: " + err.Error())
					dialog.NewInformation("登录错误", err.Error(), w)
					return []byte{}
				}
				msg_, _ := json.Marshal(&Msg{
					Option: "login",
					Data: func() json.RawMessage {
						data, _ := json.Marshal(map[string]string{
							"access_token": access_token,
						})
						return data
					}(),
				})
				return msg_
			}()
		}),
		list,
	)

	mainContent := container.NewGridWithColumns(1,
		connContent,
	)

	//go func() {
	//	for {
	//		fyne.Do(func() {
	//			list.ScrollToBottom()
	//		})
	//		time.Sleep(1 * time.Second)
	//		messageList.Append([]byte(time.Now().Format("2006-01-02 15:04:05")))
	//	}
	//}()

	w.SetContent(mainContent)
	w.ShowAndRun()
}
