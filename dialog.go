package main

import (
	"unsafe"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

// ShowErrorDialog 弹出一个错误提示 GUI 窗口
// 当主面板显示时以面板为中心，当主面板隐藏/不存在时以屏幕工作区为中心
func ShowErrorDialog(owner walk.Form, title, message string) {
	var dlg *walk.Dialog
	var acceptPB *walk.PushButton

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

	// 挂载到 Starting 事件：在布局完成但窗口尚未绘制出来的瞬间计算并修正位置
	dlg.Starting().Attach(func() {
		if parent == nil {
			// 获取弹窗自身的真实宽高
			var rect win.RECT
			win.GetWindowRect(dlg.Handle(), &rect)
			dlgW := rect.Right - rect.Left
			dlgH := rect.Bottom - rect.Top

			// 获取屏幕工作区尺寸（自动避开任务栏占据的区域）
			var workArea win.RECT
			win.SystemParametersInfo(win.SPI_GETWORKAREA, 0, unsafe.Pointer(&workArea), 0)
			workW := workArea.Right - workArea.Left
			workH := workArea.Bottom - workArea.Top

			x := workArea.Left + (workW-dlgW)/2
			y := workArea.Top + (workH-dlgH)/2

			// 移动窗口并置顶
			win.SetWindowPos(dlg.Handle(), win.HWND_TOP, x, y, 0, 0, win.SWP_NOSIZE)
			win.SetForegroundWindow(dlg.Handle())
		}
	})

	dlg.Run()
}
