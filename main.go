package main

import (
	"os"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

//go:generate go build -ldflags="-H windowsgui" -o simple-app.exe main.go
func init() {}

func handleError(err error) {
	if err != nil {
		walk.MsgBox(nil, "错误", err.Error(), walk.MsgBoxIconError)
		os.Exit(1)
	}
}

func main() {
	app, err := walk.InitApp()
	handleError(err)
	defer app.Exit(0)

	var mw *walk.MainWindow
	var ni *walk.NotifyIcon

	// 创建主窗口
	err = (MainWindow{
		AssignTo: &mw,
		Title:    "简易窗口",
		MinSize:  Size{Width: 300, Height: 150},
		Layout:   VBox{},
		
		// 【关键修复】拦截右上角 X 关闭按钮，改为隐藏而不是销毁
		OnClosing: func(canceled *bool, reason walk.CloseReason) {
			*canceled = true // 取消默认退出
			mw.Hide()         // 仅隐藏窗口，托盘存活
		},

		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "确定",
						OnClicked: func() {
							mw.Hide()
						},
					},
					PushButton{
						Text: "取消",
						OnClicked: func() {
							mw.Hide()
						},
					},
				},
			},
		},
	}).Create()
	handleError(err)

	// 启动时隐藏主窗口
	mw.Hide()

	// 创建托盘（需传入主窗口句柄作为宿主）
	ni, err = walk.NewNotifyIcon(mw)
	handleError(err)
	defer ni.Dispose()

	err = ni.SetToolTip("点击打开简易程序")
	handleError(err)

	icon := walk.IconInformation()
	err = ni.SetIcon(icon)
	handleError(err)

	// 点击托盘左键显示/隐藏窗口
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

	app.Run()
}
