package ui

import (
	"myapp/internal/core"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

func CreateMainWindow(svc *core.Service, title string) (*walk.MainWindow, error) {
	var mw *walk.MainWindow

	err := MainWindow{
		AssignTo: &mw,
		Title:    title,
		MinSize:  Size{Width: 340, Height: 220},
		Layout:   VBox{Margins: Margins{Top: 15, Bottom: 15, Left: 15, Right: 15}, Spacing: 10},
		Children: []Widget{
			Label{Text: "程序常驻后台运行。点击右上角 X 隐藏至托盘。"},
			VSpacer{Size: 5},
			PushButton{
				Text:      "同步数据 (触发报错)",
				OnClicked: func() { _ = svc.SyncData() },
			},
			PushButton{
				Text:      "导出报表 (触发报错)",
				OnClicked: func() { _ = svc.ExportReport() },
			},
			PushButton{
				Text:      "切换模拟网络状态 (开关)",
				OnClicked: func() { _ = svc.ToggleNetwork() },
			},
		},
	}.Create()

	return mw, err
}
