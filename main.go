package main

import (
	"os"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

// 捕获错误并弹窗提示，防止程序静默闪退
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

	// 创建托盘
	ni, err = walk.NewNotifyIcon()
	handleError(err)
	defer ni.Dispose()

	err = ni.SetToolTip("点击打开简易程序")
	handleError(err)

	// 使用正确的标准信息图标
	icon, err := walk.IconInformation()
	handleError(err)
	err = ni.SetIcon(icon)
	handleError(err)

	// 点击托盘左键显示窗口
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			mw.Show()
			mw.SetFocus()
		}
	})

	// 右键菜单：退出
	exitAction := walk.NewAction()
	exitAction.SetText("退出")
	exitAction.Triggered().Attach(func() {
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)

	err = ni.SetVisible(true)
	handleError(err)

	// 运行主循环
	app.Run()
}
