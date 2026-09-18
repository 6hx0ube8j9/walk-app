package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

//go:generate go build -ldflags="-H windowsgui" -o app_b.exe main.go

func main() {
	app, err := walk.InitApp()
	if err != nil {
		return
	}
	defer app.Exit(0)

	var mw *walk.MainWindow
	err = MainWindow{
		AssignTo: &mw,
		Title:    "流派B - 单进程显隐面板",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "点击右上角 X 会自动隐藏到托盘，而不是退出程序"},
		},
	}.Create()

	if err != nil {
		return
	}

	// 核心修复：在 Create 之后，通过 Attach 绑定事件。
	// 同时保留必需的 CloseReasonUnknown 判断和 Synchronize 异步隐藏。
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		// 只有用户手动点击 X (CloseReasonUnknown) 时才拦截
		if reason == walk.CloseReasonUnknown {
			*canceled = true
			// 必须放在 Synchronize 中，防止阻塞底层关闭消息导致崩溃
			mw.Synchronize(func() {
				mw.Hide()
			})
		}
	})

	// 启动时默认隐藏窗口，只留托盘
	mw.Hide()

	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("流派B - 单进程常驻")
	ni.SetIcon(walk.IconInformation())

	// 左键点击托盘：切换面板显隐状态
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
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)
	ni.SetVisible(true)

	app.Run()
}
