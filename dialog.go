package main

import (
	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

// ShowErrorDialog 弹出一个默认在屏幕正中央的错误提示 GUI 窗口
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

	// 计算屏幕居中坐标
	var rect win.RECT
	win.GetWindowRect(dlg.Handle(), &rect)
	dlgWidth := rect.Right - rect.Left
	dlgHeight := rect.Bottom - rect.Top

	screenWidth := win.GetSystemMetrics(win.SM_CXSCREEN)
	screenHeight := win.GetSystemMetrics(win.SM_CYSCREEN)

	x := (screenWidth - dlgWidth) / 2
	y := (screenHeight - dlgHeight) / 2

	// 移动窗口到居中坐标（保持原有尺寸与 Z-Order 不变）
	win.SetWindowPos(dlg.Handle(), 0, x, y, 0, 0, win.SWP_NOSIZE|win.SWP_NOZORDER)

	// 强制置顶激活
	win.SetForegroundWindow(dlg.Handle())
	dlg.Run()
}
