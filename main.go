package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

//go:generate go build -ldflags="-H windowsgui" -o app_b.exe main.go

func main() {
	var mw *walk.MainWindow

	err := MainWindow{
		AssignTo: &mw,
		Title:    "流派B - 原版 Walk 单进程",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "点击右上角 X 会自动隐藏到托盘，绝不会退出"},
		},
	}.Create()
	
	if err != nil {
		return
	}

	// ---------------- 原版 Walk 的标准拦截 ----------------
	// 在 lxn/walk 中，拦截非常可靠，不需要乱七八糟的异步处理
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		// CloseReasonUser 代表用户点了 X 或者按了 Alt+F4
		if reason == walk.CloseReasonUser {
			*canceled = true // 拦截系统销毁窗口的指令
			mw.Hide()        // 直接隐藏
		}
	})
	// ----------------------------------------------------

	// 启动时默认隐藏窗口，只留托盘
	mw.Hide()

	// 核心差异：原版 walk 创建托盘必须绑定到一个窗口句柄 (mw)
	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("原版 Walk 托盘常驻")
	ni.SetIcon(walk.IconInformation())

	// 左键点击托盘：切换面板显隐状态
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if mw.Visible() {
				mw.Hide()
			} else {
				mw.Show()
				// 原版也需要借助 win API 突破 Windows 前台焦点限制
				win.ShowWindow(mw.Handle(), win.SW_RESTORE)
				win.SetForegroundWindow(mw.Handle())
			}
		}
	})

	// 右键菜单：真正退出程序
	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		// 发送全局退出信号，这会打破 mw.Run() 的阻塞
		walk.App().Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)
	
	ni.SetVisible(true)

	// 核心差异：原版 walk 使用主窗口的 Run() 维持系统消息循环
	// 只要 mw 没被销毁，这个循环就不会停
	mw.Run()
}
