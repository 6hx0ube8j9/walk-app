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
		AssignTo:     &dlg,
		Title:        title,
		MinSize:      Size{Width: 320, Height: 150},
		Layout:       VBox{Margins: Margins{Top: 15, Bottom: 15, Left: 15, Right: 15}, Spacing: 10},
		CancelButton: &acceptPB, // ESC 原生关闭
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

	dlg.Activating().Attach(func() {
		acceptPB.SetFocus()
	})

	// 注册全局回车加速键
	enterAction := walk.NewAction()
	enterAction.SetShortcut(walk.Shortcut{Key: walk.KeyReturn})
	enterAction.Triggered().Attach(doClose)
	dlg.Actions().Add(enterAction)

	dlg.Run()
}

// RunQuestionDialog 自定义双按钮询问弹窗
func RunQuestionDialog(owner walk.Form, title, message string, icon *walk.Icon, soundStyle uint32) bool {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	confirmed := false
	closed := false

	var parent walk.Form
	if owner != nil && owner.Visible() {
		parent = owner
	}

	doAccept := func() {
		if closed {
			return
		}
		closed = true
		confirmed = true
		if dlg != nil {
			dlg.Accept()
		}
	}

	doCancel := func() {
		if closed {
			return
		}
		closed = true
		confirmed = false
		if dlg != nil {
			dlg.Cancel()
		}
	}

	// 统一回车分发：依据底层物理焦点精准判断
	handleEnter := func() {
		if cancelPB != nil && (cancelPB.Focused() || win.GetFocus() == cancelPB.Handle()) {
			doCancel()
		} else {
			doAccept()
		}
	}

	err := Dialog{
		AssignTo: &dlg,
		Title:    title,
		MinSize:  Size{Width: 320, Height: 150},
		Layout:   VBox{Margins: Margins{Top: 15, Bottom: 15, Left: 15, Right: 15}, Spacing: 10},
		// 严禁在此处设置 DefaultButton，防止 Walk 外层消息循环无条件劫持回车
		CancelButton: &cancelPB, // ESC 原生绑定“否”
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

	// 1. 展现时将焦点默认赋予“是”按钮
	dlg.Activating().Attach(func() {
		acceptPB.SetFocus()
	})

	// 2. 核心：通过 Walk Action 注册全局回车加速键（无论焦点在哪个子控件都能100%捕获）
	enterAction := walk.NewAction()
	enterAction.SetShortcut(walk.Shortcut{Key: walk.KeyReturn})
	enterAction.Triggered().Attach(handleEnter)
	dlg.Actions().Add(enterAction)

	// 3. 多通道事件兜底
	acceptPB.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			handleEnter()
		}
	})
	cancelPB.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			handleEnter()
		}
	})
	dlg.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			handleEnter()
		} else if key == walk.KeyEscape {
			doCancel()
		}
	})

	dlg.Run()
	return confirmed
}
