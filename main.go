package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)


func main() {
	app, err := walk.InitApp()
	if err != nil {
		return
	}
	defer app.Exit(0)

	var mw *walk.MainWindow
	err = MainWindow{
		AssignTo: &mw,
		Title:    "流派B - 降维打击防闪退",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "点击右上角 X 会安全隐藏，绝对不崩溃"},
		},
	}.Create()

	if err != nil {
		return
	}

	var isExiting bool

	// ---------------- 终极防御机制 ----------------
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !isExiting {
			*canceled = true 
			
			win.ShowWindow(mw.Handle(), win.SW_HIDE)
		}
	})
	// ----------------------------------------------

	// 启动时也用底层 API 隐藏
	win.ShowWindow(mw.Handle(), win.SW_HIDE)

	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("Mihomo Tray 测试")
	ni.SetIcon(walk.IconInformation())

	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			// 判断可见性同样使用底层 API
			if win.IsWindowVisible(mw.Handle()) {
				win.ShowWindow(mw.Handle(), win.SW_HIDE)
			} else {
				win.ShowWindow(mw.Handle(), win.SW_SHOW)
				win.ShowWindow(mw.Handle(), win.SW_RESTORE)
				win.SetForegroundWindow(mw.Handle())
			}
		}
	})

	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		isExiting = true 
		ni.Dispose()
		app.Exit(0)
	})
	ni.ContextMenu().Actions().Add(exitAction)
	ni.SetVisible(true)

	app.Run()
}
