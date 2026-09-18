package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

//go:generate go build -ldflags="-H windowsgui" -o app_b.exe main.go

func main() {
	var mw *walk.MainWindow

	err := MainWindow{
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

	// 修复 1：tailscale/walk 的 NewNotifyIcon 无需传参
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
		// 退出时向当前线程发送退出消息，终结 Win32 消息循环
		win.PostQuitMessage(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)

	ni.SetVisible(true)

	// 修复 2：替代 mw.Run()，采用标准的 Win32 消息循环
	var msg win.MSG
	for win.GetMessage(&msg, 0, 0, 0) > 0 {
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
}
