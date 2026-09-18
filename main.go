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

	// ================= 核心防御机制 =================
	// 自己掌控命运：定义真实退出标志，完全不信任 walk 的 CloseReason
	var isExiting bool

	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		// 只要 isExiting 不是 true，任何人、任何操作点 X 都无法关掉它
		if !isExiting {
			*canceled = true // 强制拦截关闭指令
			mw.Hide()        // 瞬间隐藏面板
		}
	})
	// ===============================================

	// 启动时默认隐藏窗口，只留托盘
	mw.Hide()

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
				win.ShowWindow(mw.Handle(), win.SW_RESTORE)
				win.SetForegroundWindow(mw.Handle())
			}
		}
	})

	// 右键菜单：真正退出程序
	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		// 1. 改变标志位，给 Closing 拦截器放行
		isExiting = true 
		
		mw.Close() 
	})
	ni.ContextMenu().Actions().Add(exitAction)

	ni.SetVisible(true)

	// 原版的灵魂：由 MainWindow 维持系统消息循环
	mw.Run()
}
