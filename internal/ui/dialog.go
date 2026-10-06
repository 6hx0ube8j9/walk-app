package ui

import (
	"unsafe"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

const spiGetWorkArea = 0x0030

// centerDialog 避免与 window.go 的 centerWindow 重名；当无可视父窗口时自动居中并置顶
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

// ShowErrorDialog 错误提示弹窗（供 presenter.go 和 tray.go 调用，播放错误音）
func ShowErrorDialog(owner walk.Form, title, message string) {
	RunAlertDialog(owner, title, message, walk.IconError(), win.MB_ICONERROR)
}

// ShowConfirmDialog 确认提示弹窗（供 tray.go 调用，播放提示音，返回是否点击“是”）
func ShowConfirmDialog(owner walk.Form, title, message string) bool {
	// 使用 MB_ICONASTERISK 触发提示音（MB_ICONQUESTION 在 Win10/11 默认静音）
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

// RunAlertDialog 自定义单按钮信息/错误弹窗
func RunAlertDialog(owner walk.Form, title, message string, icon *walk.Icon, soundStyle uint32) {
	var dlg *walk.Dialog
	var acceptPB *walk.PushButton

	var parent walk.Form
	if owner != nil && owner.Visible() {
		parent = owner
	}

	doClose := func() {
		if dlg != nil {
			dlg.Accept()
		}
	}

	err := Dialog{
		AssignTo:      &dlg,
		Title:         title,
		MinSize:       Size{Width: 320, Height: 150},
		Layout:        VBox{Margins: Margins{Top: 15, Bottom: 15, Left: 15, Right: 15}, Spacing: 10},
		DefaultButton: &acceptPB,
		CancelButton:  &acceptPB,
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
						OnClicked: doClose,
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

	// 弹窗激活时将焦点给到“确定”按钮
	dlg.Activating().Attach(func() {
		acceptPB.SetFocus()
	})

	// Enter / Esc 均可直接关闭提示弹窗
	dlg.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn || key == walk.KeyEscape {
			doClose()
		}
	})

	dlg.Run()
}

// RunQuestionDialog 自定义双按钮询问弹窗
func RunQuestionDialog(owner walk.Form, title, message string, icon *walk.Icon, soundStyle uint32) bool {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	confirmed := false

	var parent walk.Form
	if owner != nil && owner.Visible() {
		parent = owner
	}

	doAccept := func() {
		confirmed = true
		if dlg != nil {
			dlg.Accept()
		}
	}

	doCancel := func() {
		confirmed = false
		if dlg != nil {
			dlg.Cancel()
		}
	}

	err := Dialog{
		AssignTo:      &dlg,
		Title:         title,
		MinSize:       Size{Width: 320, Height: 150},
		Layout:        VBox{Margins: Margins{Top: 15, Bottom: 15, Left: 15, Right: 15}, Spacing: 10},
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
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
				Layout: HBox{MarginsZero: true, Spacing: 10},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo:  &acceptPB,
						Text:      "是",
						MinSize:   Size{Width: 70, Height: 26},
						OnClicked: doAccept,
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "否",
						MinSize:   Size{Width: 70, Height: 26},
						OnClicked: doCancel,
					},
				},
			},
		},
	}.Create(parent)

	if err != nil {
		return false
	}

	dlg.Starting().Attach(func() {
		win.MessageBeep(soundStyle)
		centerDialog(dlg, parent)
	})

	// 1. 展现时将键盘焦点落在默认的“是”按钮上
	dlg.Activating().Attach(func() {
		acceptPB.SetFocus()
	})

	// 2. 焦点在“否”上时按 Enter，执行取消逻辑
	cancelPB.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			doCancel()
		}
	})

	// 3. 弹窗全局按键分发：感知焦点所在位置
	dlg.KeyDown().Attach(func(key walk.Key) {
		switch key {
		case walk.KeyReturn:
			if cancelPB.Focused() {
				doCancel()
			} else {
				doAccept()
			}
		case walk.KeyEscape:
			doCancel()
		}
	})

	dlg.Run()
	return confirmed
}
