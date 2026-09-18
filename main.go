package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

//go:generate go build -ldflags="-H windowsgui" -o app_b.exe main.go

func main() {
	// 1. tailscale/walk 必须最先显式初始化 App 单例
	app, err := walk.InitApp()
	if err != nil {
		return
	}

	var mw *walk.MainWindow

	err = MainWindow{
		AssignTo: &mw,
		Title:    "Walk 单进程",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "点击右上角 X 会自动隐藏到托盘，绝不会退出"},
		},
	}.Create()

	if err != nil {
		return
	}

	var isExiting bool

	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !isExiting {
			*canceled = true // 强制拦截关闭指令
			mw.Hide()        // 瞬间隐藏面板
		}
	})

	// 启动时默认隐藏窗口，只留托盘
	mw.Hide()

	// 2. tailscale/walk 的 NotifyIcon 无需传参
	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("Tailscale Walk 托盘常驻")
	ni.SetIcon(walk.IconInformation())

	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if mw.Visible() {
				mw.Hide()
			} else {
				mw.Show()
				win.ShowWindow(mw.Handle(), win.SW_RESTORE)
				win.SetForegroundWindow(mw.Handle())
			}
		}
	})

	// 右键菜单：真正退出程序
	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		isExiting = true
		mw.Close()
		app.Exit(0) // 退出全局消息循环
	})
	ni.ContextMenu().Actions().Add(exitAction)

	ni.SetVisible(true)

	// 3. 由 app.Run() 取代原先的 mw.Run()
	app.Run()
}
