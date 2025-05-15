package simulation_center

import (
	"fmt"
	"fyne.io/fyne/v2"
	app2 "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/EnderCHX/DSMS-go/internal/auth"
	"github.com/EnderCHX/DSMS-go/internal/connect"
	"github.com/EnderCHX/DSMS-go/utils/log"
	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"io"
	"net"
)

const (
	eventName = "simulation"
)

var nodeName = "center"

var vectorClock = map[string]uint64{
	nodeName: 0,
}

var (
	center *connect.Conn
	writer io.WriteCloser
	reader io.Reader
)
var logger *zap.Logger

func Run() {
	//addr := os.Getenv("MESSAGE_HUB_ADDR_PORT")
	//username := os.Getenv("SIMULATION_CENTER_USERNAME")
	//password := os.Getenv("SIMULATION_CENTER_PASSWORD")
	//var addr, username, password string
	logger = log.NewLogger("[simulation_center]", "log/simulation_center.log", "debug")

	a := app2.New()
	w := a.NewWindow("仿真中心")
	w.Resize(fyne.NewSize(400, 100))
	w.SetFixedSize(true)

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

	mainContainer := container.NewVBox(
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
					logger.Error("connect failed", zap.Error(err))
					return
				}
				w.SetContent(connectContainer)
			},
		},
	)

	w.SetContent(mainContainer)
	w.ShowAndRun()
}

func connectMsgHub(addr, port, username, password string) error {
	_, access_token, err := auth.Login(username, password)
	if err != nil {
		logger.Error("login failed", zap.Error(err))
		return err
	}

	c, err := net.Dial("tcp", fmt.Sprintf("%v:%v", addr, port))
	if err != nil {
		logger.Error("connect failed", zap.Error(err))
		return err
	}

	center = connect.NewConn(&c)

	writer, err = center.Send()
	if err != nil {
		logger.Error("send failed", zap.Error(err))
		return err
	}

	reader, err = center.Receive()
	if err != nil {
		logger.Error("receive failed", zap.Error(err))
		return err
	}

	loginByte, _ := sonic.Marshal(map[string]any{
		"option": "login",
		"data": map[string]any{
			"access_token": access_token,
		},
	})
	writer.Write(loginByte)

	return nil
}

func disconnectMsgHub() {
	center.Close()
	center = nil
	writer = nil
	reader = nil
	vectorClock = map[string]uint64{
		nodeName: 0,
	}
	logger.Debug("disconnect")
}
