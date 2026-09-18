package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

//go:generate go build -ldflags="-H windowsgui" -o app_b.exe main.go

func main() {
	// 1. Tailscale 分支特有的 App 初始化
	app, err := walk.InitApp()
	if err != nil {
		return
	}
	defer app.Exit(0)

	var mw *walk.MainWindow

	err = MainWindow{
		AssignTo: &mw,
		Title:    "流派B - Tailscale Walk",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "点击右上角 X 会自动隐藏到托盘，绝不会退出"},
		},
	}.Create()

	if err != nil {
		return
	}

	// ================= 核心防御机制 =================
	var isExiting bool

	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !isExiting {
			*canceled = true // 强制拦截关闭指令
			
			// 【致命细节】Tailscale 分支必须使用 Synchronize 将 UI 操作推迟到下一个安全周期
			mw.Synchronize(func() {
				mw.Hide()
			})
		}
	})
	// ===============================================

	mw.Hide()

	// 2. Tailscale 分支的托盘不需要绑定主窗口句柄
	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("Tailscale Walk 托盘")
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

	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		isExiting = true 
		ni.Dispose() // 提前销毁托盘，防止残留图标
		
		// 3. Tailscale 分支通过主动退出 App 来终结生命周期
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)

	ni.SetVisible(true)

	// Tailscale 分支通过 app.Run() 维持消息循环
	app.Run()
}
