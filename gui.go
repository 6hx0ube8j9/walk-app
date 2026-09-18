package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

// CreateMainWindow 仅负责面板控件的声明与布局
func CreateMainWindow(app *App, title string) (*walk.MainWindow, error) {
	var mw *walk.MainWindow

	err := MainWindow{
		AssignTo: &mw,
		Title:    title,
		MinSize:  Size{Width: 360, Height: 260},
		Layout:   VBox{Margins: Margins{Top: 20, Bottom: 20, Left: 20, Right: 20}, Spacing: 10},
		Children: []Widget{
			Label{Text: "程序已常驻运行。点击右上角 X 会直接隐藏到托盘。"},
			VSpacer{Size: 10},
			PushButton{
				Text: "同步数据 (模拟报错)",
				OnClicked: func() {
					app.HandleOperation("sync_data")
				},
			},
			PushButton{
				Text: "导出报表 (模拟报错)",
				OnClicked: func() {
					app.HandleOperation("export_report")
				},
			},
			PushButton{
				Text: "切换模拟网络状态 (开关)",
				OnClicked: func() {
					app.HandleOperation("toggle_network")
				},
			},
		},
	}.Create()

	return mw, err
}
