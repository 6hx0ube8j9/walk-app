package ui

import (
	"unsafe"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

const spiGetWorkArea = 0x0030

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

func ShowErrorDialog(owner walk.Form, title, message string) {
	RunAlertDialog(owner, title, message, walk.IconError(), win.MB_ICONERROR)
}

func ShowConfirmDialog(owner walk.Form, title, message string) bool {
	return RunQuestionDialog(owner, title, message, walk.IconQuestion(), win.MB_ICONASTERISK)
}

func RunErrorDialog(owner walk.Form, title, message string) {
	ShowErrorDialog(owner, title, message)
}

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

	dlg.Activating().Attach(func() {
		acceptPB.SetFocus()
	})

	dlg.Run()
}

// RunQuestionDialog 自定义双按钮询问弹窗（焦点状态追踪版）
func RunQuestionDialog(owner walk.Form, title, message string, icon *walk.Icon, soundStyle uint32) bool {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	confirmed := false

	// activeFocusedPB 实时记录用户通过键盘或鼠标赋予焦点的目标控件
	var activeFocusedPB *walk.PushButton

	var parent walk.Form
	if owner != nil && owner.Visible() {
		parent = owner
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
						AssignTo: &acceptPB,
						Text:     "是",
						MinSize:  Size{Width: 70, Height: 26},
						OnClicked: func() {
							// 核心逻辑：若最后一次键盘焦点记录在“否”按钮上，按 Enter 强制执行取消
							if activeFocusedPB == cancelPB {
								confirmed = false
								dlg.Cancel()
								return
							}
							confirmed = true
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo: &cancelPB,
						Text:     "否",
						MinSize:  Size{Width: 70, Height: 26},
						OnClicked: func() {
							confirmed = false
							dlg.Cancel()
						},
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

	// 1. 弹窗显示时，将初始焦点赋予“是”并初始化追踪状态
	dlg.Activating().Attach(func() {
		acceptPB.SetFocus()
		activeFocusedPB = acceptPB
	})

	// 2. 实时追踪焦点变动：在用户按 Tab 键移动焦点时立即记录，不受后续点击事件抢焦影响
	acceptPB.FocusedChanged().Attach(func() {
		if acceptPB.Focused() {
			activeFocusedPB = acceptPB
		}
	})
	cancelPB.FocusedChanged().Attach(func() {
		if cancelPB.Focused() {
			activeFocusedPB = cancelPB
		}
	})

	dlg.Run()
	return confirmed
}
