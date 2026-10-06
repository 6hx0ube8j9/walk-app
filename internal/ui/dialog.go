package ui

import (
	"unsafe"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

const spiGetWorkArea = 0x0030

// centerDialog 居中并置顶
func centerDialog(dlg *walk.Dialog, parent walk.Form) {
	if parent != nil {
		return
	}

	var rect win.RECT
	win.GetWindowRect(dlg.Handle(), &rect)
	dlgW := rect.Right - rect.Left
	dlgH := rect.Bottom - rect.Top

	var workArea win.RECT
	var screenW, screenH int32
	if win.SystemParametersInfo(spiGetWorkArea, 0, unsafe.Pointer(&workArea), 0) {
		screenW = workArea.Right - workArea.Left
		screenH = workArea.Bottom - workArea.Top
	} else {
		screenW = win.GetSystemMetrics(win.SM_CXSCREEN)
		screenH = win.GetSystemMetrics(win.SM_CYSCREEN)
	}

	x := workArea.Left + (screenW-dlgW)/2
	y := workArea.Top + (screenH-dlgH)/2
	win.SetWindowPos(dlg.Handle(), win.HWND_TOP, x, y, 0, 0, win.SWP_NOSIZE)
	win.SetForegroundWindow(dlg.Handle())
}

// ShowErrorDialog 错误提示弹窗
func ShowErrorDialog(owner walk.Form, title, message string) {
	RunAlertDialog(owner, title, message, walk.IconError(), win.MB_ICONERROR)
}

// ShowConfirmDialog 确认提示弹窗
func ShowConfirmDialog(owner walk.Form, title, message string) bool {
	return RunQuestionDialog(owner, title, message, walk.IconQuestion(), win.MB_ICONASTERISK)
}

// RunErrorDialog 别名兼容
func RunErrorDialog(owner walk.Form, title, message string) {
	ShowErrorDialog(owner, title, message)
}

// RunConfirmDialog 别名兼容
func RunConfirmDialog(owner walk.Form, title, message string) bool {
	return ShowConfirmDialog(owner, title, message)
}

// RunAlertDialog 自定义单按钮信息/错误弹窗（100% 纯原生）
func RunAlertDialog(owner walk.Form, title, message string, icon *walk.Icon, soundStyle uint32) {
	var dlg *walk.Dialog
	var acceptPB *walk.PushButton

	var parent walk.Form
	if owner != nil && owner.Visible() {
		parent = owner
	}

	err := Dialog{
		AssignTo:      &dlg,
		Title:         title,
		MinSize:       Size{Width: 320, Height: 150},
		Layout:        VBox{Margins: Margins{Top: 15, Bottom: 15, Left: 15, Right: 15}, Spacing: 10},
		DefaultButton: &acceptPB, // 原生绑定：Enter 触发确定
		CancelButton:  &acceptPB, // 原生绑定：Esc 触发确定关闭
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true, Spacing: 12},
				Children: []Widget{
					ImageView{Image: icon, Margin: 0},
					Label{Text: message},
				},
			},
			VSpacer{},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo:  &acceptPB,
						Text:      "确定",
						MinSize:   Size{Width: 70, Height: 26},
						OnClicked: func() { dlg.Accept() },
					},
				},
			},
		},
	}.Create(parent)

	if err != nil {
		return
	}

	dlg.Starting().Attach(func() {
		win.MessageBeep(soundStyle)
		centerDialog(dlg, parent)
	})

	dlg.Run()
}

// RunQuestionDialog 自定义双按钮询问弹窗（100% 纯原生）
func RunQuestionDialog(owner walk.Form, title, message string, icon *walk.Icon, soundStyle uint32) bool {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	confirmed := false

	var parent walk.Form
	if owner != nil && owner.Visible() {
		parent = owner
	}

	err := Dialog{
		AssignTo:      &dlg,
		Title:         title,
		MinSize:       Size{Width: 320, Height
