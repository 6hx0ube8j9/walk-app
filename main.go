package main

import (
	"os"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

//go:generate go build -ldflags="-H windowsgui" -o simple-app.exe main.go
func init() {}

// 捕获错误并弹窗提示，防止程序静默闪退
func handleError(err error) {
	if err != nil {
		walk.MsgBox(nil, "错误", err.Error(), walk.MsgBoxIconError)
		os.Exit(1)
	}
}

func main() {
	// 1. 必须先初始化 App（tailscale/walk 的核心生命周期入口）
	app, err := walk.InitApp()
	handleError(err)
	defer app.Exit(0)

	var mw *walk.MainWindow

	// 2. 创建主窗口
	err = (MainWindow{
		AssignTo: &mw,
		Title:    "简易托盘面板",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			PushButton{
				Text:      "隐藏面板",
				OnClicked: func() { mw.Hide() },
			},
		},
	}).Create()
	handleError(err)

	// 3. 拦截右上角 X 或 Alt+F4 关闭事件，改为隐藏而不是销毁
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true // 取消默认退出
		mw.Hide()         // 仅隐藏窗口，托盘和后台保持存活
	})

	// 4. 启动时默认隐藏面板，只留托盘
	mw.Hide()

	// 5. 初始化系统托盘
	ni, err := walk.NewNotifyIcon()
	handleError(err)
	defer ni.Dispose()

	err = ni.SetToolTip("点击打开面板")
	handleError(err)

	err = ni.SetIcon(walk.IconInformation())
	handleError(err)

	// 左键点击托盘：显示/隐藏面板
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if mw.Visible() {
				mw.Hide()
			} else {
				mw.Show()
				mw.SetFocus()
			}
		}
	})

	// 右键菜单：真正退出程序
	exitAction := walk.NewAction()
	exitAction.SetText("退出")
	exitAction.Triggered().Attach(func() {
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)

	err = ni.SetVisible(true)
	handleError(err)

	// 6. 运行主消息循环
	app.Run()
}
