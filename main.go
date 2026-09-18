package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

//go:generate go build -ldflags="-s -w" -o app_b.exe main.go

func main() {
	app, err := walk.InitApp()
	if err != nil {
		return
	}
	defer app.Exit(0)

	var anchor *walk.MainWindow
	err = MainWindow{
		AssignTo: &anchor,
		Title:    "Hidden Anchor",
		Visible:  false, // 永远不要调用它的 Show()
	}.Create()
	if err != nil {
		return
	}

	var mw *walk.MainWindow
	err = MainWindow{
		AssignTo: &mw,
		Title:    "Mihomo Tray 面板",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "拥有最大化/最小化，点击 X 隐藏到托盘不死！"},
		},
	}.Create() // 注意：这里不需要传 anchor 作为父级，让它独立
	if err != nil {
		return
	}

	var isExiting bool

	// 3. 拦截真实界面的关闭事件
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !isExiting {
			*canceled = true 
			mw.Hide() // 安全隐藏。由于 anchor 还在运行，程序绝对不会退出
		}
	})

	mw.Hide()

	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("Mihomo Tray 稳定版")
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
		ni.Dispose()
		app.Exit(0) // 只有通过托盘菜单退出时，才真正终结程序
	})
	ni.ContextMenu().Actions().Add(exitAction)
	ni.SetVisible(true)

	// 开始运行，底层生命周期被 anchor 牢牢锁死
	app.Run()
}
