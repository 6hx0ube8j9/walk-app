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

	var root *walk.MainWindow
	err = MainWindow{
		AssignTo: &root,
		Title:    "Hidden Root",
		Visible:  false, // 核心：永远不要调用 root.Show()
	}.Create()
	if err != nil {
		return
	}

	var ui *walk.Dialog
	err = Dialog{
		AssignTo: &ui,
		Title:    "Mihomo Tray 面板",
		MinSize:  Size{Width: 300, Height: 200},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "这次随便点右上角的 X，托盘绝对死不了！"},
		},
	}.Create(root) // 将隐藏的 root 作为它的父窗口
	if err != nil {
		return
	}

	// 3. 拦截 Dialog 的关闭事件，改为单纯的隐藏
	ui.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		ui.Hide() // Dialog 的隐藏非常安全，再也不会牵连整个程序了
	})

	ni, err := walk.NewNotifyIcon()
	if err != nil {
		return
	}
	defer ni.Dispose()

	ni.SetToolTip("Mihomo Tray 稳定版")
	ni.SetIcon(walk.IconInformation())

	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if ui.Visible() {
				ui.Hide()
			} else {
				ui.Show()
				win.ShowWindow(ui.Handle(), win.SW_RESTORE)
				win.SetForegroundWindow(ui.Handle())
			}
		}
	})

	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		ni.Dispose()
		app.Exit(0) // 只有这里才会真正终结程序
	})
	ni.ContextMenu().Actions().Add(exitAction)
	ni.SetVisible(true)

	// 运行主循环，它会被隐藏的 root 窗口永远维持住
	app.Run()
}
