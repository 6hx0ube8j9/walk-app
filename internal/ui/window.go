package ui

import (
	"walk-app/internal/types"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
)

type MainWindowView struct {
	Window      *walk.MainWindow
	StatusLabel *walk.Label
}

func CreateMainWindow(cmdCh chan<- types.UICommand, title string) (*MainWindowView, error) {
	view := &MainWindowView{}

	err := MainWindow{
		AssignTo: &view.Window,
		Title:    title,
		MinSize:  Size{Width: 340, Height: 220},
		Layout:   VBox{Margins: Margins{Top: 15, Bottom: 15, Left: 15, Right: 15}, Spacing: 10},
		Children: []Widget{
			Label{
				AssignTo: &view.StatusLabel,
				Text:     "初始化中...",
			},
			VSpacer{Size: 5},
			PushButton{
				Text: "同步数据 (触发报错)",
				OnClicked: func() {
					cmdCh <- types.UICommand{Action: "sync_data"}
				},
			},
			PushButton{
				Text: "导出报表 (触发报错)",
				OnClicked: func() {
					cmdCh <- types.UICommand{Action: "export_report"}
				},
			},
			PushButton{
				Text: "切换模拟网络状态 (开关)",
				OnClicked: func() {
					cmdCh <- types.UICommand{Action: "toggle_network"}
				},
			},
		},
	}.Create()

	return view, err
}
