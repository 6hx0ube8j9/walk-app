package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

func main() {
	var mw *walk.MainWindow

	// 1. 创建主窗口（默认自带最大化、最小化和关闭按钮）
	if err := (MainWindow{
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
	}).Create(); err != nil {
		panic(err)
	}

	// 2. 拦截右上角 X 或 Alt+F4 关闭事件，改为隐藏而不是销毁
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true // 取消默认退出
		mw.Hide()         // 仅隐藏窗口，托盘和后台保持存活
	})

	// 启动时默认隐藏面板
	mw.Hide()

	// 3. 初始化系统托盘（注意这里无参数）
	ni, err := walk.NewNotifyIcon()
	if err != nil {
		panic(err)
	}
	defer ni.Dispose()

	ni.SetToolTip("点击打开面板")
	ni.SetIcon(walk.IconInformation())

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
		walk.App().Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)
	ni.SetVisible(true)

	// 4. 运行主消息循环
	walk.App().Run()
}
