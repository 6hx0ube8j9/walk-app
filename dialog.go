package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

// ShowErrorDialog 弹出一个独立的错误提示 GUI 窗口
func ShowErrorDialog(owner walk.Form, title, message string) {
	var dlg *walk.Dialog
	var acceptPB *walk.PushButton

	// 如果主窗口处于隐藏状态，避免绑定 owner 导致弹窗无法前置
	var parent walk.Form
	if owner != nil && owner.Visible() {
		parent = owner
	}

	err := Dialog{
		AssignTo:      &dlg,
		Title:         title,
		MinSize:       Size{Width: 340, Height: 160},
		Layout:        VBox{Margins: Margins{Top: 20, Bottom: 15, Left: 20, Right: 20}, Spacing: 15},
		DefaultButton: &acceptPB,
		Children: []Widget{
			Label{
				Text: message,
			},
			VSpacer{},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "确定",
						MinSize:  Size{Width: 80, Height: 28},
						OnClicked: func() {
							dlg.Accept()
						},
					},
				},
			},
		},
	}.Create(parent)

	if err != nil {
		return
	}

	// 强制置顶激活，防止被其他桌面窗口遮挡
	win.SetForegroundWindow(dlg.Handle())
	dlg.Run()
}
